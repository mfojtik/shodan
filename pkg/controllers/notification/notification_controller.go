package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	v1 "github.com/mfojtik/shodan/pkg/api/job/v1"

	"github.com/google/go-github/github"

	"github.com/mfojtik/shodan/pkg/config"

	"k8s.io/klog/v2"

	"github.com/openshift/library-go/pkg/controller/factory"
	"github.com/openshift/library-go/pkg/operator/events"
)

type controller struct {
	options config.CommonOptions
}

func NewController(options config.CommonOptions, recorder events.Recorder) factory.Controller {
	c := &controller{
		options: options,
	}
	return factory.New().ResyncEvery(30*time.Second).WithSync(c.sync).ToController("NotificationController", recorder)
}

func (c *controller) sync(ctx context.Context, factoryCtx factory.SyncContext) error {
	klog.Info("Syncing notifications ...")
	client := c.options.NewGithubClient(ctx)

	notifications, _, err := client.Activity.ListNotifications(ctx, &github.NotificationListOptions{
		All:           false,
		Participating: true,
		//Since:         time.Time{},
		//Before:        time.Time{},
	})
	if err != nil {
		return err
	}
	for _, n := range notifications {
		// only github mention notifications
		if n.GetReason() != "mention" {
			continue
		}
		// only notifications we have not marked as seen before
		if !n.GetUnread() {
			continue
		}

		// parse the metadata from github notification subject (source repo, issue, comment ID, etc.)
		notification, err := c.getNotificationFromSubject(ctx, client, n.GetSubject())
		if err != nil {
			return err
		}

		// construct a job we store in a config map
		// the config map name is a key that include owner-repo-issueID-commentID
		job := v1.Job{
			Type:   determineJobType(notification.message),
			Params: parseParameters(notification.message),
			Status: v1.JobStatus{
				State: v1.PendingJobState,
			},
		}

		// if we get unrecognized command, mark the job as finished, so the finished jobs controller can report
		// failure in a comment.
		if len(job.Type) == 0 {
			job.Status.State = v1.FinishedJobState
			job.Status.Message = fmt.Sprintf("Sorry human, I don't recognize this command.")
			continue
		}

		list, _ := c.options.Storage.List("")
		klog.Infof("list: %#v", list)

		res, err := c.options.Storage.Get(notification.toConfigMapName())
		klog.Infof("get: %s", string(res))
		if err == nil {
			// we already track this notification
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

		klog.Infof("store: %s", string(jobJSON))
		if err := c.options.Storage.Set(notification.toConfigMapName(), jobJSON); err != nil {
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
}

func (n notification) toConfigMapName() string {
	return fmt.Sprintf("%s-%s-%s-%s", n.ownerName, n.repositoryName, n.issueID, n.commentID)
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
