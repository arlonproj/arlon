package clustercontroller

import (
	"github.com/argoproj/argo-cd/v2/util/cli"
	"github.com/arlonproj/arlon/pkg/controller"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
)

func NewCommand() *cobra.Command {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var argocdConfigPath string
	var clientConfig clientcmd.ClientConfig

	command := &cobra.Command{
		Use:               "clustercontroller",
		Short:             "Run the Arlon Cluster controller",
		Long:              "Run the Arlon Cluster controller",
		DisableAutoGenTag: true,
		RunE: func(c *cobra.Command, args []string) error {
			config, err := clientConfig.ClientConfig()
			if err != nil {
				return err
			}
			controller.StartClusterController(config,
				argocdConfigPath, metricsAddr,
				probeAddr, enableLeaderElection)
			return nil
		},
	}
	clientConfig = cli.AddKubectlFlagsToCmd(command)
	command.Flags().StringVar(&argocdConfigPath, "argocd-config-path", "", "argocd configuration file path")
	command.Flags().StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	command.Flags().StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	command.Flags().BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	return command
}
