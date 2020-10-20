package config

import (
	"fmt"
	"os"

	"github.com/openshift/library-go/pkg/operator/events"

	"github.com/spf13/pflag"
)

var globalConfig = &CommonOptions{}

func AddConfigFlags(fs *pflag.FlagSet) {
	fs.StringVar(&globalConfig.GithubAccessToken, "github-access-token", "", "Github Access Token")
}

type CommonOptions struct {
	Recorder          events.Recorder
	GithubAccessToken string
}

func (o *CommonOptions) ValidateCommonOptions() error {
	if len(o.GithubAccessToken) == 0 {
		return fmt.Errorf("provide Github Access Token (either by --github-access-token or GITHUB_TOKEN env var)")
	}
	return nil
}

func (o *CommonOptions) CompleteCommonOptions() {
	o.Recorder = events.NewLoggingEventRecorder("shodan")

	if len(globalConfig.GithubAccessToken) == 0 {
		o.GithubAccessToken = os.Getenv("GITHUB_TOKEN")
	} else {
		o.GithubAccessToken = globalConfig.GithubAccessToken
	}
}
