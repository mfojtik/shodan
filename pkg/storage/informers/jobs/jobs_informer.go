package jobs

import (
	"github.com/mfojtik/shodan/pkg/storage/informers"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

type Informer struct {
	handlers []cache.ResourceEventHandler
}

func New() informers.StorageInformer {
	return &Informer{}
}

func (r *Informer) AddEventHandler(handler cache.ResourceEventHandler) {
	r.handlers = append(r.handlers, handler)
}

func (r *Informer) HasSynced() bool {
	return true
}

func (r *Informer) Trigger() {
	for i := range r.handlers {
		klog.Infof("Triggering jobs informer ...")
		r.handlers[i].OnAdd(&informers.FakeRuntimeObject{}) // this will queue reconcile for controllers
	}
}
