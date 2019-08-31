package main

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
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/trafficmanagers"
	"github.com/alexeldeib/incendiary-iguana/pkg/config"
	"github.com/alexeldeib/incendiary-iguana/pkg/yamlutil"
)

var (
	scheme        = runtime.NewScheme()
	log           = ctrl.Log.WithName("tinker")
	configuration = config.New(log)
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)
	_ = azurev1alpha1.AddToScheme(scheme)
	ctrl.SetLogger(zap.Logger(false))

	err := configuration.DetectAuthorizer()
	if err != nil {
		log.Error(err, "failed to detect any authorizer")
		os.Exit(1)
	}
}

func NewEnsureCommand() *cobra.Command {
	opts := NewEnsureOptions()
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

type EnsureOptions struct {
	File  string
	Debug bool
}

func NewEnsureOptions() EnsureOptions {
	return EnsureOptions{}
}

func (opts *EnsureOptions) Ensure() error {
	var reader io.ReadCloser
	if opts.File == "-" {
		reader = ioutil.NopCloser(os.Stdin)
	} else {
		path, err := filepath.Abs(opts.File)
		if err != nil {
			return err
		}

		reader, err = os.Open(path)
		if err != nil {
			return err
		}
	}

	decoder := yamlutil.NewYAMLDecoder(reader, scheme)
	defer decoder.Close()

	// slightly parallel
	limit := 5
	pool := make(chan token, limit)

	// accumulators
	gvks := []schema.GroupVersionKind{}
	objects := []metav1.Object{}

	// parsing
	for {
		obj, gvk, err := decoder.Decode(nil, nil)
		if err == io.EOF {
			break
		} else if err != nil {
			fmt.Printf("%+#v\n", err)
			return err
		}

		if runtime.IsNotRegisteredError(err) {
			continue
		}

		if err != nil {
			return err
		}

		actual, err := meta.Accessor(obj)
		if err != nil {
			return err
		}

		gvks = append(gvks, *gvk)
		objects = append(objects, actual)
	}

	if opts.Debug {
		log.V(1).Info("dumping manifests before applying")
		for _, obj := range objects {
			litter.Dump(obj)
		}
	}

	// apply objects
	for _, obj := range objects {
		// n.b. acquire
		pool <- token{}
		go func(o metav1.Object) {
			ensure(o, log)
			// n.b. release
			<-pool
		}(obj)
	}

	// n.b. wait for task completion
	for n := limit; n > 0; n-- {
		pool <- token{}
	}

	return nil
}

func ensure(obj metav1.Object, log logr.Logger) {
	switch obj.(type) {
	case *appsv1.Deployment:
		log.Info("Deployment!")
	case *azurev1alpha1.TrafficManager:
		if err := ensureTrafficManager(obj, log); err != nil {
			fmt.Printf("%+v", err)
		}
	default:
		log.Info("nothing to do.")
	}
}

func ensureTrafficManager(obj metav1.Object, log logr.Logger) error {
	tm, ok := obj.(*azurev1alpha1.TrafficManager)
	if !ok {
		return errors.New("failed type assertion after switching on type. check switch statement and function invocation.")
	}

	log = log.WithValues("name", tm.Spec.Name)
	client := trafficmanagers.New(configuration)

	if err := client.ForSubscription(tm.Spec.SubscriptionID); err != nil {
		return errors.Wrap(err, "failed to get client for subscription")
	}
	log.Info("reconciling tm")
	if err := client.Ensure(context.Background(), tm); err != nil {
		return errors.Wrap(err, "failed to ensure tm")
	}

	interval := 5 * time.Second
	timeout := 180 * time.Second

	log.Info("waiting for monitor status")

	err := wait.PollImmediate(interval, timeout, func() (done bool, err error) {
		status, err := client.GetProfileStatus(context.Background(), tm)
		log.Info("waiting for appropriate status", "status", status)
		if err != nil {
			return false, errors.Wrap(err, "failed to get tm monitor status")
		}
		if status == "Online" || status == "Disabled" {
			return true, nil
		}
		return false, nil
	})

	if err != nil {
		return errors.Wrap(err, "failed to wait for tm to come online")
	}

	log.Info("sucessfully ensured tm")
	return nil
}

type token struct{}
