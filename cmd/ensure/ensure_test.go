package ensure_test

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/cmd/ensure"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/resourcegroups"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/virtualnetworks"
	"github.com/alexeldeib/incendiary-iguana/pkg/config"
)

const testdata = "./testdata/group.yaml"

var (
	log           = ctrl.Log.WithName("test")
	configuration = config.New(log)
	rgclient      *resourcegroups.Client
	vnetclient    *virtualnetworks.Client
)

func TestEnsure(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "cli")
}

var _ = BeforeSuite(func() {
	if err := configuration.DetectAuthorizer(); err != nil {
		Fail(err.Error())
	}
	rgclient = resourcegroups.New(configuration)
	vnetclient = virtualnetworks.New(configuration)
})

var _ = Describe("read yaml + parse resources", func() {
	wd, err := os.Getwd()
	if err != nil {
		Fail(err.Error())
	}

	path, err := filepath.Abs(filepath.Join(wd, testdata))
	if err != nil {
		Fail(err.Error())
	}
	options := &ensure.EnsureOptions{
		File: path,
	}

	objects, err := options.Read()
	It("should read object successfully", func() {
		Expect(err).ToNot(HaveOccurred())
		Expect(len(objects)).To(Equal(1))
		Expect(objects[0].GetName()).To(Equal("test-crd"))
	})

	It("should parse rg successfully", func() {
		rg, ok := objects[0].(*azurev1alpha1.ResourceGroup)
		Expect(ok).To(Equal(true))
		Expect(rg.Name).To(Equal("test-crd"))
		Expect(rg.Spec.Name).To(Equal("test-crd"))
		Expect(rg.Spec.Location).To(Equal("westus2"))
		Expect(rg.Spec.SubscriptionID).To(Equal("bd6a4e14-55fa-4160-a6a7-b718d7a2c95c"))
	})
})

var _ = Describe("reconcile", func() {

	rg := &azurev1alpha1.ResourceGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-crd",
		},
		Spec: azurev1alpha1.ResourceGroupSpec{
			Name:           "test-crd",
			Location:       "westus2",
			SubscriptionID: "bd6a4e14-55fa-4160-a6a7-b718d7a2c95c",
		},
	}

	vnet := &azurev1alpha1.VirtualNetwork{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-crd",
		},
		Spec: azurev1alpha1.VirtualNetworkSpec{
			Name:           "ace-vnet",
			ResourceGroup:  "test-crd",
			Location:       "westus2",
			SubscriptionID: "bd6a4e14-55fa-4160-a6a7-b718d7a2c95c",
			Addresses: []string{
				"10.0.0.0/8",
				"192.168.0.0/24",
			},
		},
	}

	Context("ensure", func() {
		It("should create rg successfully", func() {
			err := ensure.EnsureResourceGroup(rgclient, rg, log)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should create vnet successfully", func() {
			err := ensure.EnsureVirtualNetwork(vnetclient, vnet, log)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("delete", func() {
		It("should delete vnet successfully", func() {
			err := ensure.DeleteVirtualNetwork(vnetclient, vnet, log)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should delete rg successfully", func() {
			err := ensure.DeleteResourceGroup(rgclient, rg, log)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
