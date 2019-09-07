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
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/resourcegroups"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/trafficmanagers"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/virtualnetworks"
	"github.com/alexeldeib/incendiary-iguana/pkg/config"
	"github.com/alexeldeib/incendiary-iguana/pkg/yamlutil"
)

type token struct{}

const (
	limit           = 1
	backoffSteps    = 10
	backoffFactor   = 1.5
	backoffInterval = 4 * time.Second
	backoffJitter   = 1.0
	backoffLimit    = 600 * time.Second
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

	decoder := yamlutil.NewYAMLDecoder(reader, scheme)
	defer decoder.Close()

	// accumulators
	// gvks := []schema.GroupVersionKind{}
	objects := []metav1.Object{}

	// parsing
	for {
		obj, _, err := decoder.Decode(nil, nil)
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
	var err error
	log := log.WithValues("action", "ensure")
	log.Info("starting reconciliation")
	switch obj.(type) {
	case *appsv1.Deployment:
		log.Info("Deployment!")
	case *azurev1alpha1.ResourceGroup:
		err = EnsureResourceGroup(resourcegroups.New(configuration), obj, log)
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
		fmt.Printf("%+v", err)
		return
	}
	log.Info("sucessfully reconciled")
}

func Delete(obj metav1.Object, configuration *config.Config, errs chan error) {
	var err error
	log := log.WithValues("action", "delete")
	log.Info("starting deletion")
	switch obj.(type) {
	case *appsv1.Deployment:
		log.Info("Deployment!")
	case *azurev1alpha1.ResourceGroup:
		err = DeleteResourceGroup(resourcegroups.New(configuration), obj, log)
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
		fmt.Printf("%+v", err)
		return
	}
	log.Info("sucessfully deleted")
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

	log = log.WithValues("name", local.Spec.Name)

	if err := client.ForSubscription(local.Spec.SubscriptionID); err != nil {
		return errors.Wrap(err, "failed to get client for subscription")
	}

	if err := client.Ensure(context.Background(), local); err != nil {
		return errors.Wrap(err, "failed to reconcile")
	}

	return wait.ExponentialBackoff(backoff(), func() (done bool, err error) {
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

func backoff() wait.Backoff {
	return wait.Backoff{
		Cap:      backoffLimit,
		Steps:    backoffSteps,
		Factor:   backoffFactor,
		Duration: backoffInterval,
		Jitter:   backoffJitter,
	}
}
