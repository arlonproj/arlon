package clusterspec

import (
	"fmt"
	"github.com/arlonproj/arlon/pkg/clusterspec"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

import "github.com/argoproj/argo-cd/v2/util/cli"

func deleteClusterspecCommand() *cobra.Command {
	var clientConfig clientcmd.ClientConfig
	var ns string
	command := &cobra.Command{
		Use:   "delete",
		Short: "Delete clusterspec",
		Long:  "Delete clusterspec",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			config, err := clientConfig.ClientConfig()
			if err != nil {
				return fmt.Errorf("failed to get k8s client config: %s", err)
			}
			return deleteClusterspec(config, ns, args[0])
		},
	}
	clientConfig = cli.AddKubectlFlagsToCmd(command)
	command.Flags().StringVar(&ns, "ns", "arlon", "the arlon namespace")
	return command
}

func deleteClusterspec(config *restclient.Config, ns string, clusterspecName string) error {
	kubeClient := kubernetes.NewForConfigOrDie(config)
	return clusterspec.Delete(kubeClient, ns, clusterspecName)
}
