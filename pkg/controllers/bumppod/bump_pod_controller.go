package bumppod

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"

	"github.com/openshift/library-go/pkg/controller/factory"
	"github.com/openshift/library-go/pkg/operator/events"

	jobv1 "github.com/mfojtik/shodan/pkg/api/job/v1"
	"github.com/mfojtik/shodan/pkg/config"
	"github.com/mfojtik/shodan/pkg/controllers/util"
	"github.com/mfojtik/shodan/pkg/storage"
)

type controller struct {
	options config.CommonOptions
}

func NewController(options config.CommonOptions, recorder events.Recorder) factory.Controller {
	c := &controller{
		options: options,
	}
	return factory.New().ResyncEvery(10*time.Second).WithSync(c.sync).WithInformers(c.options.Informers()...).ToController("BumpPodController", recorder)
}

func (c *controller) sync(ctx context.Context, context factory.SyncContext) error {
	jobs, err := storage.GetPendingBumpJobs(c.options.Storage)
	if err != nil {
		return err
	}

	podsToCreate := []*v1.Pod{}
	jobsToUpdate := []jobv1.Job{}

	for _, j := range jobs {
		// skip jobs that link to PR which was not merged
		if len(j.Status.BaseBranch) == 0 {
			continue
		}
		if j.Status.State != jobv1.PendingJobState {
			continue
		}

		target := strings.Split(j.Spec.Params[0], "/")
		if len(target) != 2 {
			klog.Warningf("Invalid job parameters for bump: %s", spew.Sdump(j))
			continue
		}

		podParams := podParameters{
			forkName:         "shodan-bot",
			repositoryOwner:  target[0],
			repositoryName:   target[1],
			repositoryBranch: j.Status.BaseBranch,
			goModuleName:     "github.com/" + j.Spec.Owner + "/" + j.Spec.Repository,
			goModuleBranch:   j.Status.BaseBranch,
			targetBranchName: j.Name,
		}

		podsToCreate = append(podsToCreate, getJobPod("job-"+j.Name, podParams))
	}

	if len(podsToCreate) == 0 {
		klog.Infof("There are no pods to be created :-(")
	}

	errors := []error{}

	for _, p := range podsToCreate {
		klog.Infof("Creating new pod:\n%s\n", spew.Sdump(p))
		_, err := c.options.Client.CoreV1().Pods("shodan").Create(ctx, p, metav1.CreateOptions{})
		if err != nil {
			klog.Warningf("Unable to create pod %q: %v", p.Name, err)
			errors = append(errors, err)
			continue
		}
		updatedJob, err := updateJob(strings.TrimPrefix(p.Name, "job-"), jobs, func(job *jobv1.Job) {
			job.Status.State = jobv1.RunningJobState
		})
		if err != nil {
			klog.Errorf("Unable to get job %q to update: %v", p.Name, err)
			errors = append(errors, err)
			continue
		}
		klog.Infof("Updating job %s", spew.Sdump(updatedJob))
		jobsToUpdate = append(jobsToUpdate, *updatedJob)
	}

	return util.UpdateJobs(c.options.Storage, jobsToUpdate)
}

func updateJob(name string, jobs []jobv1.Job, fn func(job *jobv1.Job)) (*jobv1.Job, error) {
	var job *jobv1.Job
	for i := range jobs {
		if jobs[i].Name == name {
			job = &jobs[i]
			break
		}
	}
	if job == nil {
		return nil, fmt.Errorf("unable to find job %q", name)
	}

	// mutate the job
	fn(job)

	return job, nil
}

func moduleNameFromParams(s []string) string {
	if len(s) != 1 {
		return ""
	}

	// @shodan-bot bump github.com/openshift/library-go
	if strings.HasPrefix(s[0], "github.com/") {
		return s[0]
	}

	// @shodan-bot bump openshift/library-go
	return "github.com/" + s[0]
}

type podParameters struct {
	forkName         string
	repositoryOwner  string
	repositoryName   string
	repositoryBranch string
	goModuleName     string
	goModuleBranch   string
	targetBranchName string
}

func (p podParameters) StringSlice() []string {
	return []string{
		p.forkName,
		p.repositoryOwner,
		p.repositoryName,
		p.repositoryBranch,
		p.goModuleName,
		p.goModuleBranch,
		p.targetBranchName,
	}
}

func int64p(i int) *int64 {
	v := int64(i)
	return &v
}

func int32p(i int) *int32 {
	v := int32(i)
	return &v
}

func boolp(b bool) *bool {
	return &b
}

func getJobPod(name string, params podParameters) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "shodan",
			Labels: map[string]string{
				"shodan.io/type": "bump",
			},
		},
		Spec: v1.PodSpec{
			Volumes: []v1.Volume{
				{
					Name: "keys",
					VolumeSource: v1.VolumeSource{
						Secret: &v1.SecretVolumeSource{
							SecretName:  "keys",
							Optional:    boolp(false),
							DefaultMode: int32p(384), // 0600 for SSH keys
						},
					},
				},
			},
			Containers: []v1.Container{
				{
					Name:    "bump",
					Image:   "quay.io/mfojtik/shodan:bumpdeps",
					Command: []string{"/usr/bin/bump-repo.sh"},
					Args:    params.StringSlice(),
					VolumeMounts: []v1.VolumeMount{
						{
							Name:      "keys",
							ReadOnly:  true,
							MountPath: "/root/.ssh",
						},
					},
					VolumeDevices:            nil,
					Lifecycle:                nil,
					TerminationMessagePolicy: v1.TerminationMessageFallbackToLogsOnError,
					ImagePullPolicy:          v1.PullIfNotPresent,
				},
			},
			RestartPolicy:                 v1.RestartPolicyNever,
			TerminationGracePeriodSeconds: int64p(10),
			ActiveDeadlineSeconds:         int64p(60 * 20), // cap the job runtime to 20 minutes
			AutomountServiceAccountToken:  boolp(true),
		},
	}
}
