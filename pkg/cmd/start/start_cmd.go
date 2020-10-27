package start

import (
	"context"

	"github.com/mfojtik/shodan/pkg/controllers/bumppod"

	"github.com/mfojtik/shodan/pkg/controllers/bump"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/klog/v2"

	"github.com/mfojtik/shodan/pkg/config"
	"github.com/mfojtik/shodan/pkg/controllers/notification"
)

// startOptions holds values to drive the start command.
type startOptions struct {
	config.CommonOptions
}

// NewStartCommand creates a render command.
func NewStartCommand(ctx context.Context) *cobra.Command {
	startOpts := startOptions{}
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Shodan will start listening to GitHub notifications",
		Run: func(cmd *cobra.Command, args []string) {
			if err := startOpts.Complete(); err != nil {
				klog.Exit(err)
			}
			if err := startOpts.Validate(); err != nil {
				klog.Exit(err)
			}
			if err := startOpts.Run(ctx); err != nil {
				klog.Exit(err)
			}
		},
	}

	startOpts.AddFlags(cmd.Flags())

	return cmd
}

func (r *startOptions) AddFlags(fs *pflag.FlagSet) {
	config.AddGithubFlags(fs)
	config.AddBoltFlags(fs)
	config.AddKubeConfigFlags(fs)
}

func (r *startOptions) Validate() error {
	if err := r.ValidateCommonOptions(); err != nil {
		return err
	}
	return nil
}

func (r *startOptions) Complete() error {
	return r.CompleteCommonOptions()
}

func (r *startOptions) Run(ctx context.Context) error {
	notificationController := notification.NewController(r.CommonOptions, r.Recorder)
	bumpController := bump.NewController(r.CommonOptions, r.Recorder)
	bumpPodController := bumppod.NewController(r.CommonOptions, r.Recorder)

	go notificationController.Run(ctx, 1)
	go bumpController.Run(ctx, 1)
	go bumpPodController.Run(ctx, 1)

	<-ctx.Done()
	return nil
}
