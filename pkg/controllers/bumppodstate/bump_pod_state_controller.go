package bumppodstate

import (
	"context"
	"fmt"
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
	podsToDelete := []string{}

	for _, j := range jobPods.Items {
		if j.Status.Phase != v1.PodFailed && j.Status.Phase != v1.PodSucceeded {
			continue
		}

		klog.Infof("Processing finished pod %q", j.Name)

		job, err := storage.GetJobByName(c.options.Storage, strings.TrimPrefix(j.Name, "job-"))
		if err != nil {
			klog.Warningf("Unable to get job %q: %v", j.Name, err)
			continue
		}
		podsToDelete = append(podsToDelete, j.Name)

		if j.Status.Phase == v1.PodFailed {
			if terminationMessage := j.Status.ContainerStatuses[0].State.Terminated.Message; len(terminationMessage) > 0 {
				if strings.Contains(terminationMessage, "nothing to commit, working tree clean") {
					job.Status.Message = "Human, it looks like the target repository is already up-to-date, after bumping there was no diff."
					job.Status.State = jobv1.FinishedJobState
					jobsToUpdate = append(jobsToUpdate, *job)
					continue
				}
				job.Status.Message = fmt.Sprintf("Human, something terrible happened during bump:\n```\n%s\n```\n", terminationMessage)
				continue
			}

			job.Status.Message = "Human, something really bad happened and the pod with bump failed."
			job.Status.State = jobv1.FinishedJobState

		}

		if j.Status.Phase == v1.PodSucceeded {
			job.Status.Message = fmt.Sprintf("Success! A bump pull request from me was open inside %s/%s repository.", job.Spec.Params[0])
			job.Status.State = jobv1.FinishedJobState
			jobsToUpdate = append(jobsToUpdate, *job)
			continue
		}

	}

	for _, podName := range podsToDelete {
		klog.Infof("Deleting pod %q ...", podName)
		if err := c.options.Client.CoreV1().Pods("shodan").Delete(ctx, podName, metav1.DeleteOptions{}); err != nil {
			klog.Warningf("Unable to delete pod %s: %v", podName, err)
		}
	}

	return util.UpdateJobs(c.options.Storage, jobsToUpdate)
}
