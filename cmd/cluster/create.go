package cluster

import (
	_ "embed"
	"fmt"
	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/argoproj/argo-cd/v2/util/cli"
	"github.com/arlonproj/arlon/pkg/argocd"
	"github.com/arlonproj/arlon/pkg/cluster"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/tools/clientcmd"
	"os"
)

func createClusterCommand() *cobra.Command {
	var clientConfig clientcmd.ClientConfig
	var argocdNs string
	var arlonNs string
	var arlonRepoUrl string
	var arlonRepoRevision string
	var arlonRepoPath string
	var clusterRepoUrl string
	var clusterRepoRevision string
	var clusterRepoPath string
	var clusterName string
	var outputYaml bool
	command := &cobra.Command{
		Use:   "create",
		Short: "create new cluster from a base",
		Long:  "create new cluster from a base",
		RunE: func(c *cobra.Command, args []string) error {
			conn, appIf := argocd.NewArgocdClientOrDie("").NewApplicationClientOrDie()
			defer conn.Close()
			config, err := clientConfig.ClientConfig()
			if err != nil {
				return fmt.Errorf("failed to get k8s client config: %s", err)
			}
			createInArgoCd := !outputYaml
			arlonApp, err := cluster.Create(appIf, config, argocdNs, arlonNs,
				clusterName, arlonRepoUrl, arlonRepoRevision, arlonRepoPath, "",
				nil, createInArgoCd, config.Host)
			if err != nil {
				return fmt.Errorf("failed to create arlon app: %s", err)
			}
			clusterApp, err := cluster.CreateClusterApp(appIf, argocdNs,
				clusterName, clusterRepoUrl, clusterRepoRevision,
				clusterRepoPath, createInArgoCd)
			if err != nil {
				return fmt.Errorf("failed to create cluster app: %s", err)
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
				err = s.Encode(arlonApp, os.Stdout)
				if err != nil {
					return fmt.Errorf("failed to serialize app resource: %s", err)
				}
				fmt.Println("---")
				err = s.Encode(clusterApp, os.Stdout)
			}
			return nil
		},
	}
	clientConfig = cli.AddKubectlFlagsToCmd(command)
	command.Flags().StringVar(&argocdNs, "argocd-ns", "argocd", "the argocd namespace")
	command.Flags().StringVar(&arlonNs, "arlon-ns", "arlon", "the arlon namespace")
	command.Flags().StringVar(&arlonRepoUrl, "arlon-repo-url", "https://github.com/arlonproj/arlon.git", "the git repository url for arlon template")
	command.Flags().StringVar(&arlonRepoRevision, "arlon-repo-revision", "private/leb/gen2", "the git revision for arlon template")
	command.Flags().StringVar(&arlonRepoPath, "arlon-repo-path", "pkg/cluster/manifests", "the git repository path for arlon template")
	command.Flags().StringVar(&clusterRepoUrl, "repo-url", "https://github.com/clusterproj/cluster.git", "the git repository url for cluster template")
	command.Flags().StringVar(&clusterRepoRevision, "repo-revision", "", "the git revision for cluster template")
	command.Flags().StringVar(&clusterRepoPath, "repo-path", "", "the git repository path for cluster template")
	command.Flags().StringVar(&clusterName, "cluster-name", "", "the cluster name")
	command.Flags().BoolVar(&outputYaml, "output-yaml", false, "output root applications YAML instead of deploying to ArgoCD")
	command.MarkFlagRequired("repo-url")
	command.MarkFlagRequired("cluster-name")
	return command
}

