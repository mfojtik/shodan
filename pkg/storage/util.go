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

func PendingBumpJobs(s config.Storage) ([]v1.Job, error) {
	return FilterJobs(s, func(job v1.Job) bool {
		return job.Type == v1.BumpJobType && job.Status.State == v1.PendingJobState
	})
}

func PendingJobs(s config.Storage) ([]v1.Job, error) {
	return FilterJobs(s, func(job v1.Job) bool {
		return job.Status.State == v1.PendingJobState
	})
}

func FilterJobs(s config.Storage, fn func(job v1.Job) bool) ([]v1.Job, error) {
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
		if !fn(job) {
			continue
		}
		result = append(result, job)
	}
	return result, nil
}
