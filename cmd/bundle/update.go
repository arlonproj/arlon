package bundle

import (
	"fmt"

	"github.com/argoproj/argo-cd/v2/util/cli"
	"github.com/arlonproj/arlon/pkg/bundle"
	"github.com/arlonproj/arlon/pkg/gitrepo"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func updateBundleCommand() *cobra.Command {
	var clientConfig clientcmd.ClientConfig
	var ns string
	var fromFile string
	var repoUrl string
	var repoAlias string
	var repoPath string
	var desc string
	var tags string
	command := &cobra.Command{
		Use:   "update",
		Short: "update configuration bundle",
		Long:  "update configuration bundle",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			if fromFile == "" && repoUrl == "" && repoAlias != "" {
				var err error
				repoUrl, err = gitrepo.GetRepoUrl(repoAlias)
				if err != nil {
					return err
				}
			}
			config, err := clientConfig.ClientConfig()
			if err != nil {
				return fmt.Errorf("failed to get k8s client config: %s", err)
			}
			kubeClient := kubernetes.NewForConfigOrDie(config)
			return bundle.Update(kubeClient, ns, args[0], fromFile, repoUrl, repoPath, desc, tags)
		},
	}
	clientConfig = cli.AddKubectlFlagsToCmd(command)
	command.Flags().StringVar(&ns, "ns", "arlon", "the arlon namespace")
	command.Flags().StringVar(&fromFile, "from-file", "", "update static bundle from this file")
	command.Flags().StringVar(&repoUrl, "repo-url", "", "update a dynamic bundle from this repo URL")
	command.Flags().StringVar(&repoAlias, "repo-alias", "", "the git repository alias to use")
	command.Flags().StringVar(&repoPath, "repo-path", "", "optional path in repo specified by --from-repo")
	command.Flags().StringVar(&desc, "desc", "", "description")
	command.Flags().StringVar(&tags, "tags", "", "comma separated list of tags")
	command.MarkFlagsMutuallyExclusive("from-file", "repo-alias", "repo-url")
	return command
}
