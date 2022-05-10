package cluster

import (
	"context"
	_ "embed"
	"fmt"
	argoapp "github.com/argoproj/argo-cd/v2/pkg/apiclient/application"
	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/argoproj/argo-cd/v2/util/cli"
	"github.com/arlonproj/arlon/pkg/argocd"
	"github.com/arlonproj/arlon/pkg/cluster"
	"github.com/arlonproj/arlon/pkg/common"
	"github.com/arlonproj/arlon/pkg/profile"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/tools/clientcmd"
	"os"
)

func updateClusterCommand() *cobra.Command {
	var clientConfig clientcmd.ClientConfig
	var argocdNs string
	var arlonNs string
	var clusterSpecName string
	var profileName string
	var outputYaml bool
	command := &cobra.Command{
		Use:   "update <clustername> [flags]",
		Short: "update existing cluster",
		Long:  "update existing cluster",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			conn, appIf := argocd.NewArgocdClientOrDie("").NewApplicationClientOrDie()
			defer conn.Close()
			config, err := clientConfig.ClientConfig()
			if err != nil {
				return fmt.Errorf("failed to get k8s client config: %s", err)
			}
			updateInArgoCd := !outputYaml
			clusterName := args[0]
			oldApp, err := appIf.Get(context.Background(),
				&argoapp.ApplicationQuery{Name: &clusterName})
			if err != nil {
				return fmt.Errorf("failed to get argocd app: %s", err)
			}
			if clusterSpecName == "" {
				clusterSpecName = oldApp.Annotations[common.ClusterSpecAnnotationKey]
				if clusterSpecName == "" {
					return fmt.Errorf("existing cluster root app is missing clusterspec annotation")
				}
			}
			if profileName == "" {
				profileName = oldApp.Annotations[common.ProfileAnnotationKey]
				if profileName == "" {
					return fmt.Errorf("existing cluster root app is missing profile annotation")
				}
			}
			prof, err := profile.Get(config, profileName, arlonNs)
			if err != nil {
				return fmt.Errorf("failed to get profile: %s", err)
			}
			rootApp, err := cluster.Update(appIf, config, argocdNs, arlonNs,
				clusterName, clusterSpecName, prof, updateInArgoCd,
				config.Host, oldApp)
			if err != nil {
				return fmt.Errorf("failed to update cluster: %s", err)
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
	command.Flags().StringVar(&profileName, "profile", "", "the configuration profile to use")
	command.Flags().StringVar(&clusterSpecName, "cluster-spec", "", "the clusterspec to use")
	command.Flags().BoolVar(&outputYaml, "output-yaml", false, "output root application YAML instead of updating ArgoCD root app")
	command.MarkFlagRequired("cluster-name")
	return command
}
