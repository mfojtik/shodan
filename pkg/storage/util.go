package storage

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"k8s.io/klog/v2"

	v1 "github.com/mfojtik/shodan/pkg/api/job/v1"
	"github.com/mfojtik/shodan/pkg/config"
)

// GetJobsStats return the time of the last seen job and the number of all jobs in the storage.
// The last seen job can be used to limit the amount of notifications we receive from GH API.
func GetJobsStats(s config.Storage) (time.Time, int, error) {
	jobs, err := s.List("")
	if err != nil {
		return time.Time{}, 0, err
	}
	lastTimestamp := int64(0)
	for _, name := range jobs {
		parts := strings.Split(name, "-")
		if len(parts) != 5 {
			klog.Warningf("invalid storage key found: %q", name)
			continue
		}
		t, _ := strconv.Atoi(parts[4])
		timestamp := int64(t)
		if lastTimestamp == 0 {
			lastTimestamp = timestamp
			continue
		}
		if lastTimestamp < timestamp {
			lastTimestamp = timestamp
		}
	}
	return time.Unix(lastTimestamp, 0), len(jobs), nil
}

// GetPendingBumpJobs return list of all jobs that are pending execution and are of type "bump".
func GetPendingBumpJobs(s config.Storage) ([]v1.Job, error) {
	return FilterJobs(s, FilterByState(v1.PendingJobState), FilterByType(v1.BumpJobType))
}

// GetPendingJobs return list of all pending jobs.
func GetPendingJobs(s config.Storage) ([]v1.Job, error) {
	return FilterJobs(s, FilterByState(v1.PendingJobState))
}

type FilterFunc func(v1.Job) bool

func FilterByState(s v1.JobState) FilterFunc {
	return func(job v1.Job) bool {
		return job.Status.State == s
	}
}

func FilterByType(s v1.JobType) FilterFunc {
	return func(job v1.Job) bool {
		return job.Spec.Type == s
	}
}

func GetJobByName(s config.Storage, name string) (*v1.Job, error) {
	jobBytes, err := s.Get(name)
	if err != nil {
		return nil, err
	}
	var job v1.Job
	if err := json.Unmarshal(jobBytes, &job); err != nil {
		return nil, err
	}
	return &job, nil
}

// FilterJobs is used to map/reduce on list of all jobs.
func FilterJobs(s config.Storage, filters ...FilterFunc) ([]v1.Job, error) {
	jobs, err := s.List("")
	if err != nil {
		return nil, err
	}
	result := []v1.Job{}
	for i := range jobs {
		jobBytes, err := s.Get(jobs[i])
		if err != nil {
			return nil, err
		}
		var job v1.Job
		if err := json.Unmarshal(jobBytes, &job); err != nil {
			return nil, err
		}
		hasMatch := true
		for _, fn := range filters {
			if !fn(job) {
				hasMatch = false
			}
		}
		if !hasMatch {
			continue
		}
		result = append(result, job)
	}
	return result, nil
}
