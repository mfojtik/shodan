package notification

import (
	"context"
	"time"

	"k8s.io/klog/v2"

	"github.com/openshift/library-go/pkg/controller/factory"
	"github.com/openshift/library-go/pkg/operator/events"
)

type controller struct {
}

func NewController(recorder events.Recorder) factory.Controller {
	c := &controller{}
	return factory.New().ResyncEvery(30*time.Second).WithSync(c.sync).ToController("NotificationController", recorder)
}

func (c *controller) sync(ctx context.Context, factoryCtx factory.SyncContext) error {
	klog.Info("Syncing notifications ...")
	return nil
}
