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
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/keyvaults"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/nics"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/publicips"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/redis"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/resourcegroups"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/secrets"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/securitygroups"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/servicebus"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/subnets"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/trafficmanagers"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/virtualnetworks"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/vms"
	"github.com/alexeldeib/incendiary-iguana/pkg/config"
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

	configuration := config.New(setupLog)
	err := configuration.DetectAuthorizer()
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
	secretsclient, err := secrets.New(configuration, client, scheme)
	if err != nil {
		setupLog.Error(err, "failed to initialize keyvault secret client")
		os.Exit(1)
	}

	if err = (&controllers.ResourceGroupReconciler{
		Client:       client,
		Log:          log.WithName("ResourceGroup"),
		GroupsClient: resourcegroups.New(configuration),
		Reconciler: &controllers.AzureReconciler{
			Client:   client,
			Az:       resourcegroups.New(configuration),
			Log:      log,
			Recorder: recorder,
		},
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ResourceGroup")
		os.Exit(1)
	}

	if err = (&controllers.KeyvaultReconciler{
		Client:       client,
		Log:          log.WithName("Keyvault"),
		Config:       configuration,
		VaultsClient: keyvaults.New(configuration),
		Reconciler: &controllers.AzureSyncReconciler{
			Client:   client,
			Az:       keyvaults.New(configuration),
			Log:      log,
			Recorder: recorder,
		},
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Keyvault")
		os.Exit(1)
	}

	if err = (&controllers.SecretReconciler{
		Client:        client,
		Log:           ctrl.Log.WithName("controllers").WithName("Secret"),
		SecretsClient: secretsclient,
		Scheme:        scheme,
		Reconciler: &controllers.AzureSyncReconciler{
			Client:   client,
			Az:       secretsclient,
			Log:      log,
			Recorder: recorder,
		},
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Secret")
		os.Exit(1)
	}

	if err = (&controllers.SecretBundleReconciler{
		Client:        client,
		Log:           ctrl.Log.WithName("controllers").WithName("SecretBundle"),
		SecretsClient: secretsclient,
		Scheme:        scheme,
		Reconciler: &controllers.AzureSyncReconciler{
			Client:   client,
			Az:       secretsclient,
			Log:      log,
			Recorder: recorder,
		},
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "SecretBundle")
		os.Exit(1)
	}

	if err = (&controllers.VirtualNetworkReconciler{
		Client:      client,
		Log:         ctrl.Log.WithName("controllers").WithName("VirtualNetwork"),
		VnetsClient: virtualnetworks.New(configuration),
		Reconciler: &controllers.AzureReconciler{
			Client:   client,
			Az:       virtualnetworks.New(configuration),
			Log:      log,
			Recorder: recorder,
		},
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "VirtualNetwork")
		os.Exit(1)
	}

	if err = (&controllers.SubnetReconciler{
		Client:        client,
		Log:           ctrl.Log.WithName("controllers").WithName("Subnet"),
		SubnetsClient: subnets.New(configuration),
		Reconciler: &controllers.AzureReconciler{
			Client:   client,
			Az:       subnets.New(configuration),
			Log:      log,
			Recorder: recorder,
		},
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Subnet")
		os.Exit(1)
	}

	if err = (&controllers.SecurityGroupReconciler{
		Client:               client,
		Log:                  ctrl.Log.WithName("controllers").WithName("SecurityGroup"),
		SecurityGroupsClient: securitygroups.New(configuration),
		Reconciler: &controllers.AzureReconciler{
			Client:   client,
			Az:       securitygroups.New(configuration),
			Log:      log,
			Recorder: recorder,
		},
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "SecurityGroup")
		os.Exit(1)
	}

	if err = (&controllers.PublicIPReconciler{
		Client:          client,
		Log:             ctrl.Log.WithName("controllers").WithName("PublicIP"),
		PublicIPsClient: publicips.New(configuration),
		Reconciler: &controllers.AzureReconciler{
			Client:   client,
			Az:       publicips.New(configuration),
			Log:      log,
			Recorder: recorder,
		},
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "PublicIP")
		os.Exit(1)
	}

	if err = (&controllers.NetworkInterfaceReconciler{
		Client:     client,
		Log:        ctrl.Log.WithName("controllers").WithName("NetworkInterface"),
		NICsClient: nics.New(configuration),
		Reconciler: &controllers.AzureReconciler{
			Client:   client,
			Az:       nics.New(configuration),
			Log:      log,
			Recorder: recorder,
		},
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "NetworkInterface")
		os.Exit(1)
	}

	if err = (&controllers.TrafficManagerReconciler{
		Client:                client,
		Log:                   ctrl.Log.WithName("controllers").WithName("TrafficManager"),
		TrafficManagersClient: trafficmanagers.New(configuration),
		Recorder:              recorder,
	}).SetupWithManager(mgr); err != nil {
		litter.Dump(err)
		setupLog.Error(err, "unable to create controller", "controller", "TrafficManager")
		os.Exit(1)
	}

	if err = (&controllers.RedisReconciler{
		Client:      client,
		Log:         ctrl.Log.WithName("controllers").WithName("Redis"),
		RedisClient: redis.New(configuration, &client, scheme),
		Reconciler: &controllers.AzureReconciler{
			Client:   client,
			Az:       redis.New(configuration, &client, scheme),
			Log:      log,
			Recorder: recorder,
		},
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Redis")
		os.Exit(1)
	}

	if err = (&controllers.ServiceBusNamespaceReconciler{
		Client:                    client,
		Log:                       ctrl.Log.WithName("controllers").WithName("ServiceBusNamespace"),
		ServiceBusNamespaceClient: servicebus.New(configuration, &client, scheme),
		Reconciler: &controllers.AzureReconciler{
			Client:   client,
			Az:       servicebus.New(configuration, &client, scheme),
			Log:      log,
			Recorder: recorder,
		},
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ServiceBusNamespace")
		os.Exit(1)
	}

	if err = (&controllers.VMReconciler{
		Client:   client,
		Log:      ctrl.Log.WithName("controllers").WithName("VM"),
		VMClient: vms.New(configuration),
		Reconciler: &controllers.AzureReconciler{
			Client:   client,
			Az:       vms.New(configuration),
			Log:      log,
			Recorder: recorder,
		},
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
