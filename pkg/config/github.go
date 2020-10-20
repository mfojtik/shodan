package config

import (
	"context"

	"github.com/spf13/pflag"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

func (o *CommonOptions) NewGithubClient(ctx context.Context) *github.Client {
	return github.NewClient(oauth2.NewClient(ctx, oauth2.StaticTokenSource(&oauth2.Token{AccessToken: o.githubAccessToken})))
}

func AddGithubFlags(fs *pflag.FlagSet) {
	fs.StringVar(&globalConfig.githubAccessToken, "github-access-token", "", "Github Access Token")
}
