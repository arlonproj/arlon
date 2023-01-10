package app

import (
	"context"
	"fmt"
	"os"

	//appset "github.com/argoproj/argo-cd/v2/pkg/apis/applicationset/v1alpha1"
	appset "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/arlonproj/arlon/pkg/app"
	"github.com/arlonproj/arlon/pkg/ctrlruntimeclient"

	"github.com/argoproj/argo-cd/v2/util/cli"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/tools/clientcmd"
)

func createAppCommand() *cobra.Command {
	var clientConfig clientcmd.ClientConfig
	var argocdNs string
	var repoUrl string
	var repoRevision string
	var repoPath string
	var appName string
	var destNs string
	var project string
	var outputYaml bool
	var autoSync bool
	var autoPrune bool
	command := &cobra.Command{
		Use:   "create appName repoUrl repoPath [--repo-revision revision][--output-yaml][--autosync][--autoprune][other flags]",
		Short: "create new Arlon app",
		Long:  "create new Arlon app, which is represented as a specialized ArgoCD ApplicationSet resource",
		Args:  cobra.ExactArgs(3),
		RunE: func(c *cobra.Command, args []string) error {
			appName = args[0]
			repoUrl = args[1]
			repoPath = args[2]
			app := app.Create(argocdNs, appName, destNs, project, repoPath, repoUrl, repoRevision, autoSync, autoPrune)
			if outputYaml {
				scheme := runtime.NewScheme()
				if err := appset.AddToScheme(scheme); err != nil {
					return fmt.Errorf("failed to add scheme: %s", err)
				}
				s := json.NewSerializerWithOptions(json.DefaultMetaFactory,
					scheme, scheme, json.SerializerOptions{
						Yaml:   true,
						Pretty: true,
						Strict: false,
					})
				err := s.Encode(&app, os.Stdout)
				if err != nil {
					return fmt.Errorf("failed to encode arlon app: %s", err)
				}
			} else {
				config, err := clientConfig.ClientConfig()
				if err != nil {
					return fmt.Errorf("failed to get k8s client config: %s", err)
				}
				cli, err := ctrlruntimeclient.NewClient(config)
				if err != nil {
					return fmt.Errorf("failed to get controller runtime client: %s", err)
				}
				err = cli.Create(context.Background(), &app)
				if err != nil {
					return fmt.Errorf("failed to create resource: %s", err)
				}
			}
			return nil
		},
	}
	clientConfig = cli.AddKubectlFlagsToCmd(command)
	command.Flags().StringVar(&argocdNs, "argocd-ns", "argocd", "the argocd namespace")
	command.Flags().StringVar(&repoRevision, "repo-revision", "HEAD", "the git revision for app template")
	command.Flags().StringVar(&destNs, "dest-ns", "default", "destination namespace in target cluster(s)")
	command.Flags().StringVar(&project, "project", "default", "ArgoCD project for ApplicationSet representing Arlon app")
	command.Flags().BoolVar(&outputYaml, "output-yaml", false, "output YAML instead of deploying to management cluster")
	command.Flags().BoolVar(&autoSync, "autosync", true, "enable ArgoCD auto-sync")
	command.Flags().BoolVar(&autoPrune, "autoprune", true, "enable ArgoCD auto-prune, only meaningful if auto-sync enabled")
	return command
}
