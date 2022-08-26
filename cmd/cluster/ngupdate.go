package cluster

import (
	"fmt"
	"os"

	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/argoproj/argo-cd/v2/util/cli"
	"github.com/argoproj/argo-cd/v2/util/io"
	"github.com/arlonproj/arlon/pkg/argocd"
	"github.com/arlonproj/arlon/pkg/cluster"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/tools/clientcmd"
)

func ngupdateClusterCommand() *cobra.Command {
	var clientConfig clientcmd.ClientConfig
	var argocdNs string
	var arlonNs string
	var profileName string
	var deleteProfileName string
	var outputYaml bool
	command := &cobra.Command{
		Use:   "ngupdate <clustername> [flags]",
		Short: "update existing next-gen cluster",
		Long:  "update existing next-gen cluster",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			name := args[0]
			conn, appIf := argocd.NewArgocdClientOrDie("").NewApplicationClientOrDie()
			defer io.Close(conn)
			config, err := clientConfig.ClientConfig()
			if err != nil {
				return fmt.Errorf("failed to get k8s client config: %s", err)
			}
			found := false
			clist, err := cluster.List(appIf, config, argocdNs)
			for _, c := range clist {
				if c.BaseCluster != nil {
					if name == c.BaseCluster.Name {
						found = true
						break
					}
				}
			}
			if !found {
				return fmt.Errorf("Failed to get the given cluster")
			}
			//case where the user either wants to switch or add a new profile to existing base cluster
			var profileApp *v1alpha1.Application
			if deleteProfileName == "" {
				profileApp, err = cluster.NgUpdate(appIf, config, argocdNs, arlonNs, name, profileName, !outputYaml)
				if err != nil {
					return fmt.Errorf("Failed to update the profile app")
				}
			} else {
				err = cluster.DestructProfileApp(appIf, name)
				if err != nil {
					return fmt.Errorf("Failed to delete the profile app")
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
	command.Flags().StringVar(&profileName, "profile", "", "the configuration profile to use")
	command.Flags().StringVar(&deleteProfileName, "delete-profile", "", "the configuration profile to be deleted")
	command.Flags().BoolVar(&outputYaml, "output-yaml", false, "output root application YAML instead of updating ArgoCD root app")
	return command
}
