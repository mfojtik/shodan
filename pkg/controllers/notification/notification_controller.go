package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	v1 "github.com/mfojtik/shodan/pkg/api/job/v1"
	"github.com/mfojtik/shodan/pkg/storage"

	"github.com/google/go-github/github"

	"github.com/mfojtik/shodan/pkg/config"

	"k8s.io/klog/v2"

	"github.com/openshift/library-go/pkg/controller/factory"
	"github.com/openshift/library-go/pkg/operator/events"
)

type controller struct {
	options config.CommonOptions
}

// NewController returns a instance of notification controller. This controller watches for Github mentions and for each new mention
// create a Job object in storage that other controller can consume.
func NewController(options config.CommonOptions, recorder events.Recorder) factory.Controller {
	c := &controller{
		options: options,
	}
	return factory.New().ResyncEvery(30*time.Second).WithSync(c.sync).WithInformers(c.options.Informers()...).ToController("NotificationController", recorder)
}

func (c *controller) sync(ctx context.Context, factoryCtx factory.SyncContext) error {
	client := c.options.NewGithubClient(ctx)

	lastSeenNotification, numJobs, err := storage.GetJobsStats(c.options.Storage)
	if err != nil {
		return err
	}

	klog.Infof("Checking Github notifications since %s (%d active jobs) ...", lastSeenNotification, numJobs)
	notifications, _, err := client.Activity.ListNotifications(ctx, &github.NotificationListOptions{
		All:           false,
		Participating: true,
		Since:         lastSeenNotification,
	})
	if err != nil {
		return err
	}

	for i := range notifications {
		// only github mention notifications
		if notifications[i].GetReason() != "mention" {
			continue
		}
		// only notifications we have not marked as seen before
		if !notifications[i].GetUnread() {
			continue
		}

		// parse the metadata from github n subject (source repo, issue, comment ID, etc.)
		n, err := c.getNotificationFromSubject(ctx, client, notifications[i].GetSubject())
		if err != nil {
			return err
		}

		// we need to store when the notification was created so we can list only new next time
		n.updatedAt = notifications[i].GetUpdatedAt()

		klog.Infof("Processing Github notification %q", n.toJobName())

		// construct a job we store in a config map
		// the config map name is a key that include owner-repo-issueID-commentID
		job := v1.Job{
			Name: n.toJobName(),
			Spec: v1.JobSpec{
				Type:       determineJobType(n.message),
				Params:     parseParameters(n.message),
				Repository: n.repositoryName,
				Owner:      n.ownerName,
				IssueID:    n.issueID,
				CommentID:  n.commentID,
			},
			Status: v1.JobStatus{
				State: v1.PendingJobState,
			},
		}

		// if we get unrecognized command, mark the job as finished, so the finished jobs controller can report
		// failure in a comment.
		if len(job.Spec.Type) == 0 {
			job.Status.State = v1.FinishedJobState
			job.Status.Message = fmt.Sprintf("Sorry human, I don't recognize this command.")
			continue
		}

		res, err := c.options.Storage.Get(n.toJobName())
		if err == nil {
			klog.Infof("Job %q already exists: %s", n.toJobName(), string(res))
			continue
		}
		if err != config.StorageNotFoundErr {
			// oops something bad happened in bolt
			return err
		}

		jobJSON, err := json.Marshal(job)
		if err != nil {
			return err
		}

		klog.Infof("Job %q created: %s", n.toJobName(), string(jobJSON))
		if err := c.options.Storage.Set(n.toJobName(), jobJSON); err != nil {
			return err
		}
	}
	return nil
}

type notification struct {
	repositoryName string
	ownerName      string
	issueID        string
	commentID      string
	message        string
	updatedAt      time.Time
}

func (n notification) toJobName() string {
	return fmt.Sprintf("%s-%s-%s-%s-%d", n.ownerName, n.repositoryName, n.issueID, n.commentID, n.updatedAt.Unix())
}

func (c *controller) getNotificationFromSubject(ctx context.Context, ghClient *github.Client, s *github.NotificationSubject) (*notification, error) {
	commentURLParts := strings.Split(strings.TrimPrefix(s.GetLatestCommentURL(), "https://api.github.com/repos/"), "/")
	if len(commentURLParts) != 5 {
		return nil, fmt.Errorf("invalid last comment URL %q", s.GetLatestCommentURL())
	}
	issueURLParts := strings.Split(strings.TrimPrefix(s.GetURL(), "https://api.github.com/repos/"), "/")
	if len(issueURLParts) != 4 {
		return nil, fmt.Errorf("invalid issue URL %q", s.GetURL())
	}
	commentBody, err := c.getCommentMessage(ctx, ghClient, s.GetLatestCommentURL())
	if err != nil {
		return nil, err
	}
	return &notification{
		repositoryName: commentURLParts[1],
		ownerName:      commentURLParts[0],
		issueID:        issueURLParts[3],
		commentID:      commentURLParts[4],
		message:        commentBody,
	}, nil
}

func trimCommentBody(body string) string {
	comment := strings.TrimSpace(body)
	return strings.TrimPrefix(comment, "@shodan-bot ")
}

func parseParameters(comment string) []string {
	if parts := strings.Split(trimCommentBody(comment), " "); len(parts) == 1 {
		return []string{}
	} else {
		return parts[1:]
	}
}

func determineJobType(comment string) v1.JobType {
	parts := strings.Split(trimCommentBody(comment), " ")
	if len(parts) == 0 {
		return ""
	}
	switch parts[0] {
	case "bump":
		return v1.BumpJobType
	default:
		return ""
	}
}

func (c *controller) getCommentMessage(ctx context.Context, ghClient *github.Client, commentURL string) (string, error) {
	parts := strings.Split(strings.TrimPrefix(commentURL, "https://api.github.com/repos/"), "/")
	if len(parts) != 5 {
		return "", fmt.Errorf("invalid commentURL: %q", commentURL)
	}
	commentID, err := strconv.Atoi(parts[4])
	if err != nil {
		return "", err
	}
	comment, _, err := ghClient.Issues.GetComment(ctx, parts[0], parts[1], int64(commentID))
	if err != nil {
		return "", err
	}
	return comment.GetBody(), nil
}
