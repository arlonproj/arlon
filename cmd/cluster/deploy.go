package cluster

import (
	"arlon.io/arlon/pkg/cluster"
	_ "embed"
	"fmt"
	"github.com/argoproj/argo-cd/v2/util/cli"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func deployClusterCommand() *cobra.Command {
	var clientConfig clientcmd.ClientConfig
	var argocdNs string
	var arlonNs string
	var repoUrl string
	var repoBranch string
	var basePath string
	var clusterName string
	var clusterSpecName string
	var profileName string
	command := &cobra.Command{
		Use:               "deploy",
		Short:             "DeployToGit cluster",
		Long:              "DeployToGit cluster",
		RunE: func(c *cobra.Command, args []string) error {
			config, err := clientConfig.ClientConfig()
			if err != nil {
				return fmt.Errorf("failed to get k8s client config: %s", err)
			}
			kubeClient := kubernetes.NewForConfigOrDie(config)
			rootApp, err := cluster.ConstructRootApp(kubeClient, argocdNs, arlonNs, clusterName, repoUrl, repoBranch, basePath, clusterSpecName)
			if err != nil {
				return fmt.Errorf("failed to construct roop app: %s", err)
			}
			err = cluster.DeployToGit(kubeClient, argocdNs, arlonNs, clusterName, repoUrl, repoBranch, basePath, profileName)
			if err != nil {
				return fmt.Errorf("failed to deploy git tree: %s", err)
			}
			buf, err := yaml.Marshal(rootApp)
			if err != nil {
				return fmt.Errorf("failed to marshal yaml from root app: %s", err)
			}
			fmt.Println(string(buf))
			return nil
		},
	}
	clientConfig = cli.AddKubectlFlagsToCmd(command)
	command.Flags().StringVar(&argocdNs, "argocd-ns", "argocd", "the argocd namespace")
	command.Flags().StringVar(&arlonNs, "arlon-ns", "arlon", "the arlon namespace")
	command.Flags().StringVar(&repoUrl, "repo-url", "", "the git repository url")
	command.Flags().StringVar(&repoBranch, "repo-branch", "main", "the git branch")
	command.Flags().StringVar(&clusterName, "cluster-name", "", "the cluster name")
	command.Flags().StringVar(&profileName, "profile", "", "the configuration profile to use")
	command.Flags().StringVar(&clusterSpecName, "cluster-spec", "", "the clusterspec to use")
	command.Flags().StringVar(&basePath, "path", "arlon", "the git repository base path")
	command.MarkFlagRequired("repo-url")
	command.MarkFlagRequired("cluster-name")
	return command
}

