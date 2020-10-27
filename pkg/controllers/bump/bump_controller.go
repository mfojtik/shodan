package bump

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"k8s.io/klog/v2"

	"github.com/openshift/library-go/pkg/controller/factory"
	"github.com/openshift/library-go/pkg/operator/events"

	v1 "github.com/mfojtik/shodan/pkg/api/job/v1"
	"github.com/mfojtik/shodan/pkg/config"
	"github.com/mfojtik/shodan/pkg/controllers/util"
	"github.com/mfojtik/shodan/pkg/storage"
)

type controller struct {
	options config.CommonOptions
}

// NewController returns a controller that handle jobs with "bump" type.
// For every job with this type, this controller will periodically watch for the pull request associated with this job.
// When the pull request is merged, the base branch of that pull request is added to the job status.
// After base branch is provided, the job should be executed by bump pod controller that create pod and run the actual bump and open a PR to target repository.
func NewController(options config.CommonOptions, recorder events.Recorder) factory.Controller {
	c := &controller{
		options: options,
	}
	return factory.New().ResyncEvery(60*time.Second).WithSync(c.sync).WithInformers(c.options.Informers()...).ToController("BumpController", recorder)
}

func (c *controller) sync(ctx context.Context, context factory.SyncContext) error {
	jobs, err := storage.GetPendingBumpJobs(c.options.Storage)
	if err != nil {
		return err
	}

	klog.Infof("Found %d pending bump jobs ...", len(jobs))
	jobsToUpdate := []v1.Job{}

	for _, j := range jobs {
		if j.Status.State == v1.FinishedJobState {
			continue
		}

		if len(j.Spec.Params) == 0 {
			jobsToUpdate = append(jobsToUpdate, setJobInvalid(j, fmt.Sprintf("Invalid job parameters %#v, expected exactly one repository name", j.Spec.Params)))
			continue
		}

		merged, baseBranch, err := c.getPullRequestStatus(ctx, j.Spec.Owner, j.Spec.Repository, j.Spec.IssueID)
		if err != nil {
			return err
		}

		// if the job is merged, update the job with base branch, this will trigger the bump pod controller that will perform the go mod bump in target repository.
		if needBaseBranch, job := setJobBaseBranch(j, baseBranch); merged && needBaseBranch {
			jobsToUpdate = append(jobsToUpdate, job)
		} else {
			klog.Infof("Waiting for %s/%s#%s to merge before bumping %s#%s ...", j.Spec.Owner, j.Spec.Repository, j.Spec.IssueID, j.Spec.Params[0], job.Status.BaseBranch)
		}
	}

	return util.UpdateJobs(c.options.Storage, jobsToUpdate)
}

// getPullRequestStatus gets info whether pull is merged or not and the base branch, it returns error if github client fails.
func (c *controller) getPullRequestStatus(ctx context.Context, owner, repo, pullID string) (bool, string, error) {
	client := c.options.NewGithubClient(ctx)
	id, err := strconv.Atoi(pullID)
	if err != nil {
		return false, "", err
	}
	pull, _, err := client.PullRequests.Get(ctx, owner, repo, id)
	if err != nil {
		return false, "", err
	}
	return pull.GetMerged(), pull.GetBase().GetRef(), nil
}

func setJobBaseBranch(j v1.Job, branch string) (bool, v1.Job) {
	if j.Status.BaseBranch == branch {
		return false, j
	}
	j.Status.BaseBranch = branch
	return true, j
}

func setJobInvalid(j v1.Job, reason string) v1.Job {
	j.Status = v1.JobStatus{
		State:   v1.FinishedJobState,
		Message: reason,
	}
	return j
}
