package util

import (
	"encoding/json"

	"k8s.io/klog/v2"

	v1 "github.com/mfojtik/shodan/pkg/api/job/v1"
	"github.com/mfojtik/shodan/pkg/config"
)

// UpdateJobs take a storage and list of jobs to be updated.
func UpdateJobs(s config.Storage, jobs []v1.Job) error {
	for i := range jobs {
		jobJSON, err := json.Marshal(jobs[i])
		if err != nil {
			return err
		}
		klog.Infof("Updating job %q: %s", jobs[i].Name, string(jobJSON))
		if err := s.Set(jobs[i].Name, jobJSON); err != nil {
			return err
		}
	}
	return nil
}
