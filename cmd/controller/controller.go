package controller

import (
	"github.com/arlonproj/arlon/pkg/controller"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var argocdConfigPath string

	command := &cobra.Command{
		Use:               "controller",
		Short:             "Run the Arlon controller",
		Long:              "Run the Arlon controller",
		DisableAutoGenTag: true,
		Run: func(c *cobra.Command, args []string) {
			controller.StartController(argocdConfigPath, metricsAddr, probeAddr, enableLeaderElection)
		},
	}
	command.Flags().StringVar(&argocdConfigPath, "argocd-config-path", "", "argocd configuration file path")
	command.Flags().StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	command.Flags().StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	command.Flags().BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	return command
}
