package ensure_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Azure/go-autorest/autorest/to"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/cmd/ensure"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/identities"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/keyvaults"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/loadbalancers"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/publicips"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/redis"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/resourcegroups"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/secretbundles"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/secrets"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/securitygroups"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/servicebus"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/subnets"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/trafficmanagers"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/virtualnetworks"
	"github.com/alexeldeib/incendiary-iguana/pkg/config"
)

const testdata = "./testdata/group.yaml"

var configuration *config.Config

var (
	log                 = ctrl.Log.WithName("test")
	identitiesClient    *identities.Client
	loadbalancersClient *loadbalancers.Client
	publicIPClient      *publicips.Client
	redisClient         *redis.Client
	rgClient            *resourcegroups.GroupClient
	sbnamespaceClient   *servicebus.Client
	secretsClient       *secrets.Client
	secretbundlesClient *secretbundles.Client
	sgClient            *securitygroups.Client
	subnetClient        *subnets.Client
	tmClient            *trafficmanagers.Client
	vaultClient         *keyvaults.Client
	vnetClient          *virtualnetworks.Client
)

func TestEnsure(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "cli")
}

var _ = BeforeSuite(func() {
	var err error
	if configuration, err = config.New(); err != nil {
		Fail(err.Error())
	}
	identitiesClient = identities.New(configuration)
	loadbalancersClient = loadbalancers.New(configuration)
	publicIPClient = publicips.New(configuration)
	rgClient = resourcegroups.NewGroupClient(configuration)
	redisClient = redis.New(configuration, nil, nil)
	sbnamespaceClient = servicebus.New(configuration, nil, nil)
	sgClient = securitygroups.New(configuration)
	subnetClient = subnets.New(configuration)
	tmClient = trafficmanagers.New(configuration)
	vaultClient = keyvaults.New(configuration)
	vnetClient = virtualnetworks.New(configuration)

	kubeclient, err := client.New(ctrl.GetConfigOrDie(), client.Options{})
	Expect(err).ToNot(HaveOccurred())

	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = azurev1alpha1.AddToScheme(scheme)

	secretbundlesClient, err = secretbundles.New(configuration, &kubeclient, scheme)
	Expect(err).ToNot(HaveOccurred())
	secretsClient, err = secrets.New(configuration, &kubeclient, scheme)
	Expect(err).ToNot(HaveOccurred())
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

	objects, err := options.Read(nil)
	It("should read object successfully", func() {
		Expect(err).ToNot(HaveOccurred())
		Expect(len(objects)).To(Equal(1))
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
			SubscriptionID: "bd6a4e14-55fa-4160-a6a7-b718d7a2c95c",
			ResourceGroup:  "test-crd",
			Location:       "westus2",
			Addresses: []string{
				"10.0.0.0/8",
				"192.168.0.0/24",
			},
		},
	}

	subnet := &azurev1alpha1.Subnet{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-crd",
		},
		Spec: azurev1alpha1.SubnetSpec{
			Name:           "ace-subnet",
			SubscriptionID: "bd6a4e14-55fa-4160-a6a7-b718d7a2c95c",
			ResourceGroup:  "test-crd",
			Network:        "ace-vnet",
			Subnet:         "10.0.0.0/28",
		},
	}

	sg := &azurev1alpha1.SecurityGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-crd",
		},
		Spec: azurev1alpha1.SecurityGroupSpec{
			Name:           "ace-subnet",
			SubscriptionID: "bd6a4e14-55fa-4160-a6a7-b718d7a2c95c",
			ResourceGroup:  "test-crd",
			Location:       "westus2",
			Rules: []azurev1alpha1.SecurityRule{
				{
					Name:                     "test-rule",
					Protocol:                 "tcp",
					Access:                   "deny",
					Direction:                "inbound",
					SourcePortRange:          to.StringPtr("1-65535"),
					DestinationPortRange:     to.StringPtr("443"),
					SourceAddressPrefix:      to.StringPtr("*"),
					DestinationAddressPrefix: to.StringPtr("*"),
					Priority:                 to.Int32Ptr(205),
				},
			},
		},
	}

	ip := &azurev1alpha1.PublicIP{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-crd",
		},
		Spec: azurev1alpha1.PublicIPSpec{
			Name:           "ace-ip",
			SubscriptionID: "bd6a4e14-55fa-4160-a6a7-b718d7a2c95c",
			ResourceGroup:  "test-crd",
			Location:       "westus2",
		},
	}

	lb := &azurev1alpha1.LoadBalancer{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-crd",
		},
		Spec: azurev1alpha1.LoadBalancerSpec{
			Name:           "ace-lb",
			SubscriptionID: "bd6a4e14-55fa-4160-a6a7-b718d7a2c95c",
			ResourceGroup:  "test-crd",
			Location:       "westus2",
			Frontends: []string{
				"/subscriptions/bd6a4e14-55fa-4160-a6a7-b718d7a2c95c/resourceGroups/test-crd/providers/Microsoft.Network/publicIpAddresses/ace-ip",
			},
			BackendPools: []string{"backendPool"},
			Probes:       &[]int{6443},
			Rules:        &[]azurev1alpha1.RuleSpec{
				// {
				// 	Name:         "lb_rule",
				// 	Frontend:     "/subscriptions/bd6a4e14-55fa-4160-a6a7-b718d7a2c95c/resourceGroups/test-crd/providers/Microsoft.Network/loadBalancers/ace-lb/frontendIPConfigurations/ace-ip",
				// 	BackendPool:  "/subscriptions/bd6a4e14-55fa-4160-a6a7-b718d7a2c95c/resourceGroups/test-crd/providers/Microsoft.Network/loadBalancers/ace-lb/backendAddressPools/backendPool",
				// 	Probe:        "/subscriptions/bd6a4e14-55fa-4160-a6a7-b718d7a2c95c/resourceGroups/test-crd/providers/Microsoft.Network/loadBalancers/ace-lb/probes/probe_6443",
				// 	FrontendPort: 443,
				// 	BackendPort:  443,
				// 	Protocol:     "Tcp",
				// },
			},
		},
	}

	tm := &azurev1alpha1.TrafficManager{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-crd",
		},
		Spec: azurev1alpha1.TrafficManagerSpec{
			Name:                 "ace-tm",
			SubscriptionID:       "bd6a4e14-55fa-4160-a6a7-b718d7a2c95c",
			ResourceGroup:        "test-crd",
			ProfileStatus:        "enabled",
			TrafficRoutingMethod: "weighted",
			DNSConfig: azurev1alpha1.DNSConfig{
				RelativeName: to.StringPtr("acetmnew"),
				TTL:          to.Int64Ptr(30),
			},
			MonitorConfig: azurev1alpha1.MonitorConfig{
				IntervalInSeconds:         to.Int64Ptr(10),
				Path:                      to.StringPtr("/"),
				Port:                      to.Int64Ptr(443),
				Protocol:                  "HTTPS",
				TimeoutInSeconds:          to.Int64Ptr(5),
				ToleratedNumberOfFailures: to.Int64Ptr(3),
				CustomHeaders: &[]azurev1alpha1.MonitorConfigCustomHeadersItem{
					{
						Name:  to.StringPtr("host"),
						Value: to.StringPtr("google.com"),
					},
				},
				ExpectedStatusCodeRanges: &[]azurev1alpha1.MonitorConfigExpectedStatusCodeRangesItem{
					{
						Min: to.Int32Ptr(200),
						Max: to.Int32Ptr(308),
					},
				},
			},
			Endpoints: &[]azurev1alpha1.EndpointSpec{
				{
					Name: "google-1",
					Properties: azurev1alpha1.EndpointProperties{
						Target:           to.StringPtr("google.com"),
						Weight:           to.Int64Ptr(1),
						Priority:         int64(1),
						EndpointLocation: "eastus",
					},
				},
			},
		},
	}

	cache := &azurev1alpha1.Redis{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-crd",
		},
		Spec: azurev1alpha1.RedisSpec{
			Name:             "ace-redis-test",
			SubscriptionID:   "bd6a4e14-55fa-4160-a6a7-b718d7a2c95c",
			ResourceGroup:    "test-crd",
			Location:         "westus2",
			EnableNonSslPort: false,
			SKU: azurev1alpha1.RedisSku{
				Name:     "premium",
				Family:   "p",
				Capacity: 1,
			},
		},
	}

	sbnamespace := &azurev1alpha1.ServiceBusNamespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-crd",
		},
		Spec: azurev1alpha1.ServiceBusNamespaceSpec{
			Name:           "ace-sb-test",
			SubscriptionID: "bd6a4e14-55fa-4160-a6a7-b718d7a2c95c",
			ResourceGroup:  "test-crd",
			Location:       "westus2",
			SKU: azurev1alpha1.ServiceBusNamespaceSku{
				Name:     "Premium",
				Tier:     "Premium",
				Capacity: 1,
			},
		},
	}

	vault := &azurev1alpha1.Keyvault{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-crd",
		},
		Spec: azurev1alpha1.KeyvaultSpec{
			Name:           "ace-kv-test",
			SubscriptionID: "bd6a4e14-55fa-4160-a6a7-b718d7a2c95c",
			ResourceGroup:  "test-crd",
			Location:       "westus2",
			TenantID:       "33e01921-4d64-4f8c-a055-5bdaffd5e33d",
		},
	}

	identity := &azurev1alpha1.Identity{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-crd",
		},
		Spec: azurev1alpha1.IdentitySpec{
			Name:           "ace-msi",
			SubscriptionID: "bd6a4e14-55fa-4160-a6a7-b718d7a2c95c",
			ResourceGroup:  "test-crd",
			Location:       "westus2",
		},
	}

	secret := &azurev1alpha1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-crd",
			Namespace: "default",
		},
		Spec: azurev1alpha1.SecretSpec{
			SecretIdentifier: azurev1alpha1.SecretIdentifier{
				Name:  "ace-secret",
				Vault: "ace-kv-test",
			},
		},
	}

	secretbundle := &azurev1alpha1.SecretBundle{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-crd",
			Namespace: "default",
		},
		Spec: azurev1alpha1.SecretBundleSpec{
			Name:    "ace-secretbundle",
			Secrets: map[string]azurev1alpha1.SecretIdentifier{},
		},
	}

	_ = vnet
	_ = subnet
	_ = sg
	_ = ip
	_ = lb
	_ = tm
	_ = cache
	_ = sbnamespace
	_ = vault
	_ = identity
	_ = secretbundle
	_ = secret

	Context("ensure", func() {
		It("should create rg successfully", func() {
			err := ensure.EnsureAsync(rgClient, rg, log)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should create vnet successfully", func() {
			err := ensure.EnsureAsync(vnetClient, vnet, log)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should create subnet successfully", func() {
			err := ensure.EnsureAsync(subnetClient, subnet, log)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should create sg successfully", func() {
			err := ensure.EnsureAsync(sgClient, sg, log)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should create ip successfully", func() {
			err := ensure.EnsureAsync(publicIPClient, ip, log)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should create lb successfully", func() {
			err := ensure.EnsureAsync(loadbalancersClient, lb, log)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should create tm successfully", func() {
			err := ensure.EnsureTrafficManager(tmClient, tm, log)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should create vault successfully", func() {
			err := ensure.EnsureSync(vaultClient, vault, log)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should create secretbundle successfully", func() {
			err := ensure.EnsureSync(secretbundlesClient, secretbundle, log)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should create managed identity successfully", func() {
			err := ensure.EnsureSync(identitiesClient, identity, log)
			Expect(err).ToNot(HaveOccurred())
		})

		// It("should create redis successfully", func() {
		// 	err := ensure.EnsureAsync(redisClient, cache, log)
		// 	Expect(err).ToNot(HaveOccurred())
		// })

		// It("should create servicebus namespace successfully", func() {
		// 	err := ensure.EnsureAsync(sbnamespaceClient, sbnamespace, log)
		// 	Expect(err).ToNot(HaveOccurred())
		// })
	})

	Context("delete", func() {
		It("should delete servicebus namespace successfully", func() {
			err := ensure.DeleteAsync(sbnamespaceClient, sbnamespace, log)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should delete redis successfully", func() {
			err := ensure.DeleteAsync(redisClient, cache, log)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should delete managed identity successfully", func() {
			err := ensure.DeleteSync(identitiesClient, identity, log)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should delete secretbundle successfully", func() {
			err := ensure.DeleteSync(secretbundlesClient, secretbundle, log)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should delete vault successfully", func() {
			err := ensure.DeleteSync(vaultClient, vault, log)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should delete tm successfully", func() {
			err := ensure.DeleteTrafficManager(tmClient, tm, log)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should delete lb successfully", func() {
			err := ensure.DeleteAsync(loadbalancersClient, lb, log)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should delete ip successfully", func() {
			err := ensure.DeleteAsync(publicIPClient, ip, log)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should delete sg successfully", func() {
			err := ensure.DeleteAsync(sgClient, sg, log)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should delete subnet successfully", func() {
			err := ensure.DeleteAsync(subnetClient, subnet, log)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should delete vnet successfully", func() {
			err := ensure.DeleteAsync(vnetClient, vnet, log)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should delete rg successfully", func() {
			err := ensure.DeleteAsync(rgClient, rg, log)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
