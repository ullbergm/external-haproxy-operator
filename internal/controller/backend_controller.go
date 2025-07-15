/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"reflect"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/go-logr/logr"
	externalhaproxyoperatorv1alpha1 "github.com/ullbergm/external-haproxy-operator/api/v1alpha1"
	"github.com/ullbergm/external-haproxy-operator/internal/haproxyclient"
	"github.com/ullbergm/external-haproxy-operator/monitoring"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// BackendReconciler reconciles a Backend object
type BackendReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	Recorder      record.EventRecorder
	HAProxyClient haproxyclient.HAProxyClient
}

func (r *BackendReconciler) setCondition(
	backend *externalhaproxyoperatorv1alpha1.Backend,
	condition metav1.Condition,
) {
	meta.SetStatusCondition(&backend.Status.Conditions, condition)
}

const backendFinalizer = "external-haproxy-operator.ullberg.us/finalizer"

// +kubebuilder:rbac:groups=external-haproxy-operator.ullberg.us,resources=backends,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=external-haproxy-operator.ullberg.us,resources=backends/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=external-haproxy-operator.ullberg.us,resources=backends/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=discoveryv1,resources=endpointslices,verbs=get;list;watch
// +kubebuilder:rbac:groups=corev1,resources=services,verbs=get;list;watch
// +kubebuilder:rbac:groups=corev1,resources=nodes,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.

// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *BackendReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := logf.FromContext(ctx)

	// Fetch the Backend instance
	backend := &externalhaproxyoperatorv1alpha1.Backend{}
	err := r.Get(ctx, req.NamespacedName, backend)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			reqLogger.Info("Backend resource not found. Ignoring since object must be deleted.")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		reqLogger.Error(err, "Failed to get Backend.")
		return ctrl.Result{}, err
	}

	reqLogger.V(1).Info("Reconciling Backend")

	// Loop through the backend's servers and validate them
	if err := externalhaproxyoperatorv1alpha1.ValidateServers(backend.Spec.Servers); err != nil {
		// Emit an event for the validation error
		r.Recorder.Event(backend, "Warning", "ValidationError", err.Error())
		reqLogger.Error(err, "Validation failed for Backend servers", "name", backend.Name)
		// Set Reconciling Condition
		r.setCondition(backend, metav1.Condition{
			Type:    "ReconcilingComplete",
			Status:  metav1.ConditionFalse,
			Reason:  "ValidationError",
			Message: "Failed to validate Backend servers: " + err.Error(),
		})
		_ = r.Status().Update(ctx, backend)
		return ctrl.Result{}, err
	}

	// For each of the backend's servers, if it is a dynamic server (using ValueFrom),
	// ensure that the referenced Kubernetes Service exists and is valid and get the endpoints where that service is running.
	// Create a copy of the backend variable and modify it to include the resolved servers.
	modifiedBackend := backend.DeepCopy()

	if err1 := resolveServerObjects(ctx, backend, reqLogger, r, modifiedBackend); err1 != nil {
		return ctrl.Result{}, err1
	}
	reqLogger.V(2).Info("Object processed", "before", backend, "after", modifiedBackend)

	// Start a transaction in HAProxy
	transaction, err := r.HAProxyClient.StartTransaction()
	if err != nil {
		// Increment the error counter metric
		monitoring.HAProxyClientErrorCountTotal.Inc()
		reqLogger.Error(err, "Failed to start HAProxy transaction")
		// Set Reconciling Condition
		r.setCondition(backend, metav1.Condition{
			Type:    "ReconcilingComplete",
			Status:  metav1.ConditionFalse,
			Reason:  "HAProxyClientError",
			Message: "Failed to start HAProxy transaction: " + err.Error(),
		})
		_ = r.Status().Update(ctx, backend)
		return ctrl.Result{}, err
	}

	// Create a backend in HAProxy if it does not exist
	if err := r.HAProxyClient.EnsureBackend(externalhaproxyoperatorv1alpha1.BackendSpecToModel(modifiedBackend.Spec)); err != nil {
		// Increment the error counter metric
		monitoring.HAProxyClientErrorCountTotal.Inc()
		reqLogger.Error(err, "Failed to create backend in HAProxy", "name", backend.Name)
		// Set Reconciling Condition
		r.setCondition(backend, metav1.Condition{
			Type:    "ReconcilingComplete",
			Status:  metav1.ConditionFalse,
			Reason:  "HAProxyClientError",
			Message: "Failed to create backend in HAProxy: " + err.Error(),
		})
		_ = r.Status().Update(ctx, backend)
		return ctrl.Result{}, err
	}

	// Commit the transaction in HAProxy
	if _, err := r.HAProxyClient.CommitTransaction(transaction.ID, false); err != nil {
		// Increment the error counter metric
		monitoring.HAProxyClientErrorCountTotal.Inc()
		reqLogger.Error(err, "Failed to commit HAProxy transaction", "transactionID", transaction.ID)
		// Set Reconciling Condition
		r.setCondition(backend, metav1.Condition{
			Type:    "ReconcilingComplete",
			Status:  metav1.ConditionFalse,
			Reason:  "HAProxyClientError",
			Message: "Failed to commit HAProxy transaction: " + err.Error(),
		})
		_ = r.Status().Update(ctx, backend)
		return ctrl.Result{}, err
	}

	// Check if the Backend instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	isBackendMarkedToBeDeleted := backend.GetDeletionTimestamp() != nil
	if isBackendMarkedToBeDeleted {
		if controllerutil.ContainsFinalizer(backend, backendFinalizer) {
			// Run finalization logic for backendFinalizer. If the
			// finalization logic fails, don't remove the finalizer so
			// that we can retry during the next reconciliation.
			if err := r.finalizeBackend(reqLogger, backend); err != nil {
				return ctrl.Result{}, err
			}

			// Remove backendFinalizer. Once all finalizers have been
			// removed, the object will be deleted.
			controllerutil.RemoveFinalizer(backend, backendFinalizer)
			err := r.Update(ctx, backend)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer for this CR
	if !controllerutil.ContainsFinalizer(backend, backendFinalizer) {
		controllerutil.AddFinalizer(backend, backendFinalizer)
		err = r.Update(ctx, backend)
		if err != nil {
			// Set Reconciling Condition
			r.setCondition(backend, metav1.Condition{
				Type:    "ReconcilingComplete",
				Status:  metav1.ConditionFalse,
				Reason:  "AddFinalizerFailed",
				Message: "Failed to add finalizer to Backend",
			})
			_ = r.Status().Update(ctx, backend)
			return ctrl.Result{}, err
		}
	}

	// Set Reconciling Condition
	r.setCondition(backend, metav1.Condition{
		Type:    "ReconcilingComplete",
		Status:  metav1.ConditionTrue,
		Reason:  "ReconcileCompleted",
		Message: "Reconciliation completed successfully",
	})
	_ = r.Status().Update(ctx, backend)

	return ctrl.Result{}, nil
}

func resolveServerObjects(ctx context.Context, backend *externalhaproxyoperatorv1alpha1.Backend, reqLogger logr.Logger, r *BackendReconciler, modifiedBackend *externalhaproxyoperatorv1alpha1.Backend) error {
	for _, server := range backend.Spec.Servers {
		reqLogger.V(2).Info("Processing server", "object", server)
		if server.ValueFrom != nil && server.ValueFrom.ServiceRef != nil {
			serviceRef := server.ValueFrom.ServiceRef
			if serviceRef.Name == "" {
				err := errors.NewBadRequest("ServiceRef name must be specified for dynamic servers")
				r.Recorder.Event(backend, "Warning", "ValidationError", err.Error())
				reqLogger.Error(err, "Validation failed for Backend server")
				// Set Reconciling Condition
				r.setCondition(backend, metav1.Condition{
					Type:    "ReconcilingComplete",
					Status:  metav1.ConditionFalse,
					Reason:  "ValidationError",
					Message: "Failed to validate Backend server: " + err.Error(),
				})
				_ = r.Status().Update(ctx, backend)
				return err
			}

			// Fetch the referenced Service to ensure it exists
			service := &corev1.Service{}
			err := r.Get(ctx, client.ObjectKey{Namespace: serviceRef.Namespace, Name: serviceRef.Name}, service)
			if err != nil {
				if errors.IsNotFound(err) {
					err = errors.NewNotFound(corev1.Resource("Service"), serviceRef.Name)
				}
				r.Recorder.Event(backend, "Warning", "ServiceNotFound", err.Error())
				reqLogger.Error(err, "Referenced Service not found", "serviceName", serviceRef.Name)
				// Update the Backend status to reflect the error
				r.setCondition(backend, metav1.Condition{
					Type:    "ReconcilingComplete",
					Status:  metav1.ConditionFalse,
					Reason:  "ServiceNotFound",
					Message: "Referenced Service not found: " + serviceRef.Name,
				})
				_ = r.Status().Update(ctx, backend)
				return err
			}

			// Get the endpointslices for the service
			endpoints := &discoveryv1.EndpointSliceList{}
			err = r.List(ctx, endpoints, client.InNamespace(serviceRef.Namespace), client.MatchingLabels{"kubernetes.io/service-name": serviceRef.Name})
			if err != nil {
				r.Recorder.Event(backend, "Warning", "EndpointsListError", err.Error())
				reqLogger.Error(err, "Failed to list endpoints for Service", "serviceName", serviceRef.Name)

				// Set Reconciling Condition
				r.setCondition(backend, metav1.Condition{
					Type:    "ReconcilingComplete",
					Status:  metav1.ConditionFalse,
					Reason:  "EndpointsListError",
					Message: "Failed to list endpoints for Service: " + serviceRef.Name,
				})
				_ = r.Status().Update(ctx, backend)
				return err
			}
			if len(endpoints.Items) == 0 {
				err = errors.NewNotFound(discoveryv1.Resource("EndpointSlice"), serviceRef.Name)
				r.Recorder.Event(backend, "Warning", "EndpointsNotFound", err.Error())
				reqLogger.Error(err, "No endpoints found for Service", "serviceName", serviceRef.Name)
				// Set Reconciling Condition
				r.setCondition(backend, metav1.Condition{
					Type:    "ReconcilingComplete",
					Status:  metav1.ConditionFalse,
					Reason:  "EndpointsNotFound",
					Message: "Failed to find endpoints for Service: " + serviceRef.Name,
				})
				_ = r.Status().Update(ctx, backend)
				return err
			}
			// Add a server to the spec for each endpoint, skipping those where the pod is terminating, not running, or not ready
			for _, endpointSlice := range endpoints.Items {
				for _, endpoint := range endpointSlice.Endpoints {
					if len(endpoint.Addresses) == 0 {
						continue
					}
					shouldSkip := false
					if endpoint.TargetRef != nil && endpoint.TargetRef.Kind == "Pod" {
						pod := &corev1.Pod{}
						err := r.Get(ctx, client.ObjectKey{Namespace: endpoint.TargetRef.Namespace, Name: endpoint.TargetRef.Name}, pod)
						if err == nil {
							podName := ""
							podNs := ""
							if endpoint.TargetRef != nil {
								podName = endpoint.TargetRef.Name
								podNs = endpoint.TargetRef.Namespace
							}

							// Skip if terminating
							if pod.DeletionTimestamp != nil {
								reqLogger.V(2).Info("Skipping endpoint for terminating pod", "podName", podName, "namespace", podNs)
								shouldSkip = true
							}
							// Skip if not running
							if pod.Status.Phase != corev1.PodRunning {
								reqLogger.V(2).Info("Skipping endpoint for non-running pod", "podName", pod.Name, "namespace", pod.Namespace, "podPhase", pod.Status.Phase)
								shouldSkip = true
							}
							// Skip if not all containers are ready
							ready := false
							for _, cond := range pod.Status.Conditions {
								if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
									ready = true
									break
								}
							}
							if !ready {
								reqLogger.V(2).Info("Skipping endpoint for non-ready pod", "podName", pod.Name, "namespace", pod.Namespace)
								shouldSkip = true
							}
						}
					}
					if shouldSkip {
						continue
					}
					// Create a new server with the address from the endpoint
					newServer := externalhaproxyoperatorv1alpha1.Server{
						Name:    *endpoint.NodeName,
						Address: endpoint.Addresses[0],
						Port:    server.Port,
					}
					modifiedBackend.Spec.Servers = append(modifiedBackend.Spec.Servers, &newServer)
				}
			}
			// Remove the original server if it was dynamic
			if server.ValueFrom != nil {
				for i, s := range modifiedBackend.Spec.Servers {
					if s.Name == server.Name && s.ValueFrom != nil {
						modifiedBackend.Spec.Servers = append(modifiedBackend.Spec.Servers[:i], modifiedBackend.Spec.Servers[i+1:]...)
						break
					}
				}
			}
		}
	}
	return nil
}

func (r *BackendReconciler) finalizeBackend(reqLogger logr.Logger, m *externalhaproxyoperatorv1alpha1.Backend) error {
	// Delete the resources associated with this Backend
	reqLogger.Info("Finalizing Backend", "name", m.Name)

	// Delete the backend from HAProxy
	if err := r.HAProxyClient.DeleteBackend(m.Name); err != nil {
		// Increment the error counter metric
		monitoring.HAProxyClientErrorCountTotal.Inc()

		reqLogger.Error(err, "Failed to delete backend from HAProxy", "name", m.Name)
		return err
	}

	// Emit an event for the finalization
	r.Recorder.Event(m, "Normal", "Finalized", "Successfully finalized backend")

	// Log the finalization
	reqLogger.Info("Successfully finalized backend")
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *BackendReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&externalhaproxyoperatorv1alpha1.Backend{}).
		Named("backend").
		WithEventFilter(predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				oldObj, ok1 := e.ObjectOld.(*externalhaproxyoperatorv1alpha1.Backend)
				newObj, ok2 := e.ObjectNew.(*externalhaproxyoperatorv1alpha1.Backend)
				if !ok1 || !ok2 {
					// Fallback: process event if types can't be asserted
					return true
				}
				oldCopy := oldObj.DeepCopy()
				newCopy := newObj.DeepCopy()
				newCopy.ObjectMeta.ResourceVersion = oldCopy.ObjectMeta.ResourceVersion
				oldCopy.Status = externalhaproxyoperatorv1alpha1.BackendStatus{}
				newCopy.Status = externalhaproxyoperatorv1alpha1.BackendStatus{}
				return !reflect.DeepEqual(oldCopy, newCopy)
			},
			// Leave other funcs as default (process all Create/Delete/Generic)
		}).
		// Also watch for changes to Services and enqueue all Backend resources in the namespace that reference them
		Watches(&corev1.Service{},
			handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []reconcile.Request {
				return enqueueRequestsFromServices(ctx, obj, r)
			}),
		).
		// Also watch for changes to where the pods for the service are running
		Watches(&discoveryv1.EndpointSlice{},
			handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []reconcile.Request {
				return enqueueRequestsFromEndpointSlices(ctx, obj, r)
			}),
		).
		Complete(r)
}

func enqueueRequestsFromEndpointSlices(ctx context.Context, obj client.Object, r *BackendReconciler) []reconcile.Request {
	reqLogger := logf.FromContext(ctx)
	reqLogger.V(2).Info("Processing EndpointSlice for Backend reconciliation", "endpointSliceName", obj.GetName(), "namespace", obj.GetNamespace())
	var reqs []reconcile.Request
	// List all Backend resources
	backendList := &externalhaproxyoperatorv1alpha1.BackendList{}
	if err := r.List(ctx, backendList, client.InNamespace("")); err != nil {
		return nil
	}
	for _, backend := range backendList.Items {
		// Get the service object from the backend and then get the endpointslices for that service and check if it matches the EndpointSlice
		for _, server := range backend.Spec.Servers {
			// Get the service reference from the server
			if server.ValueFrom != nil && server.ValueFrom.ServiceRef != nil {
				// Get the service object from kubernetes
				service := &corev1.Service{}
				err := r.Get(ctx, client.ObjectKey{
					Namespace: server.ValueFrom.ServiceRef.Namespace,
					Name:      server.ValueFrom.ServiceRef.Name,
				}, service)
				if err != nil {
					if errors.IsNotFound(err) {
						reqLogger.V(2).Info("Service not found for Backend", "serviceName", server.ValueFrom.ServiceRef.Name)
						continue
					}
					reqLogger.Error(err, "Failed to get Service for Backend", "serviceName", server.ValueFrom.ServiceRef.Name)
					continue
				}
				// Check if the EndpointSlice matches the service
				if len(obj.GetLabels()) > 0 && obj.GetLabels()["kubernetes.io/service-name"] == service.Name &&
					obj.GetNamespace() == service.Namespace {
					reqs = append(reqs, reconcile.Request{
						NamespacedName: types.NamespacedName{
							Name:      backend.Name,
							Namespace: backend.Namespace,
						},
					})
					reqLogger.V(2).Info("Found matching Backend for EndpointSlice", "backendName", backend.Name, "serviceName", service.Name, "endpointSliceName", obj.GetName())
					break // No need to check other servers in this backend
				} else {
					reqLogger.V(2).Info("EndpointSlice does not match Backend", "backendName", backend.Name, "serviceName", service.Name, "endpointSliceName", obj.GetName())
				}
			}
		}
	}
	return reqs
}

func enqueueRequestsFromServices(ctx context.Context, obj client.Object, r *BackendReconciler) []reconcile.Request {
	reqLogger := logf.FromContext(ctx)
	reqLogger.V(2).Info("Processing Service for Backend reconciliation", "serviceName", obj.GetName(), "namespace", obj.GetNamespace())
	var reqs []reconcile.Request
	// List all Backend resources
	backendList := &externalhaproxyoperatorv1alpha1.BackendList{}
	if err := r.List(ctx, backendList, client.InNamespace("")); err != nil {
		return nil
	}
	for _, backend := range backendList.Items {
		// Check if the Backend references this Service
		for _, server := range backend.Spec.Servers {
			if server.ValueFrom != nil && server.ValueFrom.ServiceRef != nil &&
				server.ValueFrom.ServiceRef.Name == obj.GetName() &&
				server.ValueFrom.ServiceRef.Namespace == obj.GetNamespace() {
				reqs = append(reqs, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      backend.Name,
						Namespace: backend.Namespace,
					},
				})
			}
		}
	}
	return reqs
}
