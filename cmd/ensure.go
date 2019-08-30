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

	"github.com/pkg/errors"

	// "text/template"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	azurev1alpha1 "github.com/alexeldeib/incendiary-iguana/api/v1alpha1"
	"github.com/alexeldeib/incendiary-iguana/pkg/clients/trafficmanagers"
	"github.com/alexeldeib/incendiary-iguana/pkg/config"
	"github.com/alexeldeib/incendiary-iguana/pkg/yamlutil"
	"github.com/sanity-io/litter"
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

type EnsureOptions struct {
	File string
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

	for {
		obj, _, err := decoder.Decode(nil, nil)

		if err == io.EOF {
			break
		} else if err != nil {
			fmt.Printf("%+#v\n", err)
		}

		if runtime.IsNotRegisteredError(err) {
			continue
		}

		if err != nil {
			return err
		}
		switch obj.(type) {
		case *appsv1.Deployment:
			litter.Dump("Deployment!")
		case *azurev1alpha1.TrafficManager:
			if err := ensureTrafficManager(obj); err != nil {
				fmt.Printf("%+v", err)
				os.Exit(1)
			}
		default:
			litter.Dump("nothing to do.")
		}
		// litter.Dump(gvk)
		// litter.Dump(obj)
		// want, err := template.New("test").Parse("{{.Status.ReadyReplicas}}")
		// if err != nil {
		// 	panic(err)
		// }
		// have, err := template.New("test").Parse("{{.Status.AvailableReplicas}}")
		// if err != nil {
		// 	panic(err)
		// }
		// wantBuf := bytes.NewBuffer([]byte{})
		// err = want.Execute(wantBuf, obj)
		// if err != nil {
		// 	panic(err)
		// }
		// haveBuf := bytes.NewBuffer([]byte{})
		// err = have.Execute(haveBuf, obj)
		// if err != nil {
		// 	panic(err)
		// }
		// fmt.Printf("%s\n", haveBuf.String())
		// fmt.Printf("%s\n", wantBuf.String())
		// fmt.Printf("%t\n", haveBuf.String() == wantBuf.String())
	}

	return nil
}

func ensureTrafficManager(obj runtime.Object) error {
	tm, ok := obj.(*azurev1alpha1.TrafficManager)
	if !ok {
		return errors.New("failed type assertion after switching on type. check switch statement and function invocation.")
	}

	client := trafficmanagers.New(configuration)

	if err := client.ForSubscription(tm.Spec.SubscriptionID); err != nil {
		return errors.Wrap(err, "failed to get client for subscription")
	}

	log.WithValues("name", tm.Spec.Name).Info("reconciling tm")
	if err := client.Ensure(context.Background(), tm); err != nil {
		return errors.Wrap(err, "failed to ensure tm")
	}

	interval := 5 * time.Second
	timeout := 60 * time.Second

	log.Info("waiting for tm monitor to come online")

	err := wait.PollImmediate(interval, timeout, func() (done bool, err error) {
		status, err := client.GetProfileStatus(context.Background(), tm)
		litter.Dump(status)
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
