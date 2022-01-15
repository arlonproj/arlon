package cluster

import (
	"arlon.io/arlon/pkg/argocd"
	"arlon.io/arlon/pkg/cluster"
	"context"
	_ "embed"
	"fmt"
	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	applicationpkg "github.com/argoproj/argo-cd/v2/pkg/apiclient/application"
	"github.com/argoproj/argo-cd/v2/util/cli"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"os"
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
	var outputYaml bool
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
			if outputYaml {
				scheme := runtime.NewScheme()
				v1alpha1.AddToScheme(scheme)
				s := json.NewSerializerWithOptions(json.DefaultMetaFactory, scheme, scheme, json.SerializerOptions{
					Yaml:   true,
					Pretty: true,
					Strict: false,
				})
				err = s.Encode(rootApp, os.Stdout)
				if err != nil {
					return fmt.Errorf("failed to serialize app resource: %s", err)
				}
				return nil
			} else {
				conn, appIf := argocd.NewArgocdClientOrDie().NewApplicationClientOrDie()
				defer conn.Close()
				appCreateRequest := applicationpkg.ApplicationCreateRequest{
					Application: *rootApp,
				}
				_, err := appIf.Create(context.Background(), &appCreateRequest)
				if err != nil {
					return fmt.Errorf("failed to create ArgoCD root application: %s", err)
				}
				return nil
			}
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
	command.Flags().BoolVar(&outputYaml, "output-yaml", false, "output root application YAML instead of deploying to ArgoCD")
	command.MarkFlagRequired("repo-url")
	command.MarkFlagRequired("cluster-name")
	return command
}

