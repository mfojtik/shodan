package bumppodstate

import (
	"context"
	"strings"
	"time"

	"github.com/mfojtik/shodan/pkg/storage"
	"k8s.io/klog/v2"

	v1 "k8s.io/api/core/v1"

	"github.com/mfojtik/shodan/pkg/controllers/util"

	jobv1 "github.com/mfojtik/shodan/pkg/api/job/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/mfojtik/shodan/pkg/config"
	"github.com/openshift/library-go/pkg/controller/factory"
	"github.com/openshift/library-go/pkg/operator/events"
)

type controller struct {
	options config.CommonOptions
}

func NewController(options config.CommonOptions, recorder events.Recorder) factory.Controller {
	c := &controller{
		options: options,
	}
	return factory.New().ResyncEvery(10*time.Second).WithSync(c.sync).WithInformers(c.options.Informers()...).ToController("BumpPodStateController", recorder)
}

func (c *controller) sync(ctx context.Context, context factory.SyncContext) error {
	jobPods, err := c.options.Client.CoreV1().Pods("shodan").List(ctx, metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{"shodan.io/type": "bump"}).String(),
	})
	if err != nil {
		return err
	}

	jobsToUpdate := []jobv1.Job{}
	for _, j := range jobPods.Items {
		if j.Status.Phase != v1.PodFailed && j.Status.Phase != v1.PodSucceeded {
			continue
		}

		job, err := storage.GetJobByName(c.options.Storage, strings.TrimPrefix(j.Name, "job-"))
		if err != nil {
			klog.Warningf("Unable to get job %q: %v", j.Name, err)
			continue
		}

		terminationMessage := string(j.Status.ContainerStatuses[0].LastTerminationState.Terminated.Message)
		if strings.Contains(terminationMessage, "")

	}

	return util.UpdateJobs(c.options.Storage, jobsToUpdate)
}
