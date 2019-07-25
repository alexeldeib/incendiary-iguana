/*
Copyright 2019 Alexander Eldeib.
*/

package controllers

import (
	"context"
	"net/http"
	"testing"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/to"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/mocks/mock_config"
	"github.com/alexeldeib/incendiary-iguana/pkg/mocks/mock_resourcegroup"
)

func TestReconcile(t *testing.T) {
	var _ = Describe("SimpleReconcile", func() {
		var (
			// resourceGroup           azurev1alpha1.ResourceGroup
			ctx                     context.Context
			mockCtrl                *gomock.Controller
			mockConfig              *mock_config.MockConfig
			mockResourceGroupClient *mock_resourcegroup.MockClient
			reconciler              *ResourceGroupReconciler
			stop                    chan struct{}
		)

		BeforeEach(func() {
			ctx = context.TODO()
			mockCtrl = gomock.NewController(GinkgoT())
			mockConfig = mock_config.NewMockConfig(mockCtrl)
			mockResourceGroupClient = mock_resourcegroup.NewMockClient(mockCtrl)
			ctrl.SetLogger(zap.Logger(true))

			mgr, err := manager.New(cfg, manager.Options{MetricsBindAddress: "0"})
			Expect(err).ShouldNot(HaveOccurred())

			reconciler = &ResourceGroupReconciler{
				Config: mockConfig,
				Client: k8sClient,
				Log:    ctrl.Log.WithName("controllers"),
				Azure:  mockResourceGroupClient,
			}

			err = reconciler.SetupWithManager(mgr)
			Expect(err).ShouldNot(HaveOccurred())
			stop = StartTestManager(mgr, t)
		})

		AfterEach(func() {
			close(stop)
			mockCtrl.Finish()
		})

		It("Should fail to find without error", func() {
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "foo",
					Namespace: "bar",
				},
			}

			spec := azurev1alpha1.ResourceGroup{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "foo",
					Namespace:  "bar",
					Finalizers: []string{"resourcegroup.azure.alexeldeib.xyz"},
				},
				Spec: azurev1alpha1.ResourceGroupSpec{
					Name:           "foobar",
					Location:       "westus2",
					SubscriptionID: "00000000-0000-0000-0000-000000000000",
				},
			}

			fakeGroup := resources.Group{
				Properties: &resources.GroupProperties{
					ProvisioningState: to.StringPtr("Succeeded"),
				},
				Response: autorest.Response{
					Response: &http.Response{
						Status:     "200 OK",
						StatusCode: 200,
						Proto:      "HTTP/1.1",
						ProtoMajor: 1,
						ProtoMinor: 1,
						Header:     make(http.Header),
					},
				},
			}

			_, err := reconciler.Reconcile(req)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(err).To(BeNil())

			err = k8sClient.Create(ctx, &spec)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(err).To(BeNil())

			mockResourceGroupClient.EXPECT().ForSubscription("00000000-0000-0000-0000-000000000000").AnyTimes()

			// TODO(ace): use accurate return values and test behavior.
			gomock.InOrder(
				mockResourceGroupClient.EXPECT().Get(ctx, &spec),
				mockResourceGroupClient.EXPECT().Ensure(ctx, gomock.Any()),
				mockResourceGroupClient.EXPECT().Get(ctx, gomock.Any()).Return(fakeGroup, nil),
				mockResourceGroupClient.EXPECT().Ensure(ctx, gomock.Any()),
			)

			_, err = reconciler.Reconcile(req)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(err).To(BeNil())

			// Rereconcile should be idempotent.
			_, err = reconciler.Reconcile(req)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(err).To(BeNil())

			err = k8sClient.Get(ctx, req.NamespacedName, &spec)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(err).To(BeNil())
			Expect(spec.Status.ProvisioningState).To(Equal("Succeeded"))

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
