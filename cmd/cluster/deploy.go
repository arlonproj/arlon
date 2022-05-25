package cluster

import (
	_ "embed"
	"fmt"
	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/argoproj/argo-cd/v2/util/cli"
	"github.com/arlonproj/arlon/pkg/argocd"
	"github.com/arlonproj/arlon/pkg/cluster"
	"github.com/arlonproj/arlon/pkg/profile"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
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
		Use:   "deploy",
		Short: "deploy new cluster",
		Long:  "deploy new cluster",
		RunE: func(c *cobra.Command, args []string) error {
			conn, appIf := argocd.NewArgocdClientOrDie("").NewApplicationClientOrDie()
			defer conn.Close()
			config, err := clientConfig.ClientConfig()
			if err != nil {
				return fmt.Errorf("failed to get k8s client config: %s", err)
			}
			createInArgoCd := !outputYaml
			prof, err := profile.Get(config, profileName, arlonNs)
			if err != nil {
				return fmt.Errorf("failed to get profile: %s", err)
			}
			rootApp, err := cluster.Create(appIf, config, argocdNs, arlonNs,
				clusterName, repoUrl, repoBranch, basePath, clusterSpecName,
				prof, createInArgoCd, config.Host)
			if err != nil {
				return fmt.Errorf("failed to create cluster: %s", err)
			}
			if outputYaml {
				scheme := runtime.NewScheme()
				if err := v1alpha1.AddToScheme(scheme); err != nil {
					return fmt.Errorf("failed to add scheme: %s", err)
				}
				s := json.NewSerializerWithOptions(json.DefaultMetaFactory,
					scheme, scheme, json.SerializerOptions{
						Yaml:   true,
						Pretty: true,
						Strict: false,
					})
				err = s.Encode(rootApp, os.Stdout)
				if err != nil {
					return fmt.Errorf("failed to serialize app resource: %s", err)
				}
			}
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
	command.Flags().StringVar(&basePath, "repo-path", "clusters", "the git repository base path (cluster subdirectory will be created under this)")
	command.Flags().BoolVar(&outputYaml, "output-yaml", false, "output root application YAML instead of deploying to ArgoCD")
	command.MarkFlagRequired("repo-url")
	command.MarkFlagRequired("cluster-name")
	command.MarkFlagRequired("profile")
	command.MarkFlagRequired("cluster-spec")
	return command
}
