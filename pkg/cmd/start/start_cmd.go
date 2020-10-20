package start

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/klog/v2"
)

// startOptions holds values to drive the start command.
type startOptions struct {
}

// NewStartCommand creates a render command.
func NewStartCommand() *cobra.Command {
	startOpts := startOptions{}
	cmd := &cobra.Command{
		Use:   "start",
		Short: "The bot will start listening to GitHub notifications and take actions",
		Run: func(cmd *cobra.Command, args []string) {
			if err := startOpts.Validate(); err != nil {
				klog.Fatal(err)
			}
			if err := startOpts.Complete(); err != nil {
				klog.Fatal(err)
			}
			if err := startOpts.Run(); err != nil {
				klog.Fatal(err)
			}
		},
	}

	startOpts.AddFlags(cmd.Flags())

	return cmd
}

func (r *startOptions) AddFlags(fs *pflag.FlagSet) {
	//fs.StringVar(&r.lockHostPath, "manifest-lock-host-path", r.lockHostPath, "A host path mounted into the apiserver pods to hold lock.")
}

func (r *startOptions) Validate() error {
	return nil
}

func (r *startOptions) Complete() error {
	return nil
}

func (r *startOptions) Run() error {
	return nil
}
