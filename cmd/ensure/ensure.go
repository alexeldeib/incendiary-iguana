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
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/identities"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/keyvaults"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/loadbalancers"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/nics"
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
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/vms"
	"github.com/alexeldeib/incendiary-iguana/pkg/config"
	"github.com/alexeldeib/incendiary-iguana/pkg/decoder"
)

type token struct{}

const (
	limit           = 1
	backoffSteps    = 30
	backoffFactor   = 1.25
	backoffInterval = 5 * time.Second
	backoffJitter   = 1
	backoffLimit    = 900 * time.Second
)

var (
	log    = ctrl.Log.WithName("tinker")
	scheme = runtime.NewScheme()
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)
	_ = azurev1alpha1.AddToScheme(scheme)

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
	cmd.MarkFlagRequired("file")
	return cmd
}

func NewDeleteCommand() *cobra.Command {
	opts := &EnsureOptions{}
	cmd := &cobra.Command{
		Use:   "ensure",
		Short: "Ensure reconciles actual resource state to match desired",
		Run: func(cmd *cobra.Command, args []string) {
			if err := opts.Delete(); err != nil {
				fmt.Printf("%+#v\n", err)
				os.Exit(1)
			}
		},
	}
	cmd.Flags().BoolVarP(&opts.Debug, "debug", "d", false, "Enable debug output")
	cmd.Flags().StringVarP(&opts.File, "file", "f", "", "File containt one or more Kubernetes manifests from a file containing multiple YAML documents (---)")
	cmd.MarkFlagRequired("file")
	return cmd
}

type EnsureOptions struct {
	File  string
	Debug bool
}

func authorize() (*config.Config, error) {
	c := config.New(log)
	return c, c.DetectAuthorizer()
}

func (opts *EnsureOptions) Ensure() error {
	objects, err := opts.Read()
	if err != nil {
		return err
	}
	configuration, err := authorize()
	if err != nil {
		return err
	}
	return do(objects, configuration, Ensure)
}

func (opts *EnsureOptions) Delete() error {
	objects, err := opts.Read()
	if err != nil {
		return err
	}
	configuration, err := authorize()
	if err != nil {
		return err
	}
	return do(objects, configuration, Delete)
}

func (opts *EnsureOptions) Read() ([]metav1.Object, error) {
	if opts.File == "" {
		return []metav1.Object{}, errors.New("must provide non-empty filepath")
	}

	var reader io.ReadCloser
	if opts.File == "-" {
		reader = ioutil.NopCloser(os.Stdin)
	} else {
		path, err := filepath.Abs(opts.File)
		if err != nil {
			return []metav1.Object{}, err
		}

		reader, err = os.Open(path)
		if err != nil {
			return []metav1.Object{}, err
		}
	}

	d := decoder.NewYAMLDecoder(reader, scheme)
	defer d.Close()

	// accumulators
	// gvks := []schema.GroupVersionKind{}
	objects := []metav1.Object{}

	// parsing
	for {
		obj, _, err := d.Decode(nil, nil)
		if err == io.EOF {
			break
		} else if err != nil {
			fmt.Printf("%+#v\n", err)
			return []metav1.Object{}, err
		}

		if runtime.IsNotRegisteredError(err) {
			fmt.Printf("%+#v\n", err)
			continue
		}

		if err != nil {
			return []metav1.Object{}, err
		}

		actual, err := meta.Accessor(obj)
		if err != nil {
			return []metav1.Object{}, err
		}

		// gvks = append(gvks, *gvk)
		objects = append(objects, actual)
	}

	if opts.Debug {
		log.V(1).Info("dumping manifests before applying")
		for _, obj := range objects {
			litter.Dump(obj)
		}
	}

	return objects, nil
}

func do(objects []metav1.Object, configuration *config.Config, applyFunc func(obj metav1.Object, configuration *config.Config, kubeclient *client.Client, errs chan error)) error {
	var kubeclient *client.Client
	kubeconfig, err := ctrl.GetConfig()
	if err != nil {
		log.Error(err, "failed to get kubeconfig")
		kubeconfig = nil
	} else {
		c, err := client.New(kubeconfig, client.Options{})
		if err != nil {
			log.Error(err, "expected to create kubeclient after finding kubeconfig")
		} else {
			kubeclient = &c
		}
	}
	// apply objects
	pool := make(chan token, limit)
	errs := make(chan error, limit)
	for _, obj := range objects {
		// n.b. acquire
		pool <- token{}
		go func(o metav1.Object) {
			applyFunc(o, configuration, kubeclient, errs)
			// n.b. release
			<-pool
		}(obj)
	}

	// n.b. wait for task completion
	for n := limit; n > 0; n-- {
		pool <- token{}
	}

	select {
	case err := <-errs:
		return err
	default:
	}

	return nil
}

func Ensure(obj metav1.Object, configuration *config.Config, kubeclient *client.Client, errs chan error) {
	var err error
	log := log.WithValues("action", "ensure")
	log.Info("starting reconciliation")
	switch obj.(type) {
	case *appsv1.Deployment:
		log.Info("Deployment!")
	case *azurev1alpha1.Identity:
		err = EnsureIdentity(identities.New(configuration), obj, log)
	case *azurev1alpha1.Keyvault:
		err = EnsureKeyvault(keyvaults.New(configuration), obj, log)
	case *azurev1alpha1.LoadBalancer:
		err = EnsureLoadBalancer(loadbalancers.New(configuration), obj, log)
	case *azurev1alpha1.NetworkInterface:
		err = EnsureNIC(nics.New(configuration), obj, log)
	case *azurev1alpha1.Redis:
		err = EnsureRedis(redis.New(configuration, nil, nil), obj, log)
	case *azurev1alpha1.ResourceGroup:
		err = EnsureResourceGroup(resourcegroups.New(configuration), obj, log)
	case *azurev1alpha1.Secret:
		client, err := secrets.New(configuration, nil, nil)
		if err == nil {
			err = EnsureSecret(client, obj, log)
		}
	case *azurev1alpha1.SecretBundle:
		client, err := secretbundles.New(configuration, nil, nil)
		if err == nil {
			err = EnsureSecretBundle(client, obj, log)
		}
	case *azurev1alpha1.ServiceBusNamespace:
		err = EnsureServiceBusNamespace(servicebus.New(configuration, nil, nil), obj, log)
	case *azurev1alpha1.Subnet:
		err = EnsureSubnet(subnets.New(configuration), obj, log)
	case *azurev1alpha1.TrafficManager:
		err = EnsureTrafficManager(trafficmanagers.New(configuration), obj, log)
	case *azurev1alpha1.VirtualNetwork:
		err = EnsureVirtualNetwork(virtualnetworks.New(configuration), obj, log)
	case *azurev1alpha1.VM:
		err = EnsureVM(vms.New(configuration), obj, log)
	default:
		log.Info("nothing to do.")
	}
	if err != nil {
		errs <- err
		log.Info("failed to reconcile")
		fmt.Printf("%s\n", err.Error())
		return
	}
	log.Info("sucessfully reconciled")
}

func Delete(obj metav1.Object, configuration *config.Config, kubeclient *client.Client, errs chan error) {
	var err error
	log := log.WithValues("action", "delete")
	log.Info("starting deletion")
	switch obj.(type) {
	case *appsv1.Deployment:
		log.Info("Deployment!")
	case *azurev1alpha1.Identity:
		err = DeleteIdentity(identities.New(configuration), obj, log)
	case *azurev1alpha1.Keyvault:
		err = DeleteKeyvault(keyvaults.New(configuration), obj, log)
	case *azurev1alpha1.LoadBalancer:
		err = DeleteLoadBalancer(loadbalancers.New(configuration), obj, log)
	case *azurev1alpha1.NetworkInterface:
		err = DeleteNIC(nics.New(configuration), obj, log)
	case *azurev1alpha1.Redis:
		err = DeleteRedis(redis.New(configuration, nil, nil), obj, log)
	case *azurev1alpha1.ResourceGroup:
		err = DeleteResourceGroup(resourcegroups.New(configuration), obj, log)
	case *azurev1alpha1.Secret:
		client, err := secrets.New(configuration, nil, nil)
		if err == nil {
			err = DeleteSecret(client, obj, log)
		}
	case *azurev1alpha1.SecretBundle:
		client, err := secretbundles.New(configuration, nil, nil)
		if err == nil {
			err = DeleteSecretBundle(client, obj, log)
		}
	case *azurev1alpha1.ServiceBusNamespace:
		err = DeleteServiceBusNamespace(servicebus.New(configuration, nil, nil), obj, log)
	case *azurev1alpha1.Subnet:
		err = DeleteSubnet(subnets.New(configuration), obj, log)
	case *azurev1alpha1.TrafficManager:
		log.Info("Traffic Manager!")
	case *azurev1alpha1.VirtualNetwork:
		err = DeleteVirtualNetwork(virtualnetworks.New(configuration), obj, log)
	case *azurev1alpha1.VM:
		err = DeleteVM(vms.New(configuration), obj, log)
	default:
		log.Info("nothing to do.")
	}
	if err != nil {
		errs <- err
		log.Info("failed to delete")
		fmt.Printf("%s\n", err.Error())
		return
	}
	log.Info("sucessfully deleted")
}

// TODO(ace): extract this pattern for async/sync resources (aka, things that return "done" vs things that only return err value)
// generalize it across resources, natively if possible or by defining some interface
func EnsureResourceGroup(client *resourcegroups.Client, obj metav1.Object, log logr.Logger) error {
	// TODO(ace): simplify the typecasting and clint
	local, ok := obj.(*azurev1alpha1.ResourceGroup)
	if !ok {
		return errors.New("failed type assertion after switching on type. check switch statement and function invocation.")
	}

	log = log.WithValues("type", "resourcegroup", "name", local.Spec.Name)

	// extract. consider keyvault and non-sub specific clients. Matrix size = 2x2 (async, sub)
	if err := client.ForSubscription(local.Spec.SubscriptionID); err != nil {
		return errors.Wrap(err, "failed to get client for subscription")
	}

	// extract this into async/sync, probably
	return wait.ExponentialBackoff(backoff(), func() (done bool, err error) {
		log.Info("reconciling")
		done, err = client.Ensure(context.Background(), local)
		if err != nil {
			log.Error(err, "failed reconcile attempt")
		}
		return done, nil
	})
}

func DeleteResourceGroup(client *resourcegroups.Client, obj metav1.Object, log logr.Logger) error {
	local, ok := obj.(*azurev1alpha1.ResourceGroup)
	if !ok {
		return errors.New("failed type assertion after switching on type. check switch statement and function invocation.")
	}

	log = log.WithValues("type", "resourcegroup", "name", local.Spec.Name)

	if err := client.ForSubscription(local.Spec.SubscriptionID); err != nil {
		return errors.Wrap(err, "failed to get client for subscription")
	}

	return wait.ExponentialBackoff(backoff(), func() (done bool, err error) {
		log.Info("deleting")
		found, err := client.Delete(context.Background(), local)
		return !found, err
	})
}

func EnsureVirtualNetwork(client *virtualnetworks.Client, obj metav1.Object, log logr.Logger) error {
	local, ok := obj.(*azurev1alpha1.VirtualNetwork)
	if !ok {
		return errors.New("failed type assertion after switching on type. check switch statement and function invocation.")
	}

	log = log.WithValues("type", "virtualnetwork", "name", local.Spec.Name)

	if err := client.ForSubscription(local.Spec.SubscriptionID); err != nil {
		return errors.Wrap(err, "failed to get client for subscription")
	}

	return wait.ExponentialBackoff(backoff(), func() (done bool, err error) {
		log.Info("reconciling")
		done, err = client.Ensure(context.Background(), local)
		if err != nil {
			log.Error(err, "failed reconcile attempt")
		}
		return done, nil
	})
}

func DeleteVirtualNetwork(client *virtualnetworks.Client, obj metav1.Object, log logr.Logger) error {
	local, ok := obj.(*azurev1alpha1.VirtualNetwork)
	if !ok {
		return errors.New("failed type assertion after switching on type. check switch statement and function invocation.")
	}

	log = log.WithValues("type", "virtualnetwork", "name", local.Spec.Name)

	if err := client.ForSubscription(local.Spec.SubscriptionID); err != nil {
		return errors.Wrap(err, "failed to get client for subscription")
	}

	return wait.ExponentialBackoff(backoff(), func() (done bool, err error) {
		log.Info("deleting")
		found, err := client.Delete(context.Background(), local)
		return !found, err
	})
}

func EnsureTrafficManager(client *trafficmanagers.Client, obj metav1.Object, log logr.Logger) error {
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

func DeleteTrafficManager(client *trafficmanagers.Client, obj metav1.Object, log logr.Logger) error {
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

func EnsureSubnet(client *subnets.Client, obj metav1.Object, log logr.Logger) error {
	local, ok := obj.(*azurev1alpha1.Subnet)
	if !ok {
		return errors.New("failed type assertion after switching on type. check switch statement and function invocation.")
	}

	log = log.WithValues("type", "subnet", "name", local.Spec.Name)

	if err := client.ForSubscription(local.Spec.SubscriptionID); err != nil {
		return errors.Wrap(err, "failed to get client for subscription")
	}

	return wait.ExponentialBackoff(backoff(), func() (done bool, err error) {
		log.Info("reconciling")
		done, err = client.Ensure(context.Background(), local)
		if err != nil {
			log.Error(err, "failed reconcile attempt")
		}
		return done, nil
	})
}

func DeleteSubnet(client *subnets.Client, obj metav1.Object, log logr.Logger) error {
	local, ok := obj.(*azurev1alpha1.Subnet)
	if !ok {
		return errors.New("failed type assertion after switching on type. check switch statement and function invocation.")
	}

	log = log.WithValues("type", "subnet", "name", local.Spec.Name)

	if err := client.ForSubscription(local.Spec.SubscriptionID); err != nil {
		return errors.Wrap(err, "failed to get client for subscription")
	}

	return wait.ExponentialBackoff(backoff(), func() (done bool, err error) {
		log.Info("deleting")
		found, err := client.Delete(context.Background(), local)
		return !found, err
	})
}

func EnsurePublicIP(client *publicips.Client, obj metav1.Object, log logr.Logger) error {
	local, ok := obj.(*azurev1alpha1.PublicIP)
	if !ok {
		return errors.New("failed type assertion after switching on type. check switch statement and function invocation.")
	}

	log = log.WithValues("type", "publicip", "name", local.Spec.Name)

	if err := client.ForSubscription(local.Spec.SubscriptionID); err != nil {
		return errors.Wrap(err, "failed to get client for subscription")
	}

	return wait.ExponentialBackoff(backoff(), func() (done bool, err error) {
		log.Info("reconciling")
		done, err = client.Ensure(context.Background(), local)
		if err != nil {
			log.Error(err, "failed reconcile attempt")
		}
		return done, nil
	})
}

func DeletePublicIP(client *publicips.Client, obj metav1.Object, log logr.Logger) error {
	local, ok := obj.(*azurev1alpha1.PublicIP)
	if !ok {
		return errors.New("failed type assertion after switching on type. check switch statement and function invocation.")
	}

	log = log.WithValues("type", "publicip", "name", local.Spec.Name)

	if err := client.ForSubscription(local.Spec.SubscriptionID); err != nil {
		return errors.Wrap(err, "failed to get client for subscription")
	}

	return wait.ExponentialBackoff(backoff(), func() (done bool, err error) {
		log.Info("deleting")
		found, err := client.Delete(context.Background(), local)
		return !found, err
	})
}

func EnsureSecurityGroup(client *securitygroups.Client, obj metav1.Object, log logr.Logger) error {
	local, ok := obj.(*azurev1alpha1.SecurityGroup)
	if !ok {
		return errors.New("failed type assertion after switching on type. check switch statement and function invocation.")
	}

	log = log.WithValues("type", "securitygroup", "name", local.Spec.Name)

	if err := client.ForSubscription(local.Spec.SubscriptionID); err != nil {
		return errors.Wrap(err, "failed to get client for subscription")
	}

	return wait.ExponentialBackoff(backoff(), func() (done bool, err error) {
		log.Info("reconciling")
		done, err = client.Ensure(context.Background(), local)
		if err != nil {
			log.Error(err, "failed reconcile attempt")
		}
		return done, nil
	})
}

func DeleteSecurityGroup(client *securitygroups.Client, obj metav1.Object, log logr.Logger) error {
	local, ok := obj.(*azurev1alpha1.SecurityGroup)
	if !ok {
		return errors.New("failed type assertion after switching on type. check switch statement and function invocation.")
	}

	log = log.WithValues("type", "securitygroup", "name", local.Spec.Name)

	if err := client.ForSubscription(local.Spec.SubscriptionID); err != nil {
		return errors.Wrap(err, "failed to get client for subscription")
	}

	return wait.ExponentialBackoff(backoff(), func() (done bool, err error) {
		log.Info("deleting")
		found, err := client.Delete(context.Background(), local)
		return !found, err
	})
}

func EnsureRedis(client *redis.Client, obj metav1.Object, log logr.Logger) error {
	local, ok := obj.(*azurev1alpha1.Redis)
	if !ok {
		return errors.New("failed type assertion after switching on type. check switch statement and function invocation.")
	}

	log = log.WithValues("type", "redis", "name", local.Spec.Name)

	if err := client.ForSubscription(local.Spec.SubscriptionID); err != nil {
		return errors.Wrap(err, "failed to get client for subscription")
	}

	return wait.ExponentialBackoff(backoff(), func() (done bool, err error) {
		log.Info("reconciling")
		done, err = client.Ensure(context.Background(), local)
		if err != nil {
			log.Error(err, "failed reconcile attempt")
		}
		return done, nil
	})
}

func DeleteRedis(client *redis.Client, obj metav1.Object, log logr.Logger) error {
	local, ok := obj.(*azurev1alpha1.Redis)
	if !ok {
		return errors.New("failed type assertion after switching on type. check switch statement and function invocation.")
	}

	log = log.WithValues("type", "redis", "name", local.Spec.Name)

	if err := client.ForSubscription(local.Spec.SubscriptionID); err != nil {
		return errors.Wrap(err, "failed to get client for subscription")
	}

	return wait.ExponentialBackoff(backoff(), func() (done bool, err error) {
		log.Info("deleting")
		found, err := client.Delete(context.Background(), local)
		return !found, err
	})
}

func EnsureNIC(client *nics.Client, obj metav1.Object, log logr.Logger) error {
	local, ok := obj.(*azurev1alpha1.NetworkInterface)
	if !ok {
		return errors.New("failed type assertion after switching on type. check switch statement and function invocation.")
	}

	log = log.WithValues("type", "nics", "name", local.Spec.Name)

	if err := client.ForSubscription(local.Spec.SubscriptionID); err != nil {
		return errors.Wrap(err, "failed to get client for subscription")
	}

	return wait.ExponentialBackoff(backoff(), func() (done bool, err error) {
		log.Info("reconciling")
		done, err = client.Ensure(context.Background(), local)
		if err != nil {
			log.Error(err, "failed reconcile attempt")
		}
		return done, nil
	})
}

func DeleteNIC(client *nics.Client, obj metav1.Object, log logr.Logger) error {
	local, ok := obj.(*azurev1alpha1.NetworkInterface)
	if !ok {
		return errors.New("failed type assertion after switching on type. check switch statement and function invocation.")
	}

	log = log.WithValues("type", "nics", "name", local.Spec.Name)

	if err := client.ForSubscription(local.Spec.SubscriptionID); err != nil {
		return errors.Wrap(err, "failed to get client for subscription")
	}

	return wait.ExponentialBackoff(backoff(), func() (done bool, err error) {
		log.Info("deleting")
		found, err := client.Delete(context.Background(), local)
		return !found, err
	})
}

func EnsureServiceBusNamespace(client *servicebus.Client, obj metav1.Object, log logr.Logger) error {
	local, ok := obj.(*azurev1alpha1.ServiceBusNamespace)
	if !ok {
		return errors.New("failed type assertion after switching on type. check switch statement and function invocation.")
	}

	log = log.WithValues("type", "servicebus", "name", local.Spec.Name)

	if err := client.ForSubscription(local.Spec.SubscriptionID); err != nil {
		return errors.Wrap(err, "failed to get client for subscription")
	}

	return wait.ExponentialBackoff(backoff(), func() (done bool, err error) {
		log.Info("reconciling")
		done, err = client.Ensure(context.Background(), local)
		if err != nil {
			log.Error(err, "failed reconcile attempt")
		}
		return done, nil
	})
}

func DeleteServiceBusNamespace(client *servicebus.Client, obj metav1.Object, log logr.Logger) error {
	local, ok := obj.(*azurev1alpha1.ServiceBusNamespace)
	if !ok {
		return errors.New("failed type assertion after switching on type. check switch statement and function invocation.")
	}

	log = log.WithValues("type", "servicebus", "name", local.Spec.Name)

	if err := client.ForSubscription(local.Spec.SubscriptionID); err != nil {
		return errors.Wrap(err, "failed to get client for subscription")
	}

	return wait.ExponentialBackoff(backoff(), func() (done bool, err error) {
		log.Info("deleting")
		found, err := client.Delete(context.Background(), local)
		return !found, err
	})
}

func EnsureVM(client *vms.Client, obj metav1.Object, log logr.Logger) error {
	local, ok := obj.(*azurev1alpha1.VM)
	if !ok {
		return errors.New("failed type assertion after switching on type. check switch statement and function invocation.")
	}

	log = log.WithValues("type", "vm", "name", local.Spec.Name)

	if err := client.ForSubscription(local.Spec.SubscriptionID); err != nil {
		return errors.Wrap(err, "failed to get client for subscription")
	}

	return wait.ExponentialBackoff(backoff(), func() (done bool, err error) {
		log.Info("reconciling")
		done, err = client.Ensure(context.Background(), local)
		if err != nil {
			log.Error(err, "failed reconcile attempt")
		}
		return done, nil
	})
}

func DeleteVM(client *vms.Client, obj metav1.Object, log logr.Logger) error {
	local, ok := obj.(*azurev1alpha1.VM)
	if !ok {
		return errors.New("failed type assertion after switching on type. check switch statement and function invocation.")
	}

	log = log.WithValues("type", "vm", "name", local.Spec.Name)

	if err := client.ForSubscription(local.Spec.SubscriptionID); err != nil {
		return errors.Wrap(err, "failed to get client for subscription")
	}

	return wait.ExponentialBackoff(backoff(), func() (done bool, err error) {
		log.Info("deleting")
		found, err := client.Delete(context.Background(), local)
		return !found, err
	})
}

func EnsureKeyvault(client *keyvaults.Client, obj metav1.Object, log logr.Logger) error {
	local, ok := obj.(*azurev1alpha1.Keyvault)
	if !ok {
		return errors.New("failed type assertion after switching on type. check switch statement and function invocation.")
	}

	log = log.WithValues("type", "keyvault", "name", local.Spec.Name)

	if err := client.ForSubscription(local.Spec.SubscriptionID); err != nil {
		return errors.Wrap(err, "failed to get client for subscription")
	}

	return wait.ExponentialBackoff(backoff(), func() (done bool, err error) {
		log.Info("reconciling")
		err = client.Ensure(context.Background(), local)
		if err != nil {
			log.Error(err, "failed reconcile attempt")
		}
		return err == nil, nil
	})
}

func DeleteKeyvault(client *keyvaults.Client, obj metav1.Object, log logr.Logger) error {
	local, ok := obj.(*azurev1alpha1.Keyvault)
	if !ok {
		return errors.New("failed type assertion after switching on type. check switch statement and function invocation.")
	}

	log = log.WithValues("type", "keyvault", "name", local.Spec.Name)

	if err := client.ForSubscription(local.Spec.SubscriptionID); err != nil {
		return errors.Wrap(err, "failed to get client for subscription")
	}

	return wait.ExponentialBackoff(backoff(), func() (done bool, err error) {
		log.Info("deleting")
		err = client.Delete(context.Background(), local)
		return err == nil, err
	})
}

func EnsureIdentity(client *identities.Client, obj metav1.Object, log logr.Logger) error {
	local, ok := obj.(*azurev1alpha1.Identity)
	if !ok {
		return errors.New("failed type assertion after switching on type. check switch statement and function invocation.")
	}

	log = log.WithValues("type", "identity", "name", local.Spec.Name)

	if err := client.ForSubscription(local.Spec.SubscriptionID); err != nil {
		return errors.Wrap(err, "failed to get client for subscription")
	}

	return wait.ExponentialBackoff(backoff(), func() (done bool, err error) {
		log.Info("reconciling")
		err = client.Ensure(context.Background(), local)
		if err != nil {
			log.Error(err, "failed reconcile attempt")
		}
		return err == nil, nil
	})
}

func DeleteIdentity(client *identities.Client, obj metav1.Object, log logr.Logger) error {
	local, ok := obj.(*azurev1alpha1.Identity)
	if !ok {
		return errors.New("failed type assertion after switching on type. check switch statement and function invocation.")
	}

	log = log.WithValues("type", "identity", "name", local.Spec.Name)

	if err := client.ForSubscription(local.Spec.SubscriptionID); err != nil {
		return errors.Wrap(err, "failed to get client for subscription")
	}

	return wait.ExponentialBackoff(backoff(), func() (done bool, err error) {
		log.Info("deleting")
		err = client.Delete(context.Background(), local)
		return err == nil, err
	})
}

func EnsureLoadBalancer(client *loadbalancers.Client, obj metav1.Object, log logr.Logger) error {
	local, ok := obj.(*azurev1alpha1.LoadBalancer)
	if !ok {
		return errors.New("failed type assertion after switching on type. check switch statement and function invocation.")
	}

	log = log.WithValues("type", "loadbalancer", "name", local.Spec.Name)

	if err := client.ForSubscription(local.Spec.SubscriptionID); err != nil {
		return errors.Wrap(err, "failed to get client for subscription")
	}

	return wait.ExponentialBackoff(backoff(), func() (done bool, err error) {
		log.Info("reconciling")
		done, err = client.Ensure(context.Background(), local)
		if err != nil {
			log.Error(err, "failed reconcile attempt")
		}
		return done, nil
	})
}

func DeleteLoadBalancer(client *loadbalancers.Client, obj metav1.Object, log logr.Logger) error {
	local, ok := obj.(*azurev1alpha1.LoadBalancer)
	if !ok {
		return errors.New("failed type assertion after switching on type. check switch statement and function invocation.")
	}

	log = log.WithValues("type", "loadbalancer", "name", local.Spec.Name)

	if err := client.ForSubscription(local.Spec.SubscriptionID); err != nil {
		return errors.Wrap(err, "failed to get client for subscription")
	}

	return wait.ExponentialBackoff(backoff(), func() (done bool, err error) {
		log.Info("deleting")
		found, err := client.Delete(context.Background(), local)
		return !found, err
	})
}

func EnsureSecret(client *secrets.Client, obj metav1.Object, log logr.Logger) error {
	local, ok := obj.(*azurev1alpha1.Secret)
	if !ok {
		return errors.New("failed type assertion after switching on type. check switch statement and function invocation.")
	}

	log = log.WithValues("type", "secret", "name", local.Spec.Name)

	return wait.ExponentialBackoff(backoff(), func() (done bool, err error) {
		log.Info("reconciling")
		err = client.Ensure(context.Background(), local)
		if err != nil {
			log.Error(err, "failed reconcile attempt")
		}
		return err == nil, nil
	})
}

func DeleteSecret(client *secrets.Client, obj metav1.Object, log logr.Logger) error {
	local, ok := obj.(*azurev1alpha1.Secret)
	if !ok {
		return errors.New("failed type assertion after switching on type. check switch statement and function invocation.")
	}

	log = log.WithValues("type", "secret", "name", local.Spec.Name)

	return wait.ExponentialBackoff(backoff(), func() (done bool, err error) {
		log.Info("deleting")
		err = client.Delete(context.Background(), local)
		return err == nil, err
	})
}

func EnsureSecretBundle(client *secretbundles.Client, obj metav1.Object, log logr.Logger) error {
	local, ok := obj.(*azurev1alpha1.SecretBundle)
	if !ok {
		return errors.New("failed type assertion after switching on type. check switch statement and function invocation.")
	}

	log = log.WithValues("type", "secretbundle", "name", local.Spec.Name)

	return wait.ExponentialBackoff(backoff(), func() (done bool, err error) {
		log.Info("reconciling")
		err = client.Ensure(context.Background(), local)
		if err != nil {
			log.Error(err, "failed reconcile attempt")
		}
		return err == nil, nil
	})
}

func DeleteSecretBundle(client *secretbundles.Client, obj metav1.Object, log logr.Logger) error {
	local, ok := obj.(*azurev1alpha1.SecretBundle)
	if !ok {
		return errors.New("failed type assertion after switching on type. check switch statement and function invocation.")
	}

	log = log.WithValues("type", "secretbundle", "name", local.Spec.Name)

	return wait.ExponentialBackoff(backoff(), func() (done bool, err error) {
		log.Info("deleting")
		err = client.Delete(context.Background(), local)
		return err == nil, err
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
