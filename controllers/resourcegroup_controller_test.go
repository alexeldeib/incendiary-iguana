/*
Copyright 2019 Alexander Eldeib.
*/

package controllers

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func TestReconcile(t *testing.T) {
	var _ = Describe("SimpleReconcile", func() {
		var (
			// resourceGroup           azurev1alpha1.ResourceGroup
			// ctx        context.Context
			reconciler *ResourceGroupReconciler
			stop       chan struct{}
		)

		BeforeEach(func() {
			// ctx = context.TODO()
			ctrl.SetLogger(zap.Logger(true))

			mgr, err := manager.New(cfg, manager.Options{MetricsBindAddress: "0"})
			Expect(err).ShouldNot(HaveOccurred())

			reconciler = &ResourceGroupReconciler{
				// Config:       mockConfig,
				Client: k8sClient,
				Log:    ctrl.Log.WithName("controllers"),
				// GroupsClient: mockResourceGroupClient,
			}

			err = reconciler.SetupWithManager(mgr)
			Expect(err).ShouldNot(HaveOccurred())
			stop = StartTestManager(mgr, t)
		})

		AfterEach(func() {
			close(stop)
		})

		It("Should fail to find without error", func() {
			// req := reconcile.Request{
			// 	NamespacedName: types.NamespacedName{
			// 		Name:      "foo",
			// 		Namespace: "bar",
			// 	},
			// }

			// spec := azurev1alpha1.ResourceGroup{
			// 	ObjectMeta: metav1.ObjectMeta{
			// 		Name:       "foo",
			// 		Namespace:  "bar",
			// 		Finalizers: []string{"resourcegroup.azure.alexeldeib.xyz"},
			// 	},
			// 	Spec: azurev1alpha1.ResourceGroupSpec{
			// 		Name:           "foobar",
			// 		Location:       "westus2",
			// 		SubscriptionID: "00000000-0000-0000-0000-000000000000",
			// 	},
			// }

			// fakeGroup := resources.Group{
			// 	Properties: &resources.GroupProperties{
			// 		ProvisioningState: to.StringPtr("Succeeded"),
			// 	},
			// 	Response: autorest.Response{
			// 		Response: &http.Response{
			// 			Status:     "200 OK",
			// 			StatusCode: 200,
			// 			Proto:      "HTTP/1.1",
			// 			ProtoMajor: 1,
			// 			ProtoMinor: 1,
			// 			Header:     make(http.Header),
			// 		},
			// 	},
			// }
		})
	})
}

// func TestSimpleReconcile(t *testing.T) {
// 	ctrl := gomock.NewController(t)
// 	defer ctrl.Finish()

// 	mockReconciler := mock_controllers.NewMockAzureResourceGroupReconciler(ctrl)
// 	_ = mockReconciler
// 	return
// }
