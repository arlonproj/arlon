package profile

import (
	"arlon.io/arlon/pkg/profile"
	"fmt"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

import "github.com/argoproj/argo-cd/v2/util/cli"

func createProfileCommand() *cobra.Command {
	var clientConfig clientcmd.ClientConfig
	var arlonNs string
	var argocdNs string
	var desc string
	var bundles string
	var tags string
	var repoUrl string
	var repoPath string
	var repoBranch string
	command := &cobra.Command{
		Use:               "create",
		Short:             "Create profile",
		Long:              "Create profile",
		Args: cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			config, err := clientConfig.ClientConfig()
			if err != nil {
				return fmt.Errorf("failed to get k8s client config: %s", err)
			}
			kubeClient := kubernetes.NewForConfigOrDie(config)
			return profile.Create(kubeClient, argocdNs, arlonNs, args[0], repoUrl,
				repoPath, repoBranch, bundles, desc, tags)
		},
	}
	clientConfig = cli.AddKubectlFlagsToCmd(command)
	command.Flags().StringVar(&arlonNs, "arlon-ns", "arlon", "the arlon namespace")
	command.Flags().StringVar(&argocdNs, "argocd-ns", "argocd", "the ArgoCD namespace")
	command.Flags().StringVar(&desc, "desc", "", "description")
	command.Flags().StringVar(&bundles, "bundles", "", "comma separated list of bundles")
	command.Flags().StringVar(&tags, "tags", "", "comma separated list of tags")
	command.Flags().StringVar(&repoUrl, "repo-url", "", "create a dynamic profile and store in specified git repository")
	command.Flags().StringVar(&repoPath, "repo-path", "", "optional git base path for dynamic profile. The profile directory will be created under this.")
	command.Flags().StringVar(&repoBranch, "repo-branch", "main", "optional git branch for dynamic profile (requires --repo-url)")
	command.MarkFlagRequired("bundles")
	return command
}


