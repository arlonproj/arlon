package bundle

import (
	"fmt"
	"github.com/arlonproj/arlon/pkg/bundle"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

import "github.com/argoproj/argo-cd/v2/util/cli"

func deleteBundleCommand() *cobra.Command {
	var clientConfig clientcmd.ClientConfig
	var ns string
	command := &cobra.Command{
		Use:   "delete",
		Short: "Delete configuration bundle",
		Long:  "Delete configuration bundle",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			config, err := clientConfig.ClientConfig()
			if err != nil {
				return fmt.Errorf("failed to get k8s client config: %s", err)
			}
			return deleteBundle(config, ns, args[0])
		},
	}
	clientConfig = cli.AddKubectlFlagsToCmd(command)
	command.Flags().StringVar(&ns, "ns", "arlon", "the arlon namespace")
	return command
}

func deleteBundle(config *restclient.Config, ns string, bundleName string) error {
	kubeClient := kubernetes.NewForConfigOrDie(config)
	return bundle.Delete(kubeClient, ns, bundleName)
}
