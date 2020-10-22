package informers

import (
	"github.com/openshift/library-go/pkg/controller/factory"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type StorageInformer interface {
	// Trigger is used in storage to artificially trigger the informer handlers.
	// This is used by boltdb for example when the object is stored, updated or deleted.
	// This allows faster reaction in controllers and simulates Kubernetes etcd informers.
	Trigger()

	factory.Informer
}

// FakeRuntimeObject is needed for resource handler to succeed.
type FakeRuntimeObject struct{}

var _ runtime.Object = &FakeRuntimeObject{}

func (f *FakeRuntimeObject) GetObjectKind() schema.ObjectKind {
	return nil
}

func (f *FakeRuntimeObject) DeepCopyObject() runtime.Object {
	n := *f
	return &n
}
