package app

import (
	"fmt"
	"github.com/arlonproj/arlon/pkg/app"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
)

import "github.com/argoproj/argo-cd/v2/util/cli"

func addToProfileCommand() *cobra.Command {
	var clientConfig clientcmd.ClientConfig
	var ns string
	command := &cobra.Command{
		Use:   "addtoprofile <appName> <profileName> [flags]",
		Short: "Add app to next-gen profile",
		Long:  "Add app to next-gen profile",
		Args:  cobra.ExactArgs(2),
		RunE: func(c *cobra.Command, args []string) error {
			appName := args[0]
			profName := args[1]
			config, err := clientConfig.ClientConfig()
			if err != nil {
				return fmt.Errorf("failed to get k8s client config: %s", err)
			}
			modified, err := app.AddToProfile(config, ns, appName, profName)
			if err != nil {
				return fmt.Errorf("failed to add app to profile: %s", err)
			}
			if !modified {
				fmt.Println("no change: app was already in profile")
			}
			return nil
		},
	}
	clientConfig = cli.AddKubectlFlagsToCmd(command)
	command.Flags().StringVar(&ns, "ns", "argocd", "the argo-cd namespace")
	return command
}
