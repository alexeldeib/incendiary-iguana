/*
Copyright 2019 Alexander Eldeib.
*/

package controllers

import (
	"testing"

	. "github.com/onsi/ginkgo"
	// . "github.com/onsi/gomega"
)

func TestReconcile(t *testing.T) {
	var _ = Describe("SimpleReconcile", func() {
		It("Should fail to find without error", func() {
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
		})
	})
}
