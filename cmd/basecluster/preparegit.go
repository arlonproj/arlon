package basecluster

import (
	"fmt"
	"github.com/argoproj/argo-cd/v2/util/cli"
	bcl "github.com/arlonproj/arlon/pkg/basecluster"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
)

func prepareGitBaseClusterCommand() *cobra.Command {
	var clientConfig clientcmd.ClientConfig
	var argocdNs string
	var repoUrl string
	var repoPath string
	var repoRevision string
	command := &cobra.Command{
		Use:   "preparegit --repo-url repoUrl [--repo-revision revision] [--repo-path path]",
		Short: "prepare base cluster directory in git",
		Long:  "prepare base cluster directory in git",
		RunE: func(c *cobra.Command, args []string) error {
			config, err := clientConfig.ClientConfig()
			if err != nil {
				return fmt.Errorf("failed to get k8s client config: %s", err)
			}
			clusterName, err := bcl.PrepareGitDir(config, argocdNs, repoUrl,
				repoRevision, repoPath)
			if err != nil {
				return err
			}
			fmt.Println("preparation successful, cluster name:", clusterName)
			return nil
		},
	}
	clientConfig = cli.AddKubectlFlagsToCmd(command)
	command.Flags().StringVar(&argocdNs, "argocd-ns", "argocd", "the argocd namespace")
	command.Flags().StringVar(&repoUrl, "repo-url", "", "the git repository url for base cluster directory")
	command.Flags().StringVar(&repoRevision, "repo-revision", "main", "the git revision for base cluster directory")
	command.Flags().StringVar(&repoPath, "repo-path", "", "the git repository path for base cluster directory")
	command.MarkFlagRequired("repo-url")
	return command
}
