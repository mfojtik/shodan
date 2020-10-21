package v1

type JobType string

var (
	BumpJobType JobType = "bump"
)

type Job struct {
	Name string `json:"name"`

	// Type of the job.
	Type JobType `json:"type"`

	Params []string  `json:"params"`
	Status JobStatus `json:"status"`
}

type JobState string

var (
	PendingJobState  JobState = "pending"
	RunningJobState  JobState = "running"
	FinishedJobState JobState = "finished"
)

type JobStatus struct {
	// State represents current job state
	State JobState `json:"state"`
	// Message is optional indication we should give in source PR about the job state
	Message string `json:"message"`
}
