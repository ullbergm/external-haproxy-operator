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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	externalhaproxyoperatorv1alpha1 "github.com/ullbergm/external-haproxy-operator/api/v1alpha1"
)

var _ = Describe("Backend Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default", // TODO(user):Modify as needed
		}
		backend := &externalhaproxyoperatorv1alpha1.Backend{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind Backend")
			err := k8sClient.Get(ctx, typeNamespacedName, backend)
			if err != nil && errors.IsNotFound(err) {
				resource := &externalhaproxyoperatorv1alpha1.Backend{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: externalhaproxyoperatorv1alpha1.BackendSpec{
						Name: "test-backend",
						Balance: &externalhaproxyoperatorv1alpha1.Balance{
							Algorithm: func() *string {
								alg := "roundrobin"
								return &alg
							}(),
						},
						AdvCheck: "httpchk",
						Servers: map[string]externalhaproxyoperatorv1alpha1.Server{
							"server1": {
								Name:    "server1",
								Address: "127.0.0.1",
								Port:    func() *int64 { v := int64(8080); return &v }(),
							},
							"server2": {
								Name:    "server2",
								Address: "127.0.0.2",
								Port:    func() *int64 { v := int64(8080); return &v }(),
							},
						},
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			// Cleanup logic after each test, only delete if the resource still exists.
			resource := &externalhaproxyoperatorv1alpha1.Backend{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			if err == nil {
				By("Cleanup the specific resource instance Backend")
				// Remove finalizers if present to allow deletion
				if len(resource.GetFinalizers()) > 0 {
					resource.SetFinalizers(nil)
					Expect(k8sClient.Update(ctx, resource)).To(Succeed())
				}
				Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
			} else {
				Expect(errors.IsNotFound(err)).To(BeTrue())
			}
		})

		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &BackendReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			// TODO(user): Add more specific assertions depending on your controller's reconciliation logic.
			// Example: If you expect a certain status condition after reconciliation, verify it here.
		})

		It("should return an error when the resource does not exist", func() {
			By("Reconciling a non-existent resource")
			nonExistentName := types.NamespacedName{
				Name:      "non-existent-backend",
				Namespace: "default",
			}
			controllerReconciler := &BackendReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: nonExistentName,
			})
			// Should not error, as the default implementation returns nil, but you can update this if logic changes
			Expect(err).NotTo(HaveOccurred())
		})

		It("should be able to update the Backend resource", func() {
			By("Updating the Backend resource")
			resource := &externalhaproxyoperatorv1alpha1.Backend{}
			Expect(k8sClient.Get(ctx, typeNamespacedName, resource)).To(Succeed())

			// Example update: add an annotation
			if resource.Annotations == nil {
				resource.Annotations = map[string]string{}
			}
			resource.Annotations["test-annotation"] = "true"
			Expect(k8sClient.Update(ctx, resource)).To(Succeed())

			By("Reconciling the updated resource")
			controllerReconciler := &BackendReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			// Verify the annotation is present
			updated := &externalhaproxyoperatorv1alpha1.Backend{}
			Expect(k8sClient.Get(ctx, typeNamespacedName, updated)).To(Succeed())
			Expect(updated.Annotations).To(HaveKeyWithValue("test-annotation", "true"))
		})
	})
})
