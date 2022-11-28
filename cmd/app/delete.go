package app

import (
	_ "embed"
	"fmt"
	"github.com/argoproj/argo-cd/v2/util/cli"
	"github.com/arlonproj/arlon/pkg/app"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
)

func deleteAppCommand() *cobra.Command {
	var clientConfig clientcmd.ClientConfig
	var argocdNs string
	var appName string
	command := &cobra.Command{
		Use:   "delete appName",
		Short: "delete existing Arlon app",
		Long:  "delete existing Arlon app",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			appName = args[0]
			config, err := clientConfig.ClientConfig()
			if err != nil {
				return fmt.Errorf("failed to get k8s client config: %s", err)
			}
			if err = app.Delete(config, argocdNs, appName); err != nil {
				return fmt.Errorf("failed to delete arlon app: %s", err)
			}
			return nil
		},
	}
	clientConfig = cli.AddKubectlFlagsToCmd(command)
	command.Flags().StringVar(&argocdNs, "argocd-ns", "argocd", "the argocd namespace")
	return command
}
