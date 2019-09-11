package ensure

import (
	// "bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	// "text/template"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"github.com/sanity-io/litter"
	"github.com/spf13/cobra"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/publicips"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/redis"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/resourcegroups"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/securitygroups"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/servicebus"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/subnets"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/trafficmanagers"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/virtualnetworks"
	"github.com/alexeldeib/incendiary-iguana/pkg/config"
	"github.com/alexeldeib/incendiary-iguana/pkg/decoder"
)

type token struct{}

const (
	limit           = 5
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
		Use:   "delete",
		Short: "Deletes specified resources.",
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

func do(objects []metav1.Object, configuration *config.Config, applyFunc func(obj metav1.Object, configuration *config.Config, errs chan error)) error {
	// apply objects
	pool := make(chan token, limit)
	errs := make(chan error, limit)
	for _, obj := range objects {
		// n.b. acquire
		pool <- token{}
		go func(o metav1.Object) {
			applyFunc(o, configuration, errs)
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

func Ensure(obj metav1.Object, configuration *config.Config, errs chan error) {
	// var err error
	// var kubeconfig *rest.Config
	// var kubeclient *client.Client

	// if kubeconfig, err = ctrl.GetConfig(); err == nil {
	// 	c, err := client.New(kubeconfig, client.Options{})
	// 	if err != nil {
	// 		log.Error(err, "error getting kubeclient after successfully finding kubeconfig")
	// 	} else {
	// 		kubeclient = new(client.Client)
	// 		*kubeclient = c
	// 	}
	// }

	log := log.WithValues("action", "ensure")
	log.Info("starting reconciliation")
	switch obj.(type) {
	case *appsv1.Deployment:
		log.Info("deploy")
		// To avoid the painful copy paste you are about to see, it might be useful to create separate decoders for
		// types which require a kubernetes client to apply
		// if kubeclient == nil {
		// 	err = errors.New("kubeclient is nil while trying to apply kubernetes object")
		// 	errs <- err
		// 	log.Info("failed to delete")
		// 	fmt.Printf("%s\n", err.Error())
		// }
		// err = EnsureDeployment(*kubeclient, obj, log)
	case *corev1.Service:
		log.Info("service")
		// To avoid the painful copy paste you are about to see, it might be useful to create separate decoders for
		// types which require a kubernetes client to apply
		// if kubeclient == nil {
		// 	err = errors.New("kubeclient is nil while trying to apply kubernetes object")
		// 	errs <- err
		// 	log.Info("failed to delete")
		// 	fmt.Printf("%s\n", err.Error())
		// }
		// err = EnsureService(*kubeclient, obj, log)
	case *azurev1alpha1.Redis:
		err = EnsureRedis(redis.New(configuration, nil, nil), obj, log)
	case *azurev1alpha1.ResourceGroup:
		err = EnsureResourceGroup(resourcegroups.New(configuration), obj, log)
	case *azurev1alpha1.ServiceBusNamespace:
		err = EnsureServiceBusNamespace(servicebus.New(configuration, nil, nil), obj, log)
	case *azurev1alpha1.TrafficManager:
		err = EnsureTrafficManager(trafficmanagers.New(configuration), obj, log)
	case *azurev1alpha1.VirtualNetwork:
		err = EnsureVirtualNetwork(virtualnetworks.New(configuration), obj, log)
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

func Delete(obj metav1.Object, configuration *config.Config, errs chan error) {
	// var err error
	// var kubeconfig *rest.Config
	// var kubeclient *client.Client

	// if kubeconfig, err = ctrl.GetConfig(); err == nil {
	// 	c, err := client.New(kubeconfig, client.Options{})
	// 	if err != nil {
	// 		log.Error(err, "error getting kubeclient after successfully finding kubeconfig")
	// 	} else {
	// 		kubeclient = new(client.Client)
	// 		*kubeclient = c
	// 	}
	// }

	log := log.WithValues("action", "delete")
	log.Info("starting deletion")
	switch obj.(type) {
	case *appsv1.Deployment:
		log.Info("deploy")
		// if kubeclient == nil {
		// 	err = errors.New("kubeclient is nil while trying to apply kubernetes object")
		// 	errs <- err
		// 	log.Info("failed to delete")
		// 	fmt.Printf("%s\n", err.Error())
		// }
		// err = DeleteDeployment(*kubeclient, obj, log)
	case *corev1.Service:
		log.Info("service")
		// To avoid the painful copy paste you are about to see, it might be useful to create separate decoders for
		// types which require a kubernetes client to apply
		// if kubeclient == nil {
		// 	err = errors.New("kubeclient is nil while trying to apply kubernetes object")
		// 	errs <- err
		// 	log.Info("failed to delete")
		// 	fmt.Printf("%s\n", err.Error())
		// }
		// err = DeleteService(*kubeclient, obj, log)
	case *azurev1alpha1.Redis:
		err = DeleteRedis(redis.New(configuration, nil, nil), obj, log)
	case *azurev1alpha1.ResourceGroup:
		err = DeleteResourceGroup(resourcegroups.New(configuration), obj, log)
	case *azurev1alpha1.ServiceBusNamespace:
		err = DeleteServiceBusNamespace(servicebus.New(configuration, nil, nil), obj, log)
	case *azurev1alpha1.TrafficManager:
		log.Info("Traffic Manager!")
	case *azurev1alpha1.VirtualNetwork:
		err = DeleteVirtualNetwork(virtualnetworks.New(configuration), obj, log)
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

func EnsureDeployment(c client.Client, obj metav1.Object, log logr.Logger) error {
	local, _ := obj.(*appsv1.Deployment)
	kind := strings.ToLower(local.TypeMeta.GroupVersionKind().Kind)
	log = log.WithValues("type", kind, "namespacedName", fmt.Sprintf("%s/%s", local.ObjectMeta.Namespace, local.ObjectMeta.Name))
	return wait.ExponentialBackoff(backoff(), func() (done bool, err error) {
		log.Info("reconciling")
		t := time.Now()
		dateLabel := t.Format(time.RFC3339)
		_, err = controllerutil.CreateOrUpdate(context.Background(), c, local, func() error {
			if local.Spec.Template.ObjectMeta.Annotations == nil {
				local.Spec.Template.ObjectMeta.Annotations = map[string]string{}
			}
			local.Spec.Template.ObjectMeta.Annotations["deployment-marker"] = dateLabel
			return nil
		})
		if err != nil {
			return false, err
		}
		req := client.ObjectKey{
			Namespace: local.ObjectMeta.Namespace,
			Name:      local.ObjectMeta.Name,
		}
		if err := c.Get(context.Background(), req, local); err != nil {
			return false, err
		}
		if local.Spec.Replicas == nil {
			return false, errors.New("deployment replicas was nil. should not occur (apiserver error)")
		}
		expected := *local.Spec.Replicas
		done = local.ObjectMeta.Generation == local.Status.ObservedGeneration && local.Status.UpdatedReplicas == expected && local.Status.ReadyReplicas == expected && local.Status.AvailableReplicas == expected
		return done, nil
	})
}

func DeleteDeployment(c client.Client, obj metav1.Object, log logr.Logger) error {
	local, _ := obj.(*appsv1.Deployment)
	kind := strings.ToLower(local.TypeMeta.GroupVersionKind().Kind)
	log = log.WithValues("type", kind, "namespacedName", fmt.Sprintf("%s/%s", local.ObjectMeta.Namespace, local.ObjectMeta.Name))
	return wait.ExponentialBackoff(backoff(), func() (done bool, err error) {
		log.Info("deleting")
		if err := c.Delete(context.Background(), local); client.IgnoreNotFound(err) != nil {
			return false, err
		}
		req := client.ObjectKey{
			Namespace: local.ObjectMeta.Namespace,
			Name:      local.ObjectMeta.Name,
		}
		if err := c.Get(context.Background(), req, local); err != nil {
			return client.IgnoreNotFound(err) == nil, client.IgnoreNotFound(err)
		}
		return false, nil
	})
}

func EnsureService(c client.Client, obj metav1.Object, log logr.Logger) error {
	local, _ := obj.(*corev1.Service)
	kind := strings.ToLower(local.TypeMeta.GroupVersionKind().Kind)
	log = log.WithValues("type", kind, "namespacedName", fmt.Sprintf("%s/%s", local.ObjectMeta.Namespace, local.ObjectMeta.Name))
	return wait.ExponentialBackoff(backoff(), func() (done bool, err error) {
		log.Info("reconciling")
		_, err = controllerutil.CreateOrUpdate(context.Background(), c, local, func() error {
			return nil
		})
		if err != nil {
			return false, err
		}
		req := client.ObjectKey{
			Namespace: local.ObjectMeta.Namespace,
			Name:      local.ObjectMeta.Name,
		}
		if err := c.Get(context.Background(), req, local); err != nil {
			return false, err
		}
		if local.Spec.Type == corev1.ServiceType("LoadBalancer") {
			ingress := local.Status.LoadBalancer.Ingress
			if len(ingress) >= 1 && ingress[0].IP != "" {
				return true, nil
			}
			return false, nil
		}
		return done, nil
	})
}

func DeleteService(c client.Client, obj metav1.Object, log logr.Logger) error {
	local, _ := obj.(*corev1.Service)
	kind := strings.ToLower(local.TypeMeta.GroupVersionKind().Kind)
	log = log.WithValues("type", kind, "namespacedName", fmt.Sprintf("%s/%s", local.ObjectMeta.Namespace, local.ObjectMeta.Name))
	return wait.ExponentialBackoff(backoff(), func() (done bool, err error) {
		log.Info("deleting")
		if err := c.Delete(context.Background(), local); client.IgnoreNotFound(err) != nil {
			return false, err
		}
		req := client.ObjectKey{
			Namespace: local.ObjectMeta.Namespace,
			Name:      local.ObjectMeta.Name,
		}
		if err := c.Get(context.Background(), req, local); err != nil {
			return client.IgnoreNotFound(err) == nil, client.IgnoreNotFound(err)
		}
		return false, nil
	})
}

func EnsureResourceGroup(client *resourcegroups.Client, obj metav1.Object, log logr.Logger) error {
	local, ok := obj.(*azurev1alpha1.ResourceGroup)
	if !ok {
		return errors.New("failed type assertion after switching on type. check switch statement and function invocation.")
	}

	log = log.WithValues("type", "resourcegroup", "name", local.Spec.Name)

	if err := client.ForSubscription(local.Spec.SubscriptionID); err != nil {
		return errors.Wrap(err, "failed to get client for subscription")
	}

	return wait.ExponentialBackoff(backoff(), func() (done bool, err error) {
		log.Info("reconciling")
		return client.Ensure(context.Background(), local)
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
		return client.Ensure(context.Background(), local)
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
		return client.Ensure(context.Background(), local)
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
		return client.Ensure(context.Background(), local)
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
		return client.Ensure(context.Background(), local)
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
		return client.Ensure(context.Background(), local)
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
		return client.Ensure(context.Background(), local)
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

func backoff() wait.Backoff {
	return wait.Backoff{
		Cap:      backoffLimit,
		Steps:    backoffSteps,
		Factor:   backoffFactor,
		Duration: backoffInterval,
		Jitter:   backoffJitter,
	}
}
