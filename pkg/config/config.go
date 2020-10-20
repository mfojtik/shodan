package config

import (
	"fmt"
	"os"

	"github.com/openshift/library-go/pkg/operator/events"
)

var globalConfig = &CommonOptions{}

type CommonOptions struct {
	Recorder events.Recorder
	Storage  Storage

	githubAccessToken string
	boltPath          string
}

func (o *CommonOptions) ValidateCommonOptions() error {
	if len(globalConfig.githubAccessToken) == 0 {
		return fmt.Errorf("provide Github Access Token (either by --github-access-token or GITHUB_TOKEN env var)")
	}
	if len(globalConfig.boltPath) == 0 {
		return fmt.Errorf("boltdb path must be specified using --boltdb-path")
	}
	return nil
}

func (o *CommonOptions) CompleteCommonOptions() {
	o.Recorder = events.NewLoggingEventRecorder("shodan")

	if len(globalConfig.githubAccessToken) == 0 {
		o.githubAccessToken = os.Getenv("GITHUB_TOKEN")
	} else {
		o.githubAccessToken = globalConfig.githubAccessToken
	}

	if err := o.initializeBoltDB(globalConfig.boltPath); err != nil {
		panic(err)
	}
}
