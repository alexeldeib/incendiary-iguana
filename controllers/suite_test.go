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
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/resourcegroups"
	"github.com/alexeldeib/incendiary-iguana/pkg/config"
	// +kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var (
	cfg          *rest.Config
	k8sClient    client.Client
	mgr          ctrl.Manager
	testEnv      *envtest.Environment
	groupsClient *resourcegroups.Client
	doneMgr      = make(chan struct{})
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"Controller Suite",
		[]Reporter{envtest.NewlineReporter{}})
}

var _ = BeforeSuite(func(done Done) {
	logf.SetLogger(zap.LoggerTo(GinkgoWriter, true))

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		ControlPlaneStartTimeout: 60 * time.Second,
		CRDDirectoryPaths:        []string{filepath.Join("..", "config", "crd", "bases")},
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(cfg).ToNot(BeNil())

	err = scheme.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = azurev1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	By("setting up a new manager")
	mgr, err = ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
		MetricsBindAddress: "0",
		NewCache: func(config *rest.Config, opts cache.Options) (cache.Cache, error) {
			syncPeriod := 1 * time.Second
			opts.Resync = &syncPeriod
			return cache.New(config, opts)
		},
	})
	By("waiting for manager")
	Expect(err).ToNot(HaveOccurred())

	// +kubebuilder:scaffold:scheme
	By("initializing azure config")
	configuration := config.New(ctrl.Log.WithName("configuration"))
	groupsClient = resourcegroups.New(configuration)
	log := ctrl.Log.WithName("testmanager")
	recorder := mgr.GetEventRecorderFor("testmanager")

	By("creating reconciler")
	Expect((&ResourceGroupReconciler{
		Client:       k8sClient,
		Log:          log.WithName("ResourceGroup"),
		GroupsClient: resourcegroups.New(configuration),
		Reconciler: &AzureReconciler{
			Client:   k8sClient,
			Az:       resourcegroups.New(configuration),
			Log:      log,
			Recorder: recorder,
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
