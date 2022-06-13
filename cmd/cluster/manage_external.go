package cluster

import (
	_ "embed"
	"fmt"
	"github.com/argoproj/argo-cd/v2/util/cli"
	"github.com/arlonproj/arlon/pkg/argocd"
	"github.com/arlonproj/arlon/pkg/cluster"
	"github.com/arlonproj/arlon/pkg/profile"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
)

func manageClusterCommand() *cobra.Command {
	var clientConfig clientcmd.ClientConfig
	var argocdNs string
	var arlonNs string
	command := &cobra.Command{
		Use:   "manage <clustername> <profilename> [flags]",
		Short: "manage external cluster with specified profile",
		Long:  "manage external cluster with specified profile",
		Args:  cobra.ExactArgs(2),
		RunE: func(c *cobra.Command, args []string) error {
			argoIf := argocd.NewArgocdClientOrDie("")
			config, err := clientConfig.ClientConfig()
			if err != nil {
				return fmt.Errorf("failed to get k8s client config: %s", err)
			}
			clusterName := args[0]
			profileName := args[1]
			prof, err := profile.Get(config, profileName, arlonNs)
			if err != nil {
				return fmt.Errorf("failed to get profile: %s", err)
			}
			err = cluster.ManageExternal(argoIf, config, argocdNs, clusterName, prof)
			if err != nil {
				return fmt.Errorf("failed to manage cluster: %s", err)
			}
			return nil
		},
	}
	clientConfig = cli.AddKubectlFlagsToCmd(command)
	command.Flags().StringVar(&argocdNs, "argocd-ns", "argocd", "the argocd namespace")
	command.Flags().StringVar(&arlonNs, "arlon-ns", "arlon", "the arlon namespace")
	return command
}
