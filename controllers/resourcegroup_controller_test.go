/*
Copyright 2019 Alexander Eldeib.
*/

package controllers

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/stringutil"
)

var _ = Describe("resource group controller", func() {

	const timeout = time.Second * 900
	const interval = time.Second * 5

	It("should create successfully", func() {
		name := fmt.Sprintf("test-ctrl-%s", stringutil.GenerateLowerCaseAlphaNumeric(16))
		subscription := os.Getenv("AZURE_SUBSCRIPTION_ID")
		if subscription == "" {
			Fail("subscription can't be empty")
		}

		key := types.NamespacedName{
			Name:      name,
			Namespace: "default",
		}

		rg := &azurev1alpha1.ResourceGroup{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: "default",
			},
			Spec: azurev1alpha1.ResourceGroupSpec{
				Name:           name,
				Location:       "westus2",
				SubscriptionID: subscription,
			},
		}

		cleanup := func() {
			future, err := groupsClient.Delete(context.Background(), name)
			res := future.Response()
			for res.StatusCode == http.StatusConflict {
				future, err = groupsClient.Delete(context.Background(), name)
				res = future.Response()
			}
			if res.StatusCode == http.StatusNotFound {
				return
			}
			if err != nil {
				Fail(fmt.Sprintf("failed to clean up resource group manually, please go clean up the resources. sub: %s, rg: %s, error: %v\n", subscription, name, err))
			}
			err = future.WaitForCompletionRef(context.Background(), groupsClient.Client)
			if err != nil {
				Fail(fmt.Sprintf("failed to wait for cleanup when deleting resource group, please go ensure resources are cleaned up. sub: %s, rg: %s, error: %v\n", subscription, name, err))
			}
		}

		defer cleanup()

		// Create
		By("expecting successful creation")
		Expect(k8sClient.Create(context.Background(), rg)).Should(Succeed())

		By("checking against crd")
		Eventually(func() bool {
			local := &azurev1alpha1.ResourceGroup{}
			k8sClient.Get(context.Background(), key, local)
			return local.Status.ProvisioningState != nil && *local.Status.ProvisioningState == "Succeeded"
		}, timeout, interval).Should(BeTrue())

		By("checking against azure")
		Eventually(func() bool {
			_, err := groupService.Get(context.Background(), rg)
			return err == nil
		}, timeout, interval).Should(BeTrue())

		// Delete
		By("expecting successful deletion")
		Eventually(func() error {
			err := k8sClient.Delete(context.Background(), rg)
			if err != nil {
				log.Error(err, err.Error())
			}
			return err
		}, timeout, interval).Should(Succeed())

		By("checking deletion against crd")
		Eventually(func() error {
			local := &azurev1alpha1.ResourceGroup{}
			err := k8sClient.Get(context.Background(), key, local)
			if err != nil {
				log.Error(err, err.Error(), "provisioningState", local.Status.ProvisioningState)
			}
			return err
		}, timeout, interval).ShouldNot(Succeed())

		By("checking deletion against azure")
		Eventually(func() bool {
			remote, err := groupService.Get(context.Background(), rg)
			if err != nil {
				log.Error(err, err.Error())
			}
			if remote.Properties != nil {
				log.Info("state", "provisioningState", remote.Properties.ProvisioningState, "statusCode", remote.Response.StatusCode)
			}
			return err != nil && remote.IsHTTPStatus(http.StatusNotFound)
		}, timeout, interval).Should(BeTrue())
	})
})
