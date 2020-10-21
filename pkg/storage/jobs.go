package storage

import (
	"strings"
)

type JobMeta struct {
	Repository string
	Owner      string
	IssueID    string
	CommentID  string
	CreatedAt  string
}

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
