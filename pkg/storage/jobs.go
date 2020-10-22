package storage

import (
	"strings"
)

type JobMeta struct {
	// Repository is the source repository the job was created from.
	Repository string

	// Owner is the organization name where source repository exists.
	Owner string

	// IssueID represents the pull request or issue the comment was made.
	IssueID string

	// CommentID represents the comment used to trigger this job.
	CommentID string

	// CreatedAt represents the time the comment was created.
	CreatedAt string
}

// GetJobMeta returns a metadata encoded in a Job name.
func GetJobMeta(name string) *JobMeta {
	parts := strings.Split(name, "-")
	return &JobMeta{
		Repository: parts[1],
		Owner:      parts[0],
		IssueID:    parts[2],
		CommentID:  parts[3],
		CreatedAt:  parts[4],
	}
}
