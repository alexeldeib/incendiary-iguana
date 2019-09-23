/*
Copyright 2019 Alexander Eldeib.
*/

package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/go-autorest/autorest/to"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("sql server controller", func() {

	const timeout = time.Second * 900
	const interval = time.Second * 5

	It("should create successfully", func() {
		key := types.NamespacedName{
			Name:      "test-crd-sql",
			Namespace: "default",
		}

		rg := &azurev1alpha1.ResourceGroup{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-crd-sql",
				Namespace: "default",
			},
			Spec: azurev1alpha1.ResourceGroupSpec{
				Name:           "test-crd-sql",
				Location:       "eastus",
				SubscriptionID: "bd6a4e14-55fa-4160-a6a7-b718d7a2c95c",
			},
		}

		server := &azurev1alpha1.SQLServer{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-crd-sql",
				Namespace: "default",
			},
			Spec: azurev1alpha1.SQLServerSpec{
				ResourceGroup:           "test-crd-sql",
				Name:                    "test-crd-sql",
				Location:                "eastus",
				SubscriptionID:          "bd6a4e14-55fa-4160-a6a7-b718d7a2c95c",
				AllowAzureServiceAccess: to.BoolPtr(false),
			},
		}

		// Create backing rg
		Expect(k8sClient.Create(context.Background(), rg)).Should(Succeed())
		By("expecting successful creation")
		Eventually(func() bool {
			local := &azurev1alpha1.ResourceGroup{}
			k8sClient.Get(context.Background(), key, local)
			return local.Status.ProvisioningState != nil && *local.Status.ProvisioningState == "Succeeded"
		}, timeout, interval).Should(BeTrue())

		Expect(k8sClient.Create(context.Background(), server)).Should(Succeed())

		Eventually(func() bool {
			local := &azurev1alpha1.SQLServer{}
			k8sClient.Get(context.Background(), key, local)
			return local.Status.State != nil && *local.Status.State == "Ready"
		}, timeout, interval).Should(BeTrue())

		By("changing firewall rule")
		server.Spec.AllowAzureServiceAccess = to.BoolPtr(true)
		Expect(k8sClient.Update(context.Background(), server)).Should(Succeed())

		By("waiting for update")
		Eventually(func() bool {
			local := &azurev1alpha1.SQLServer{}
			k8sClient.Get(context.Background(), key, local)
			return local.Status.State != nil && *local.Status.State == "Ready"
		}, timeout, interval).Should(BeTrue())

		By("expecting to find firewall rule")
		Eventually(func() error {
			local := &azurev1alpha1.SQLFirewallRule{}
			return k8sClient.Get(context.Background(), key, local)
		}).Should(Succeed())

		server.Spec.AllowAzureServiceAccess = to.BoolPtr(true)
		Eventually(func() error {
			return k8sClient.Update(context.Background(), server)
		}, timeout, interval).Should(Succeed())

		By("expecting to find secret")
		Eventually(func() error {
			local := &corev1.Secret{}
			return k8sClient.Get(context.Background(), key, local)
		}, timeout, interval).Should(Succeed())

		By("expecting to find admin login/pass")
		secret := &corev1.Secret{}
		Expect(k8sClient.Get(context.Background(), key, secret)).Should(Succeed())
		Expect(len(secret.Data)).Should(BeNumerically("==", 4))

		// Delete
		By("expecting successful deletion")
		Eventually(func() error {
			local := &azurev1alpha1.SQLFirewallRule{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-crd-sql",
					Namespace: "default",
				},
			}
			return k8sClient.Delete(context.Background(), local)
		}, timeout, interval).Should(Succeed())

		// Delete
		By("expecting successful deletion")
		Eventually(func() error {
			local := &azurev1alpha1.ResourceGroup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-crd-sql",
					Namespace: "default",
				},
			}
			return k8sClient.Delete(context.Background(), local)
		}, timeout, interval).Should(Succeed())

		By("expecting successful completion")
		Eventually(func() error {
			local := &azurev1alpha1.ResourceGroup{}
			err := k8sClient.Get(context.Background(), key, local)
			fmt.Printf("error: %+#v\n", err)
			return err
		}, timeout, interval).ShouldNot(Succeed())
	})
})
