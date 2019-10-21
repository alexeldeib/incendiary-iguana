/*
Copyright 2019 Alexander Eldeib.
*/

package main

import (
	"flag"
	"math/rand"
	"os"
	"time"

	"github.com/sanity-io/litter"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/controllers"
	"github.com/alexeldeib/incendiary-iguana/pkg/services/keyvaults"
	"github.com/alexeldeib/incendiary-iguana/pkg/services/nics"
	"github.com/alexeldeib/incendiary-iguana/pkg/services/publicips"
	"github.com/alexeldeib/incendiary-iguana/pkg/services/redis"
	"github.com/alexeldeib/incendiary-iguana/pkg/services/resourcegroups"
	"github.com/alexeldeib/incendiary-iguana/pkg/services/secrets"
	"github.com/alexeldeib/incendiary-iguana/pkg/services/securitygroups"
	"github.com/alexeldeib/incendiary-iguana/pkg/services/servicebus"
	"github.com/alexeldeib/incendiary-iguana/pkg/services/subnets"
	"github.com/alexeldeib/incendiary-iguana/pkg/services/tlssecrets"
	"github.com/alexeldeib/incendiary-iguana/pkg/services/trafficmanagers"
	"github.com/alexeldeib/incendiary-iguana/pkg/services/virtualnetworks"
	"github.com/alexeldeib/incendiary-iguana/pkg/services/vms"
	"github.com/alexeldeib/incendiary-iguana/pkg/config"
	"github.com/alexeldeib/incendiary-iguana/pkg/reconciler"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)
	_ = azurev1alpha1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

func main() {
	rand.Seed(time.Now().Unix())
	var metricsAddr string
	var enableLeaderElection bool
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")

	flag.Parse()

	ctrl.SetLogger(zap.Logger(false))

	configuration, err := config.New()
	if err != nil {
		setupLog.Error(err, "failed to detect any authorizer")
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		LeaderElection:     enableLeaderElection,
	})

	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	log := ctrl.Log.WithName("incendiaryiguana")
	recorder := mgr.GetEventRecorderFor("incendiaryiguana")
	client := mgr.GetClient()

	// Global client initialization
	secretsclient, err := secrets.New(configuration, &client, scheme)
	if err != nil {
		setupLog.Error(err, "failed to initialize keyvault secret client")
		os.Exit(1)
	}

	tlssecretsclient, err := tlssecrets.New(configuration, &client, scheme)
	if err != nil {
		setupLog.Error(err, "failed to initialize keyvault tlssecret client")
		os.Exit(1)
	}

	// TODO(ace): handle this in a loop.
	if err = (&controllers.ResourceGroupController{
		Reconciler: reconciler.NewAsyncReconciler(
			client,
			resourcegroups.NewGroupClient(configuration),
			log,
			recorder,
			scheme,
		),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ResourceGroup")
		os.Exit(1)
	}

	if err = (&controllers.KeyvaultController{
		Reconciler: reconciler.NewSyncReconciler(
			client,
			keyvaults.New(configuration),
			log,
			recorder,
			scheme,
		),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Keyvault")
		os.Exit(1)
	}

	if err = (&controllers.SecretController{
		Reconciler: reconciler.NewSyncReconciler(
			client,
			tlssecretsclient,
			log,
			recorder,
			scheme,
		),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Secret")
		os.Exit(1)
	}

	if err = (&controllers.TLSSecretController{
		Reconciler: reconciler.NewSyncReconciler(
			client,
			tlssecretsclient,
			log,
			recorder,
			scheme,
		),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "TLSSecret")
		os.Exit(1)
	}

	if err = (&controllers.SecretBundleController{
		Reconciler: reconciler.NewSyncReconciler(
			client,
			secretsclient,
			log,
			recorder,
			scheme,
		),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "SecretBundle")
		os.Exit(1)
	}

	if err = (&controllers.VirtualNetworkController{
		Reconciler: reconciler.NewAsyncReconciler(
			client,
			virtualnetworks.New(configuration),
			log,
			recorder,
			scheme,
		),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "VirtualNetwork")
		os.Exit(1)
	}

	if err = (&controllers.SubnetController{
		Reconciler: reconciler.NewAsyncReconciler(
			client,
			subnets.New(configuration),
			log,
			recorder,
			scheme,
		),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Subnet")
		os.Exit(1)
	}

	if err = (&controllers.SecurityGroupController{
		Reconciler: reconciler.NewAsyncReconciler(
			client,
			securitygroups.New(configuration),
			log,
			recorder,
			scheme,
		),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "SecurityGroup")
		os.Exit(1)
	}

	if err = (&controllers.PublicIPController{
		Reconciler: reconciler.NewAsyncReconciler(
			client,
			publicips.New(configuration),
			log,
			recorder,
			scheme,
		),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "PublicIP")
		os.Exit(1)
	}

	if err = (&controllers.NetworkInterfaceController{
		Reconciler: reconciler.NewAsyncReconciler(
			client,
			nics.New(configuration),
			log,
			recorder,
			scheme,
		),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "NetworkInterface")
		os.Exit(1)
	}

	if err = (&controllers.TrafficManagerController{
		Client:                client,
		Log:                   ctrl.Log.WithName("controllers").WithName("TrafficManager"),
		TrafficManagersClient: trafficmanagers.New(configuration),
		Recorder:              recorder,
	}).SetupWithManager(mgr); err != nil {
		litter.Dump(err)
		setupLog.Error(err, "unable to create controller", "controller", "TrafficManager")
		os.Exit(1)
	}

	if err = (&controllers.RedisController{
		Reconciler: reconciler.NewAsyncReconciler(
			client,
			redis.New(configuration, &client, scheme),
			log,
			recorder,
			scheme,
		),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Redis")
		os.Exit(1)
	}

	if err = (&controllers.ServiceBusNamespaceController{
		Reconciler: reconciler.NewAsyncReconciler(
			client,
			servicebus.New(configuration, &client, scheme),
			log,
			recorder,
			scheme,
		),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ServiceBusNamespace")
		os.Exit(1)
	}

	if err = (&controllers.VMController{
		Reconciler: reconciler.NewAsyncReconciler(
			client,
			vms.New(configuration),
			log,
			recorder,
			scheme,
		),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "VM")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
