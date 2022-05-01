package bundle

import (
	"fmt"
	"github.com/platform9/arlon/pkg/bundle"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

import "github.com/argoproj/argo-cd/v2/util/cli"

func createBundleCommand() *cobra.Command {
	var clientConfig clientcmd.ClientConfig
	var ns string
	var fromFile string
	var repoUrl string
	var repoPath string
	var repoRevision string
	var srcType string
	var desc string
	var tags string
	command := &cobra.Command{
		Use:   "create",
		Short: "Create configuration bundle",
		Long:  "Create configuration bundle",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			config, err := clientConfig.ClientConfig()
			if err != nil {
				return fmt.Errorf("failed to get k8s client config: %s", err)
			}
			kubeClient := kubernetes.NewForConfigOrDie(config)
			return bundle.Create(kubeClient, ns, args[0], fromFile, repoUrl,
				repoPath, repoRevision, srcType, desc, tags)
		},
	}
	clientConfig = cli.AddKubectlFlagsToCmd(command)
	command.Flags().StringVar(&ns, "ns", "arlon", "the arlon namespace")
	command.Flags().StringVar(&fromFile, "from-file", "", "create static bundle from this file")
	command.Flags().StringVar(&repoUrl, "repo-url", "", "create a dynamic bundle from this repo URL")
	command.Flags().StringVar(&repoPath, "repo-path", "", "optional path in repo specified by --from-repo")
	command.Flags().StringVar(&repoRevision, "repo-revision", "main", "revision")
	command.Flags().StringVar(&srcType, "srctype", "", "manifest source type (directory/helm/ksonnet/kustomize, empty means autodetect)")
	command.Flags().StringVar(&desc, "desc", "", "description")
	command.Flags().StringVar(&tags, "tags", "", "comma separated list of tags")
	return command
}
