/*
Copyright 2019 Alexander Eldeib.
*/

package controllers

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("resource group controller", func() {

	const timeout = time.Second * 60
	const interval = time.Second * 3

	It("should create successfully", func() {
		key := types.NamespacedName{
			Name:      "test-crd",
			Namespace: "default",
		}

		rg := &azurev1alpha1.ResourceGroup{
			ObjectMeta: metav1.ObjectMeta{
				Name:       "test-crd",
				Namespace:  "default",
				Finalizers: []string{"resourcegroup.azure.alexeldeib.xyz"},
			},
			Spec: azurev1alpha1.ResourceGroupSpec{
				Name:           "test-crd",
				Location:       "westus2",
				SubscriptionID: "bd6a4e14-55fa-4160-a6a7-b718d7a2c95c",
			},
		}

		// Create
		Expect(k8sClient.Create(context.Background(), rg)).Should(Succeed())

		By("expecting successful creation")
		Eventually(func() bool {
			local := &azurev1alpha1.ResourceGroup{}
			k8sClient.Get(context.Background(), key, local)
			return local.Status.ProvisioningState != nil && *local.Status.ProvisioningState == "Succeeded"
		}, timeout, interval).Should(BeTrue())

		// Delete
		By("expecting successful deletion")
		Eventually(func() error {
			local := &azurev1alpha1.ResourceGroup{}
			k8sClient.Get(context.Background(), key, local)
			return k8sClient.Delete(context.Background(), local)
		}, timeout, interval).Should(Succeed())

		By("expecting successful completion")
		Eventually(func() error {
			local := &azurev1alpha1.ResourceGroup{}
			return k8sClient.Get(context.Background(), key, local)
		}, timeout, interval).ShouldNot(Succeed())
	})
})
