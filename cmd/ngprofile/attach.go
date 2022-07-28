package ngprofile

import (
	_ "embed"
	"fmt"
	"github.com/argoproj/argo-cd/v2/util/cli"
	"github.com/arlonproj/arlon/pkg/argocd"
	"github.com/arlonproj/arlon/pkg/ngprofile"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
)

func attachCommand() *cobra.Command {
	var clientConfig clientcmd.ClientConfig
	var argocdNs string
	command := &cobra.Command{
		Use:   "attach <profilename> <clustername> [flags]",
		Short: "attach profile to existing cluster",
		Long:  "attach profile to existing cluster",
		Args:  cobra.ExactArgs(2),
		RunE: func(c *cobra.Command, args []string) error {
			argoIf := argocd.NewArgocdClientOrDie("")
			profName := args[0]
			clusterName := args[1]
			config, err := clientConfig.ClientConfig()
			if err != nil {
				return fmt.Errorf("failed to get k8s client config: %s", err)
			}
			profMap, err := ngprofile.Enumerate(config, argocdNs)
			if err != nil {
				return fmt.Errorf("failed to enumerate profiles: %s", err)
			}
			if profMap[profName] == nil {
				return fmt.Errorf("profile does not exist")
			}
			modified, err := ngprofile.AttachToCluster(argoIf, profName, clusterName)
			if err != nil {
				return fmt.Errorf("failed to attach profile to cluster: %s", err)
			}
			if !modified {
				fmt.Println("cluster was already using that profile")
				return nil
			}
			return nil
		},
	}
	clientConfig = cli.AddKubectlFlagsToCmd(command)
	command.Flags().StringVar(&argocdNs, "argocd-ns", "argocd", "the argocd namespace")
	return command
}
