package profile

import (
	"fmt"
	arlonv1 "github.com/arlonproj/arlon/api/v1"
	"github.com/arlonproj/arlon/pkg/profile"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
	"strings"
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
	var repoBasePath string
	var repoBranch string
	var overrides []string
	command := &cobra.Command{
		Use:   "create",
		Short: "Create profile",
		Long:  "Create profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			config, err := clientConfig.ClientConfig()
			if err != nil {
				return fmt.Errorf("failed to get k8s client config: %s", err)
			}
			o, err := processOverrides(overrides)
			if err != nil {
				return fmt.Errorf("failed to process overrides: %s", err)
			}
			return profile.Create(config, argocdNs, arlonNs, args[0], repoUrl,
				repoBasePath, repoBranch, bundles, desc, tags, o)
		},
	}
	clientConfig = cli.AddKubectlFlagsToCmd(command)
	command.Flags().StringVar(&arlonNs, "arlon-ns", "arlon", "the arlon namespace")
	command.Flags().StringVar(&argocdNs, "argocd-ns", "argocd", "the ArgoCD namespace")
	command.Flags().StringVar(&desc, "desc", "", "description")
	command.Flags().StringVar(&bundles, "bundles", "", "comma separated list of bundles")
	command.Flags().StringVar(&tags, "tags", "", "comma separated list of tags")
	command.Flags().StringVar(&repoUrl, "repo-url", "", "create a dynamic profile and store in specified git repository")
	command.Flags().StringVar(&repoBasePath, "repo-base-path", "profiles", "optional git base path for dynamic profile. The profile directory will be created under this.")
	command.Flags().StringVar(&repoBranch, "repo-branch", "main", "optional git branch for dynamic profile (requires --repo-url)")
	command.Flags().StringArrayVarP(&overrides, "param", "p", nil, "a single parameter override of the form bundle,key,value ... can be repeated")
	command.MarkFlagRequired("bundles")
	return command
}

func processOverrides(overrides []string) (res []arlonv1.Override, err error) {
	for _, o := range overrides {
		items := strings.Split(o, ",")
		if len(items) != 3 {
			return nil, fmt.Errorf("malformed override parameter, it should be a triple")
		}
		res = append(res, arlonv1.Override{
			Bundle: items[0],
			Key:    items[1],
			Value:  items[2],
		})
	}
	return
}
