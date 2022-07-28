package app

import (
	"fmt"
	"github.com/arlonproj/arlon/pkg/app"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
)

import "github.com/argoproj/argo-cd/v2/util/cli"

func removeFromProfileCommand() *cobra.Command {
	var clientConfig clientcmd.ClientConfig
	var ns string
	command := &cobra.Command{
		Use:   "removefromprofile <appName> <profileName> [flags]",
		Short: "Remove app from next-gen profile",
		Long:  "Remove app from next-gen profile",
		Args:  cobra.ExactArgs(2),
		RunE: func(c *cobra.Command, args []string) error {
			appName := args[0]
			profName := args[1]
			config, err := clientConfig.ClientConfig()
			if err != nil {
				return fmt.Errorf("failed to get k8s client config: %s", err)
			}
			found, err := app.RemoveFromProfile(config, ns, appName, profName)
			if err != nil {
				return fmt.Errorf("failed to add app to profile: %s", err)
			}
			if !found {
				fmt.Println("no change: app was not in profile")
			}
			return nil
		},
	}
	clientConfig = cli.AddKubectlFlagsToCmd(command)
	command.Flags().StringVar(&ns, "ns", "argocd", "the argo-cd namespace")
	return command
}
