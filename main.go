/*
Copyright 2019 Alexander Eldeib.
*/

package main

import (
	"errors"
	"flag"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/Azure/go-autorest/autorest/azure"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/controllers"
	"github.com/alexeldeib/incendiary-iguana/pkg/authorizer"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients"
	"github.com/alexeldeib/incendiary-iguana/pkg/reconcilers"
	"github.com/alexeldeib/incendiary-iguana/pkg/reconcilers/generic"
	"github.com/alexeldeib/incendiary-iguana/pkg/services"
	// +kubebuilder:scaffold:imports
)

var (
	scheme         = runtime.NewScheme()
	setupLog       = ctrl.Log.WithName("setup")
	controllerName = "incendiaryiguana"
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)
	_ = azurev1alpha1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

func main() {
	rand.Seed(time.Now().Unix())
	ctrl.SetLogger(zap.Logger(false))
	setupLog.Info("starting manager")

	var (
		metricsAddr           string
		enableLeaderElection  bool
		app, key, tenant, env string
	)
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&app, "app", "", "The AAD app ID for authentication.")
	flag.StringVar(&key, "key", "", "The AAD client secret for authentication.")
	flag.StringVar(&tenant, "tenant", "", "The AAD tenant ID for authentication.")
	flag.StringVar(&env, "env", "AzurePublicCloud", "The Azure cloud environment. options include AzurePublicCloud, ...")
	flag.Parse()

	if app == "" || key == "" || tenant == "" {
		setupLog.Error(errors.New("must specify all of app, key, and tenant for client credential authentication"), "", "app", app, "tenant", tenant, "key length", len(key))
		os.Exit(1)
	}

	// Must have some cloud set. Default to Azure public cloud.
	var cloud azure.Environment
	var err error
	if env == "" {
		cloud = azure.PublicCloud
	} else {
		c, err := azure.EnvironmentFromName(env)
		if err != nil {
			setupLog.Error(err, "failed to create azure environment from user-provided value", "env", env)
			os.Exit(1)
		}
		cloud = c
	}

	// Fetch kubeconfig
	kubeconfig, err := ctrl.GetConfig()
	if err != nil {
		setupLog.Error(err, "failed to get kubeconfig")
		os.Exit(1)
	}

	// Setup manager
	mgr, err := ctrl.NewManager(kubeconfig, ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		LeaderElection:     enableLeaderElection,
	})

	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Initialize shared logger, event recorder, and kubeclient
	log := ctrl.Log.WithName(controllerName)
	recorder := mgr.GetEventRecorderFor(controllerName)
	client := mgr.GetClient()

	// Initialize Azure authorizers.
	mgmtAuthorizer, err := authorizer.New(authorizer.ClientCredentials(app, key, tenant))
	if err != nil {
		setupLog.Error(err, "failed to create arm authorizer")
	}

	// Keyvault data plane uses a separate AAD resource for the token.
	// There are only ~six of these, so it's worth making retrieving all of them easy.
	kvAuthorizer, err := authorizer.New(
		authorizer.Resource(strings.TrimSuffix(cloud.KeyVaultEndpoint, "/")),
		authorizer.ClientCredentials(app, key, tenant),
	)

	if err != nil {
		setupLog.Error(err, "failed to create keyvault data plane authorizer")
	}

	// Create Azure service clients. Thin wrappers on Azure SDK with mockable interfaces.
	groupService := &services.ResourceGroupService{
		Authorizer: mgmtAuthorizer,
	}

	secretService := &services.SecretService{
		DNSSuffix: cloud.KeyVaultDNSSuffix,
		Client:    clients.NewSecretsClient(kvAuthorizer),
	}

	storageService := &services.StorageService{
		Authorizer: mgmtAuthorizer,
	}

	trafficManagerService := &services.TrafficManagerService{
		Authorizer: mgmtAuthorizer,
	}

	// Initialize controllers
	if err = (&controllers.ResourceGroupController{
		Reconciler: &generic.AsyncReconciler{
			Client:   client,
			Logger:   log,
			Recorder: recorder,
			Scheme:   scheme,
			AsyncActuator: &reconcilers.ResourceGroupReconciler{
				Service:    groupService,
				Scheme:     scheme,
			},
		},
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Secret")
		os.Exit(1)
	}

	if err = (&controllers.SecretController{
		Reconciler: &generic.SyncReconciler{
			Client:   client,
			Logger:   log,
			Recorder: recorder,
			Scheme:   scheme,
			SyncActuator: &reconcilers.SecretReconciler{
				Service:    secretService,
				Kubeclient: client,
				Scheme:     scheme,
			},
		},
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Secret")
		os.Exit(1)
	}

	if err = (&controllers.SecretBundleController{
		Reconciler: &generic.SyncReconciler{
			Client:   client,
			Logger:   log,
			Recorder: recorder,
			Scheme:   scheme,
			SyncActuator: &reconcilers.SecretBundleReconciler{
				Service:    secretService,
				Kubeclient: client,
				Scheme:     scheme,
			},
		},
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "SecretBundle")
		os.Exit(1)
	}

	if err = (&controllers.StorageKeyController{
		Reconciler: &generic.SyncReconciler{
			Client:   client,
			Logger:   log,
			Recorder: recorder,
			Scheme:   scheme,
			SyncActuator: &reconcilers.StorageKeyReconciler{
				Service:    storageService,
				Kubeclient: client,
				Scheme:     scheme,
			},
		},
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "StorageKey")
		os.Exit(1)
	}

	if err = (&controllers.TrafficManagerController{
		Reconciler: &generic.AsyncReconciler{
			Client:   client,
			Logger:   log,
			Recorder: recorder,
			Scheme:   scheme,
			AsyncActuator: &reconcilers.TrafficManagerReconciler{
				Service: trafficManagerService,
			},
		},
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "TrafficManager")
		os.Exit(1)
	}

	// Start manager
	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
