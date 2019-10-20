package ensure

import (
	// "bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	// "text/template"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"github.com/sanity-io/litter"
	"github.com/spf13/cobra"
	appsv1 "k8s.io/api/apps/v1"
	extensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/alexeldeib/taskpool"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/controllers"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/dockercfg"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/identities"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/keyvaults"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/loadbalancers"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/nics"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/redis"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/rediskeys"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/resourcegroups"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/secretbundles"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/secrets"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/servicebus"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/servicebuskey"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/sqlservers"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/storagekeys"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/subnets"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/tlssecrets"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/trafficmanagers"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/virtualnetworks"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/vms"
	"github.com/alexeldeib/incendiary-iguana/pkg/config"
	"github.com/alexeldeib/incendiary-iguana/pkg/decoder"
)

const (
	limit           = 5
	backoffSteps    = 30
	backoffFactor   = 1.25
	backoffInterval = 5 * time.Second
	backoffJitter   = 1
	backoffLimit    = 900 * time.Second
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)
	_ = azurev1alpha1.AddToScheme(scheme)
	_ = extensionsv1beta1.AddToScheme(scheme)
	ctrl.SetLogger(zap.Logger(false))
}

func NewEnsureCommand() *cobra.Command {
	opts := &EnsureOptions{}
	cmd := &cobra.Command{
		Use:   "ensure",
		Short: "Ensure reconciles actual resource state to match desired",
		Run: func(cmd *cobra.Command, args []string) {
			if err := opts.Ensure(); err != nil {
				fmt.Printf("%+#v\n", err)
				os.Exit(1)
			}
		},
	}
	cmd.Flags().BoolVarP(&opts.Debug, "debug", "d", false, "Enable debug output")
	cmd.Flags().StringVarP(&opts.File, "file", "f", "", "File containt one or more Kubernetes manifests from a file containing multiple YAML documents (---)")
	cmd.Flags().StringVar(&opts.App, "AppId", "", "app id to authenticate with")
	cmd.Flags().StringVar(&opts.Key, "AppKey", "", "app key to authenticate with")
	cmd.Flags().StringVar(&opts.Tenant, "AppTenant", "", "tenant id to authenticate with")
	cmd.MarkFlagRequired("file")
	cmd.MarkFlagRequired("AppId")
	cmd.MarkFlagRequired("AppKey")
	cmd.MarkFlagRequired("AppTenant")
	return cmd
}

func NewDeleteCommand() *cobra.Command {
	opts := &EnsureOptions{}
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete enforces deletion of supplied resources.",
		Run: func(cmd *cobra.Command, args []string) {
			if err := opts.Delete(); err != nil {
				fmt.Printf("%+#v\n", err)
				os.Exit(1)
			}
		},
	}
	cmd.Flags().BoolVarP(&opts.Debug, "debug", "d", false, "Enable debug output")
	cmd.Flags().StringVarP(&opts.File, "file", "f", "", "File containt one or more Kubernetes manifests from a file containing multiple YAML documents (---)")
	cmd.Flags().StringVar(&opts.App, "AppId", "", "app id to authenticate with")
	cmd.Flags().StringVar(&opts.Key, "AppKey", "", "app key to authenticate with")
	cmd.Flags().StringVar(&opts.Tenant, "AppTenant", "", "tenant id to authenticate with")
	cmd.MarkFlagRequired("file")
	cmd.MarkFlagRequired("AppId")
	cmd.MarkFlagRequired("AppKey")
	cmd.MarkFlagRequired("AppTenant")
	return cmd
}

type EnsureOptions struct {
	File   string
	Debug  bool
	App    string
	Key    string
	Tenant string
}

func (opts *EnsureOptions) authorize() (*config.Config, error) {
	return config.New(
		config.App(opts.App),
		config.Key(opts.Key),
		config.Tenant(opts.Tenant),
	)
}

func (opts *EnsureOptions) Ensure() error {
	log := ctrl.Log.WithName("tinker")
	objects, err := opts.Read(log)
	if err != nil {
		return err
	}
	configuration, err := opts.authorize()
	if err != nil {
		return err
	}
	log.WithValues("App", opts.App, "Tenant", opts.Tenant, "KeyLen", len(opts.Key)).Info("args")
	return do(objects, configuration, Ensure, log)
}

func (opts *EnsureOptions) Delete() error {
	log := ctrl.Log.WithName("tinker")
	objects, err := opts.Read(log)
	if err != nil {
		return err
	}
	configuration, err := opts.authorize()
	if err != nil {
		return err
	}
	return do(objects, configuration, Delete, log)
}

func (opts *EnsureOptions) Read(log logr.Logger) ([]runtime.Object, error) {
	if opts.File == "" {
		return []runtime.Object{}, errors.New("must provide non-empty filepath")
	}

	var reader io.ReadCloser
	if opts.File == "-" {
		reader = ioutil.NopCloser(os.Stdin)
	} else {
		path, err := filepath.Abs(opts.File)
		if err != nil {
			return []runtime.Object{}, err
		}

		reader, err = os.Open(path)
		if err != nil {
			return []runtime.Object{}, err
		}
	}

	d := decoder.NewYAMLDecoder(reader, scheme)
	defer d.Close()

	// accumulators
	// gvks := []schema.GroupVersionKind{}
	objects := []runtime.Object{}

	// parsing
	for {
		obj, _, err := d.Decode(nil, nil)
		if err == io.EOF {
			break
		} else if err != nil {
			fmt.Printf("%+#v\n", err)
			return []runtime.Object{}, err
		}

		if runtime.IsNotRegisteredError(err) {
			fmt.Printf("%+#v\n", err)
			continue
		}

		if err != nil {
			return []runtime.Object{}, err
		}

		// actual, err := meta.Accessor(obj)
		// if err != nil {
		// 	return []runtime.Object{}, err
		// }

		// gvks = append(gvks, *gvk)
		objects = append(objects, obj)
	}

	if opts.Debug {
		log.V(1).Info("dumping manifests before applying")
		for _, obj := range objects {
			litter.Dump(obj)
		}
	}

	return objects, nil
}

func do(objects []runtime.Object, configuration *config.Config, applyFunc func(obj runtime.Object, configuration *config.Config, log logr.Logger) error, log logr.Logger) error {
	// apply objects
	tasks := []*taskpool.Task{}

	for key := range objects {
		// If you don't do this, you will end up ranging a non-deterministic subset of the array, duplicating some elements and missing others.
		val := objects[key] // This will get me in Go everytime.
		t := taskpool.NewTask(func() error {
			return applyFunc(val, configuration, log)
		})
		tasks = append(tasks, t)
	}

	pool := taskpool.NewPool(tasks, limit)

	log.Info("waiting for all tasks to complete")
	pool.Run()

	var numErrors int
	for _, task := range pool.Tasks {
		if task.Err != nil {
			log.Error(task.Err, "failed to reconcile object")
			numErrors++
		}
		if numErrors >= 10 {
			log.Info("Too many errors.")
			break
		}
	}

	if numErrors > 0 {
		return errors.New("one or more resources failed to deploy, check log output for further details")
	}
	return nil
}

func Ensure(obj runtime.Object, configuration *config.Config, log logr.Logger) error {
	var err error
	log = log.WithValues("action", "ensure", "type", obj.GetObjectKind().GroupVersionKind().String())
	log.Info("starting reconciliation")

	kubeclient, err := GetKubeclient()
	if err != nil {
		log.Error(err, "err with kubeclient")
		fmt.Printf("%#+v\n", err)
		return err
	}

	switch obj.(type) {
	case *appsv1.Deployment:
		log.Info("Deployment!")
	case *azurev1alpha1.DockerConfig:
		client, err := dockercfg.New(configuration, &kubeclient, scheme)
		if err != nil {
			log.Error(err, "got error with docker new client")
			fmt.Printf("%#+v\n", err.Error())
			break
		}
		err = EnsureSync(client, obj, log)
		if err != nil {
			log.Error(err, "got error with docker cfg")
			fmt.Printf("%#+v\n", err.Error())
		}
	case *azurev1alpha1.Identity:
		err = EnsureSync(identities.New(configuration), obj, log)
	case *azurev1alpha1.Keyvault:
		err = EnsureSync(keyvaults.New(configuration), obj, log)
	case *azurev1alpha1.LoadBalancer:
		err = EnsureAsync(loadbalancers.New(configuration), obj, log)
	case *azurev1alpha1.NetworkInterface:
		err = EnsureAsync(nics.New(configuration), obj, log)
	case *azurev1alpha1.Redis:
		err = EnsureAsync(redis.New(configuration, &kubeclient, scheme), obj, log)
	case *azurev1alpha1.ResourceGroup:
		err = EnsureAsync(resourcegroups.NewGroupClient(configuration), obj, log)
	case *azurev1alpha1.Secret:
		client, err := secrets.New(configuration, &kubeclient, scheme)
		if err != nil {
			fmt.Printf("%#+v\n", err)
		}
		err = EnsureSync(client, obj, log)
		if err != nil {
			fmt.Printf("%#+v\n", err)
		}
	case *azurev1alpha1.SecretBundle:
		client, err := secretbundles.New(configuration, &kubeclient, scheme)
		if err == nil {
			err = EnsureSync(client, obj, log)
		}
		if err != nil {
			fmt.Printf("%#+v\n", err)
		}
	case *azurev1alpha1.ServiceBusKey:
		err = EnsureSync(servicebuskey.New(configuration, &kubeclient, scheme), obj, log)
	case *azurev1alpha1.ServiceBusNamespace:
		err = EnsureAsync(servicebus.New(configuration, &kubeclient, scheme), obj, log)
	case *azurev1alpha1.SQLServer:
		err = EnsureSync(sqlservers.New(configuration, &kubeclient, scheme), obj, log)
	case *azurev1alpha1.StorageKey:
		err = EnsureSync(storagekeys.New(configuration, &kubeclient, scheme), obj, log)
	case *azurev1alpha1.Subnet:
		err = EnsureAsync(subnets.New(configuration), obj, log)
	case *azurev1alpha1.RedisKey:
		err = EnsureSync(rediskeys.New(configuration, &kubeclient, scheme), obj, log)
	case *azurev1alpha1.TLSSecret:
		client, err := tlssecrets.New(configuration, &kubeclient, scheme)
		if err == nil {
			err = EnsureSync(client, obj, log)
		}
		if err != nil {
			fmt.Printf("%#+v\n", err)
		}
	case *azurev1alpha1.TrafficManager:
		err = EnsureTrafficManager(trafficmanagers.New(configuration), obj, log)
	case *azurev1alpha1.VirtualNetwork:
		err = EnsureAsync(virtualnetworks.New(configuration), obj, log)
	case *azurev1alpha1.VM:
		err = EnsureAsync(vms.New(configuration), obj, log)
	default:
		log.Info("nothing to do.")
	}
	if err != nil {
		log.Info("failed to reconcile")
		fmt.Printf("%#+v\n", err)
		fmt.Printf("%s\n", err.Error())
		return err
	}
	log.Info("successfully reconciled")
	return nil
}

func Delete(obj runtime.Object, configuration *config.Config, log logr.Logger) error {
	log = log.WithValues("action", "delete", "type", obj.GetObjectKind().GroupVersionKind().String())
	log.Info("starting deletion")

	kubeclient, err := GetKubeclient()
	if err != nil {
		fmt.Printf("%#+v\n", err)
		return err
	}

	switch obj.(type) {
	case *appsv1.Deployment:
		log.Info("Deployment!")
	case *azurev1alpha1.DockerConfig:
		client, err := dockercfg.New(configuration, &kubeclient, scheme)
		if err == nil {
			err = DeleteSync(client, obj, log)
		}
	case *azurev1alpha1.Identity:
		err = DeleteSync(identities.New(configuration), obj, log)
	case *azurev1alpha1.Keyvault:
		err = DeleteSync(keyvaults.New(configuration), obj, log)
	case *azurev1alpha1.LoadBalancer:
		err = DeleteAsync(loadbalancers.New(configuration), obj, log)
	case *azurev1alpha1.NetworkInterface:
		err = DeleteAsync(nics.New(configuration), obj, log)
	case *azurev1alpha1.Redis:
		err = DeleteAsync(redis.New(configuration, &kubeclient, scheme), obj, log)
	case *azurev1alpha1.ResourceGroup:
		err = DeleteAsync(resourcegroups.NewGroupClient(configuration), obj, log)
	case *azurev1alpha1.Secret:
		client, err := secrets.New(configuration, &kubeclient, scheme)
		if err == nil {
			err = DeleteSync(client, obj, log)
		}
	case *azurev1alpha1.SecretBundle:
		client, err := secretbundles.New(configuration, &kubeclient, scheme)
		if err == nil {
			err = DeleteSync(client, obj, log)
		}
	case *azurev1alpha1.ServiceBusNamespace:
		err = DeleteAsync(servicebus.New(configuration, &kubeclient, scheme), obj, log)
	case *azurev1alpha1.SQLServer:
		err = DeleteSync(sqlservers.New(configuration, &kubeclient, scheme), obj, log)
	case *azurev1alpha1.StorageKey:
		err = DeleteSync(storagekeys.New(configuration, &kubeclient, scheme), obj, log)
	case *azurev1alpha1.Subnet:
		err = DeleteAsync(subnets.New(configuration), obj, log)
	case *azurev1alpha1.TLSSecret:
		client, err := tlssecrets.New(configuration, &kubeclient, scheme)
		if err == nil {
			err = DeleteSync(client, obj, log)
		}
	case *azurev1alpha1.TrafficManager:
		log.Info("Traffic Manager!")
	case *azurev1alpha1.VirtualNetwork:
		err = DeleteAsync(virtualnetworks.New(configuration), obj, log)
	case *azurev1alpha1.VM:
		err = DeleteAsync(vms.New(configuration), obj, log)
	default:
		log.Info("nothing to do.")
	}
	if err != nil {
		log.Info("failed to delete")
		fmt.Printf("%s\n", err.Error())
		return err
	}
	log.Info("successfully deleted")
	return nil
}

func EnsureSync(client controllers.SyncClient, obj runtime.Object, log logr.Logger) error {
	local, ok := obj.(metav1.Object)
	if !ok {
		return errors.New("failed type assertion after switching on type. check switch statement and function invocation.")
	}

	log = log.WithValues("type", obj.GetObjectKind().GroupVersionKind().String(), "namespace", local.GetNamespace(), "name", local.GetName())

	// extract. consider keyvault and non-sub specific clients. Matrix size = 2x2 (async, sub)
	if err := client.ForSubscription(context.Background(), obj); err != nil {
		return errors.Wrap(err, "failed to get client for subscription")
	}

	var err error

	// extract this into async/sync, probably
	return wait.ExponentialBackoff(backoff(), func() (bool, error) {
		log.Info("reconciling")
		err = client.Ensure(context.Background(), obj)
		if err != nil {
			log.Error(err, "failed reconcile attempt")
		}
		return err == nil, nil
	})
}

func DeleteSync(client controllers.SyncClient, obj runtime.Object, log logr.Logger) error {
	local, ok := obj.(metav1.Object)
	if !ok {
		return errors.New("failed type assertion after switching on type. check switch statement and function invocation.")
	}

	log = log.WithValues("type", obj.GetObjectKind().GroupVersionKind().String(), "namespace", local.GetNamespace(), "name", local.GetName())

	// extract. consider keyvault and non-sub specific clients. Matrix size = 2x2 (async, sub)
	if err := client.ForSubscription(context.Background(), obj); err != nil {
		return errors.Wrap(err, "failed to get client for subscription")
	}

	// extract this into async/sync, probably
	return wait.ExponentialBackoff(backoff(), func() (done bool, err error) {
		log.Info("reconciling")
		err = client.Delete(context.Background(), obj, log)
		if err != nil {
			log.Error(err, "failed reconcile attempt")
		}
		return err == nil, nil
	})
}

func EnsureAsync(client controllers.AsyncClient, obj runtime.Object, log logr.Logger) error {
	local, ok := obj.(metav1.Object)
	if !ok {
		return errors.New("failed type assertion after switching on type. check switch statement and function invocation.")
	}

	log = log.WithValues("type", obj.GetObjectKind().GroupVersionKind().String(), "namespace", local.GetNamespace(), "name", local.GetName())

	if err := client.ForSubscription(context.Background(), obj); err != nil {
		return errors.Wrap(err, "failed to get client for subscription")
	}

	return wait.ExponentialBackoff(backoff(), func() (done bool, err error) {
		log.Info("reconciling")
		done, err = client.Ensure(context.Background(), obj)
		if err != nil {
			log.Error(err, "failed reconcile attempt")
		}
		return done, nil
	})
}

func DeleteAsync(client controllers.AsyncClient, obj runtime.Object, log logr.Logger) error {
	local, ok := obj.(metav1.Object)
	if !ok {
		return errors.New("failed type assertion after switching on type. check switch statement and function invocation.")
	}

	log = log.WithValues("type", obj.GetObjectKind().GroupVersionKind().String(), "namespace", local.GetNamespace(), "name", local.GetName())

	if err := client.ForSubscription(context.Background(), obj); err != nil {
		return errors.Wrap(err, "failed to get client for subscription")
	}

	return wait.ExponentialBackoff(backoff(), func() (done bool, err error) {
		log.Info("reconciling")
		found, err := client.Delete(context.Background(), obj, log)
		if err != nil {
			log.Error(err, "failed reconcile attempt")
		}
		return !found, nil
	})
}

func EnsureTrafficManager(client *trafficmanagers.Client, obj runtime.Object, log logr.Logger) error {
	local, ok := obj.(*azurev1alpha1.TrafficManager)
	if !ok {
		return errors.New("failed type assertion after switching on type. check switch statement and function invocation.")
	}

	log = log.WithValues("type", "trafficmanager", "name", local.Spec.Name)

	if err := client.ForSubscription(local.Spec.SubscriptionID); err != nil {
		return errors.Wrap(err, "failed to get client for subscription")
	}

	return wait.ExponentialBackoff(backoff(), func() (done bool, err error) {
		log.Info("reconciling")
		if _, err := client.Ensure(context.Background(), local); err != nil {
			return false, errors.Wrap(err, "failed to reconcile")
		}
		status, err := client.GetProfileStatus(context.Background(), local)
		log.Info("waiting for appropriate status", "status", status)
		if err != nil {
			return false, errors.Wrap(err, "failed to get monitor status")
		}
		if status == "Online" || status == "Disabled" {
			return true, nil
		}
		return false, nil
	})
}

func DeleteTrafficManager(client *trafficmanagers.Client, obj runtime.Object, log logr.Logger) error {
	local, ok := obj.(*azurev1alpha1.TrafficManager)
	if !ok {
		return errors.New("failed type assertion after switching on type. check switch statement and function invocation.")
	}

	log = log.WithValues("type", "trafficmanager", "name", local.Spec.Name)

	if err := client.ForSubscription(local.Spec.SubscriptionID); err != nil {
		return errors.Wrap(err, "failed to get client for subscription")
	}

	return wait.ExponentialBackoff(backoff(), func() (done bool, err error) {
		log.Info("deleting")
		// n.b.: returning true *should* allow failing with an error.
		// implementation:
		// https://github.com/kubernetes/apimachinery/blob/461753078381c979582f217a28eb759ebee5295d/pkg/util/wait/wait.go#L290-L301
		return true, client.Delete(context.Background(), local)
	})
}

func backoff() wait.Backoff {
	return wait.Backoff{
		Cap:      backoffLimit,
		Steps:    backoffSteps,
		Factor:   backoffFactor,
		Duration: backoffInterval,
		Jitter:   backoffJitter,
	}
}

func GetKubeclient() (client.Client, error) {
	var (
		kubeconfig *rest.Config
		err        error
	)

	if kubeconfig, err = ctrl.GetConfig(); err != nil {
		return nil, err
	}

	return client.New(kubeconfig, client.Options{})
}
