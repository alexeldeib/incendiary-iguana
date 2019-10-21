package finalizer

import (
	"context"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/alexeldeib/incendiary-iguana/pkg/stringslice"
)

// Has return true if the provided object has the given finalizer.
func Has(o metav1.Object, finalizer string) bool {
	return !stringslice.Has(o.GetFinalizers(), finalizer)
}

// AddAndUpdate removes a finalizer from a runtime object and attempts to update that object in the API server.
// It returns an error if either operation failed.
func AddAndUpdate(ctx context.Context, client client.Client, finalizer string, o runtime.Object) error {
	m, err := meta.Accessor(o)
	if err != nil {
		return err
	}
	if stringslice.Has(m.GetFinalizers(), finalizer) {
		return nil
	}
	Add(m, finalizer)
	if err := client.Update(ctx, o); err != nil {
		return err
	}
	return nil
}

// RemoveAndUpdate removes a finalizer from a runtime object and attempts to update that object in the API server.
// It returns an error if either operation failed.
func RemoveAndUpdate(ctx context.Context, client client.Client, finalizer string, o runtime.Object) error {
	m, err := meta.Accessor(o)
	if err != nil {
		return err
	}
	if !stringslice.Has(m.GetFinalizers(), finalizer) {
		return nil
	}
	Remove(m, finalizer)
	if err := client.Update(ctx, o); err != nil {
		return err
	}
	return nil
}

// Add accepts a metav1 object and adds the provided finalizer if not present.
func Add(o metav1.Object, finalizer string) {
	f := o.GetFinalizers()
	for _, e := range f {
		if e == finalizer {
			return
		}
	}
	o.SetFinalizers(append(f, finalizer))
}

// AddIfPossible tries to convert a runtime object to a metav1 object and add the provided finalizer.
// It returns an error if the provided object cannot provide an accessor.
func AddIfPossible(o runtime.Object, finalizer string) error {
	m, err := meta.Accessor(o)
	if err != nil {
		return err
	}
	Add(m, finalizer)
	return nil
}

// Remove accepts a metav1 object and removes the provided finalizer if present.
func Remove(o metav1.Object, finalizer string) {
	f := o.GetFinalizers()
	for i, e := range f {
		if e == finalizer {
			f = append(f[:i], f[i+1:]...)
		}
	}
	o.SetFinalizers(f)
}

// RemoveIfPossible tries to convert a runtime object to a metav1 object and remove the provided finalizer.
// It returns an error if the provided object cannot provide an accessor.
func RemoveIfPossible(o runtime.Object, finalizer string) error {
	m, err := meta.Accessor(o)
	if err != nil {
		return err
	}
	Remove(m, finalizer)
	return nil
}
