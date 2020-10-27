package config

import (
	"context"

	"github.com/spf13/pflag"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

// NewGithubClient returns a client that can be used to talk to Github API using the Github Access Token
func (o *CommonOptions) NewGithubClient(ctx context.Context) *github.Client {
	return github.NewClient(oauth2.NewClient(ctx, oauth2.StaticTokenSource(&oauth2.Token{AccessToken: o.githubAccessToken})))
}

// AddGithubFlags will add flags needed to communicate with GitHub
func AddGithubFlags(fs *pflag.FlagSet) {
	fs.StringVar(&globalConfig.githubAccessToken, "github-access-token", "", "Github Access Token")
}

func AddKubeConfigFlags(fs *pflag.FlagSet) {
	fs.StringVar(&globalConfig.kubeconfigPath, "kubeconfig", "", "Path to kubeconfig file (optional)")
}
