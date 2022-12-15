package cluster

import (
	_ "embed"
	"fmt"
	"os"

	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/argoproj/argo-cd/v2/util/cli"
	arlonv1 "github.com/arlonproj/arlon/api/v1"
	"github.com/arlonproj/arlon/pkg/argocd"
	bcl "github.com/arlonproj/arlon/pkg/basecluster"
	"github.com/arlonproj/arlon/pkg/cluster"
	"github.com/arlonproj/arlon/pkg/gitrepo"
	"github.com/arlonproj/arlon/pkg/profile"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/tools/clientcmd"
)

func createClusterCommand() *cobra.Command {
	var clientConfig clientcmd.ClientConfig
	var argocdNs string
	var arlonNs string
	var arlonRepoUrl string
	var arlonRepoRevision string
	var arlonRepoPath string
	var clusterRepoUrl string
	var repoAlias string
	var clusterRepoRevision string
	var clusterRepoPath string
	var clusterName string
	var outputYaml bool
	var profileName string
	var gen2CASEnabled bool //gen2 specific flag to enable cluster autoscaler
	command := &cobra.Command{
		Use:   "create",
		Short: "create new cluster from a base",
		Long:  "create new cluster from a base",
		RunE: func(c *cobra.Command, args []string) error {
			if clusterRepoUrl == "" {
				var err error
				clusterRepoUrl, err = gitrepo.GetRepoUrl(repoAlias)
				if err != nil {
					return err
				}
			}
			conn, appIf := argocd.NewArgocdClientOrDie("").NewApplicationClientOrDie()
			defer conn.Close()
			config, err := clientConfig.ClientConfig()
			if err != nil {
				return fmt.Errorf("failed to get k8s client config: %s", err)
			}
			_, creds, err := argocd.GetKubeclientAndRepoCreds(config, argocdNs, clusterRepoUrl)
			if err != nil {
				return fmt.Errorf("failed to get repository credentials: %s", err)
			}
			createInArgoCd := !outputYaml
			baseClusterName, err := bcl.ValidateGitDir(creds,
				clusterRepoUrl, clusterRepoRevision, clusterRepoPath)
			if err != nil {
				return fmt.Errorf("failed to validate base cluster: %s", err)
			}
			var prof *arlonv1.Profile
			if profileName != "" {
				prof, err = profile.Get(config, profileName, arlonNs)
				if err != nil {
					return fmt.Errorf("failed to get profile: %s", err)
				}
				if prof.Spec.RepoUrl == "" {
					return fmt.Errorf("profile %s is not dynamic",
						profileName)
				}
			}
			// Create "arlon app" for cluster
			arlonApp, err := cluster.Create(appIf, config, argocdNs, arlonNs,
				clusterName, baseClusterName, arlonRepoUrl, arlonRepoRevision,
				arlonRepoPath, "",
				nil, createInArgoCd, config.Host, gen2CASEnabled)
			if err != nil {
				return fmt.Errorf("failed to create arlon app: %s", err)
			}
			// Create "cluster app" for cluster
			clusterApp, err := cluster.CreateClusterApp(appIf, argocdNs,
				clusterName, baseClusterName, clusterRepoUrl, clusterRepoRevision,
				clusterRepoPath, createInArgoCd)
			if err != nil {
				return fmt.Errorf("failed to create cluster app: %s", err)
			}
			// Create "profile app" for cluster if necessary
			var profileApp *v1alpha1.Application
			if prof != nil {
				profileAppName := fmt.Sprintf("%s-profile-%s", clusterName, prof.Name)
				profileApp, err = cluster.CreateProfileApp(profileAppName,
					appIf, argocdNs, clusterName, prof, createInArgoCd)
				if err != nil {
					return fmt.Errorf("failed to create profile app: %s", err)
				}
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
					return fmt.Errorf("failed to encode arlon app: %s", err)
				}
				fmt.Println("---")
				err = s.Encode(clusterApp, os.Stdout)
				if err != nil {
					return fmt.Errorf("failed to encode cluster app: %s", err)
				}
				if profileApp != nil {
					fmt.Println("---")
					err = s.Encode(profileApp, os.Stdout)
					if err != nil {
						return fmt.Errorf("failed to encode profile app: %s", err)
					}
				}
			}
			return nil
		},
	}
	clientConfig = cli.AddKubectlFlagsToCmd(command)
	command.Flags().StringVar(&argocdNs, "argocd-ns", "argocd", "the argocd namespace")
	command.Flags().StringVar(&arlonNs, "arlon-ns", "arlon", "the arlon namespace")
	command.Flags().StringVar(&arlonRepoUrl, "arlon-repo-url", "https://github.com/arlonproj/arlon.git", "the git repository url for arlon template")
	command.Flags().StringVar(&arlonRepoRevision, "arlon-repo-revision", "private/Rohitrajak1807/chc-debug", "the git revision for arlon template")
	command.Flags().StringVar(&arlonRepoPath, "arlon-repo-path", "pkg/cluster/manifests", "the git repository path for arlon template")
	command.Flags().StringVar(&clusterRepoUrl, "repo-url", "", "the git repository url for cluster template")
	command.Flags().StringVar(&repoAlias, "repo-alias", gitrepo.RepoDefaultCtx, "git repository alias to use")
	command.Flags().StringVar(&clusterRepoRevision, "repo-revision", "main", "the git revision for cluster template")
	command.Flags().StringVar(&clusterRepoPath, "repo-path", "", "the git repository path for cluster template")
	command.Flags().StringVar(&clusterName, "cluster-name", "", "the cluster name")
	command.Flags().BoolVar(&outputYaml, "output-yaml", false, "output root applications YAML instead of deploying to ArgoCD")
	command.Flags().StringVar(&profileName, "profile", "", "profile name (if specified, must refer to dynamic profile)")
	command.Flags().BoolVar(&gen2CASEnabled, "autoscaler", false, "enable CAPI cluster autoscaler for cluster template based clusters")
	command.MarkFlagRequired("cluster-name")
	command.MarkFlagsMutuallyExclusive("repo-url", "repo-alias")
	return command
}
