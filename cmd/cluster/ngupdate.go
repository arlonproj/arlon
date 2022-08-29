package cluster

import (
	"context"
	"fmt"

	argoapp "github.com/argoproj/argo-cd/v2/pkg/apiclient/application"
	"github.com/argoproj/argo-cd/v2/util/cli"
	"github.com/argoproj/argo-cd/v2/util/io"
	"github.com/arlonproj/arlon/pkg/argocd"
	"github.com/arlonproj/arlon/pkg/cluster"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
)

func ngupdateClusterCommand() *cobra.Command {
	var clientConfig clientcmd.ClientConfig
	var argocdNs string
	var arlonNs string
	var profileName string
	var deleteProfileName string
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
			apps, err := appIf.List(context.Background(),
				&argoapp.ApplicationQuery{Selector: "arlon-cluster=" + name + ",arlon-type=cluster-app"})
			if err != nil {
				return fmt.Errorf("failed to list apps related to cluster: %s", err)
			}
			if len(apps.Items) == 0 {
				return fmt.Errorf("Failed to get the given cluster")
			}
			if deleteProfileName == "" {
				_, err = cluster.NgUpdate(appIf, config, argocdNs, arlonNs, name, profileName, true)
				if err != nil {

					return fmt.Errorf("Error: %s", err)
				}
			} else {
				err = cluster.DestroyProfileApps(appIf, name)
				if err != nil {
					return fmt.Errorf("Failed to delete the profile app")
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
	return command
}
