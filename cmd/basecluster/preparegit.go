package basecluster

import (
	"fmt"

	"github.com/argoproj/argo-cd/v2/util/cli"
	"github.com/arlonproj/arlon/pkg/argocd"
	bcl "github.com/arlonproj/arlon/pkg/basecluster"
	"github.com/arlonproj/arlon/pkg/gitrepo"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
)

func prepareGitBaseClusterCommand() *cobra.Command {
	var clientConfig clientcmd.ClientConfig
	var argocdNs string
	var repoUrl string
	var repoPath string
	var repoAlias string
	var repoRevision string
	var casMin int
	var casMax int
	command := &cobra.Command{
		Use:   "preparegit --repo-url repoUrl [--repo-revision revision] [--repo-path path]",
		Short: "prepare base cluster directory in git",
		Long:  "prepare base cluster directory in git",
		RunE: func(c *cobra.Command, args []string) error {
			if repoUrl == "" {
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
			_, creds, err := argocd.GetKubeclientAndRepoCreds(config, argocdNs, repoUrl)
			if err != nil {
				return fmt.Errorf("failed to get repository credentials: %s", err)
			}
			clusterName, changed, err := bcl.PrepareGitDir(creds,
				repoUrl, repoRevision, repoPath, casMax, casMin)
			if err != nil {
				return fmt.Errorf("git preparation failed: %s", err)
			}
			if changed {
				fmt.Println("preparation successful, cluster name:", clusterName)
			} else {
				fmt.Println("the files for cluster <",
					clusterName,
					"> are already compliant as base cluster, no preparation necessary")
			}
			return nil
		},
	}
	clientConfig = cli.AddKubectlFlagsToCmd(command)
	command.Flags().StringVar(&argocdNs, "argocd-ns", "argocd", "the argocd namespace")
	command.Flags().StringVar(&repoUrl, "repo-url", "", "the git repository url for base cluster directory")
	command.Flags().StringVar(&repoAlias, "repo-alias", gitrepo.RepoDefaultCtx, "git repository alias to use")
	command.Flags().StringVar(&repoRevision, "repo-revision", "main", "the git revision for base cluster directory")
	command.Flags().StringVar(&repoPath, "repo-path", "", "the git repository path for base cluster directory")
	command.Flags().IntVar(&casMin, "cas-min", 1, "set minimum number of nodes for capi-cluster autoscaler, for MachineDeployment based clusters")
	command.Flags().IntVar(&casMax, "cas-max", 9, "set maximum number of nodes for capi-cluster autoscaler, for MachineDeployment based clusters")
	command.MarkFlagsMutuallyExclusive("repo-url", "repo-alias")
	return command
}
