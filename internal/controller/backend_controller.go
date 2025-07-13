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
	"github.com/ullbergm/external-haproxy-operator/internal/monitoring"
)

// BackendReconciler reconciles a Backend object
type BackendReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	Recorder      record.EventRecorder
	HAProxyClient haproxyclient.HAProxyClient
}

const backendFinalizer = "external-haproxy-operator.ullberg.us/finalizer"

// +kubebuilder:rbac:groups=external-haproxy-operator.ullberg.us,resources=backends,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=external-haproxy-operator.ullberg.us,resources=backends/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=external-haproxy-operator.ullberg.us,resources=backends/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.

// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *BackendReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := logf.FromContext(ctx)

	reqLogger.Info("Reconciling Backend", "name", req.Name, "namespace", req.Namespace)

	// Fetch the Backend instance
	// backend := &externalhaproxyoperatorv1alpha1.Backend{}
	// if err := r.Get(ctx, req.NamespacedName, backend); err != nil {
	// 	reqLogger.Error(err, "unable to fetch Backend")
	// 	return ctrl.Result{}, client.IgnoreNotFound(err)
	// }
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

	// Create a backend in HAProxy if it does not exist
	if err := r.HAProxyClient.EnsureBackend(externalhaproxyoperatorv1alpha1.BackendSpecToModel(backend.Spec)); err != nil {
		// Increment the error counter metric
		monitoring.HAProxyClientErrorCountTotal.Inc()
		reqLogger.Error(err, "Failed to create backend in HAProxy", "name", backend.Name)
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
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
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
		Complete(r)
}
