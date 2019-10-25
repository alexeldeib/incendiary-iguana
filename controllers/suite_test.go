/*

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

package controllers

import (
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/Azure/go-autorest/autorest"
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/reconcilers"
	"github.com/alexeldeib/incendiary-iguana/pkg/reconcilers/generic"
	"github.com/alexeldeib/incendiary-iguana/pkg/services"
	// +kubebuilder:scaffold:imports
)

var (
	cfg            *rest.Config
	k8sClient      client.Client
	mgr            ctrl.Manager
	testEnv        *envtest.Environment
	doneMgr        = make(chan struct{})
	log            logr.Logger
	mgmtAuthorizer autorest.Authorizer
	groupsClient   resources.GroupsClient
	groupService   *services.FakeResourceGroupService
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"Controller Suite",
		[]Reporter{envtest.NewlineReporter{}})
}

var _ = BeforeSuite(func(done Done) {
	rand.Seed(time.Now().Unix())
	logf.SetLogger(zap.LoggerTo(GinkgoWriter, true))

	// Retrieve authentication parameters
	app := os.Getenv("AZURE_CLIENT_ID")
	key := os.Getenv("AZURE_CLIENT_SECRET")
	tenant := os.Getenv("AZURE_TENANT_ID")
	subscription := os.Getenv("AZURE_SUBSCRIPTION_ID")

	if app == "" || key == "" || tenant == "" {
		Fail(fmt.Sprintf("must specify all of app, key, and tenant for client credential authentication, app: [ %s ], sub: [ %s ], tenant: [ %s ], key length: [ %d ]", app, subscription, tenant, len(key)))
	}

	log = logf.Log.WithName("testmanager")
	log.WithValues("subscription", subscription, "app", app, "tenant", tenant).Info("using client configuration")

	// // Setup authorizer
	// mgmtAuthorizer, err := authorizer.New(authorizer.ClientCredentials(app, key, tenant))
	// Expect(err).ToNot(HaveOccurred())

	// // Real Azure clients
	// groupsClient, err = clients.NewGroupsClient(subscription, mgmtAuthorizer)
	// Expect(err).ToNot(HaveOccurred())

	// // Azure service wrappers
	// groupService = &services.ResourceGroupService{
	// 	Authorizer: mgmtAuthorizer,
	// }

	groupService = &services.FakeResourceGroupService{
		State:  map[string]string{},
		Exists: map[string]bool{},
		Start:  time.Now(),
	}

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{},
	}

	cfg, err := testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(cfg).ToNot(BeNil())

	By("Building scheme")
	err = azurev1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	By("setting up a new manager")
	mgr, err = ctrl.NewManager(cfg, ctrl.Options{
		Scheme:             scheme.Scheme,
		MetricsBindAddress: "0",
	})
	By("waiting for manager")
	Expect(err).ToNot(HaveOccurred())

	k8sClient = mgr.GetClient()
	Expect(k8sClient).ToNot(BeNil())

	// +kubebuilder:scaffold:scheme

	By("creating reconciler")
	Expect((&ResourceGroupController{
		Reconciler: &generic.AsyncReconciler{
			Client:   k8sClient,
			Logger:   log,
			Recorder: mgr.GetEventRecorderFor("testmanager"),
			Scheme:   scheme.Scheme,
			AsyncActuator: &reconcilers.ResourceGroupReconciler{
				Service: groupService,
			},
		},
	}).SetupWithManager(mgr)).NotTo(HaveOccurred())

	By("starting the manager")
	go func() {
		Expect(mgr.Start(doneMgr)).ToNot(HaveOccurred())
	}()

	close(done)
}, 120)

var _ = AfterSuite(func() {
	By("closing the manager")
	close(doneMgr)
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())
})
