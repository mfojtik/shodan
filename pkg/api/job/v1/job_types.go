package v1

type JobType string

type Job struct {
	// Name is the name of the job.
	Name string `json:"name"`

	Spec JobSpec

	// Status contain details about the current state of the job.
	// This can be modified by any controller.
	Status JobStatus `json:"status"`
}

var (
	BumpJobType JobType = "bump"
)

type JobSpec struct {
	// Type of the job.
	// This is filled by notification controller.
	Type JobType `json:"type"`

	// Params contain parameters engineer passed to comment.
	// This could be a target repository name for opening pull requests or something else.
	// This is filled by notification controller.
	Params []string `json:"params"`

	// These metadata are set on job creation.
	Repository string `json:"repository"`
	Owner      string `json:"owner"`
	IssueID    string `json:"issue_id"`
	CommentID  string `json:"comment_id"`
}

// JobState represents the state the jobs current are.
type JobState string

var (
	// Pending means the job was accepted and is now pending to run after pre-conditions are met (like the PR is merged).
	PendingJobState JobState = "pending"

	// Running means the job is being handled by a controller at the moment.
	RunningJobState JobState = "running"

	// Finished means no controller should act on this job other than response controller to deliver the finished message
	// in source issue.
	FinishedJobState JobState = "finished"

	// DeleteJobState means the job is to be deleted as it was finished and the status was successfully
	// reported back in the original issue.
	DeleteJobState JobState = "delete"
)

type JobStatus struct {
	// State represents current job state
	State JobState `json:"state"`

	// BaseBranch represents a branch name to where the source pull request is opened against
	BaseBranch string `json:"base_branch"`

	// Message is optional indication we should give in source PR about the job state
	Message string `json:"message"`
}
