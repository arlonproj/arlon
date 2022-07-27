package app

import (
	"fmt"
	"github.com/arlonproj/arlon/pkg/app"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
)

import "github.com/argoproj/argo-cd/v2/util/cli"

func listAppsCommand() *cobra.Command {
	var clientConfig clientcmd.ClientConfig
	var ns string
	command := &cobra.Command{
		Use:   "list",
		Short: "List apps",
		Long:  "List apps",
		RunE: func(c *cobra.Command, args []string) error {
			config, err := clientConfig.ClientConfig()
			if err != nil {
				return fmt.Errorf("failed to get k8s client config: %s", err)
			}
			return app.ListToStdout(config, ns)
		},
	}
	clientConfig = cli.AddKubectlFlagsToCmd(command)
	command.Flags().StringVar(&ns, "ns", "argocd", "the argo-cd namespace")
	return command
}
