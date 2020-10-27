package config

import (
	"fmt"
	"os"

	client "k8s.io/client-go/kubernetes"

	"github.com/openshift/library-go/pkg/controller/factory"

	"github.com/mfojtik/shodan/pkg/storage/informers"

	"github.com/mfojtik/shodan/pkg/storage/informers/jobs"

	clientconfig "github.com/openshift/library-go/pkg/config/client"
	"github.com/openshift/library-go/pkg/operator/events"
)

var globalConfig = &CommonOptions{}

// CommonOptions are common for every controller shodan runs.
// That include interface to Github API and also interface to persistence layer (storage).
type CommonOptions struct {
	Recorder events.Recorder
	Client   client.Interface
	Storage  Storage

	kubeconfigPath    string
	githubAccessToken string
	boltPath          string

	storageInformers []informers.StorageInformer
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

// Informers return a list of factory informers that can be consumed by controllers.
// This provides a faster way to reconcile controllers when the persisted data change.
func (o *CommonOptions) Informers() []factory.Informer {
	ret := []factory.Informer{}
	for i := range o.storageInformers {
		ret = append(ret, o.storageInformers[i].(factory.Informer))
	}
	return ret
}

// CompleteCommonOptions will finalize the common options.
func (o *CommonOptions) CompleteCommonOptions() error {
	o.Recorder = events.NewLoggingEventRecorder("shodan")

	if len(globalConfig.githubAccessToken) == 0 {
		o.githubAccessToken = os.Getenv("GITHUB_TOKEN")
	} else {
		o.githubAccessToken = globalConfig.githubAccessToken
	}

	o.storageInformers = []informers.StorageInformer{
		jobs.New(),
	}
	if err := o.initializeBoltDB(globalConfig.boltPath, o.storageInformers...); err != nil {
		return err
	}

	restConfig, err := clientconfig.GetKubeConfigOrInClusterConfig(globalConfig.kubeconfigPath, &clientconfig.ClientConnectionOverrides{})
	if err != nil {
		return fmt.Errorf("unable to configure kubernetes client, please provide kubeconfig file via --kubeconfig flag")
	}

	o.Client, err = client.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	return nil
}
