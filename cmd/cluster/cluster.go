package cluster

import (
	"context"
	"fmt"
	"os"

	apppkg "github.com/argoproj/argo-cd/v2/pkg/apiclient/application"
	"github.com/argoproj/argo-cd/v2/util/io"
	"github.com/arlonproj/arlon/pkg/argocd"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	command := &cobra.Command{
		Use:               "cluster",
		Short:             "Manage clusters",
		Long:              "Manage clusters",
		DisableAutoGenTag: true,
		PersistentPreRun:  checkForArgocd,
		Run: func(c *cobra.Command, args []string) {
			c.Usage()
		},
	}
	command.AddCommand(deployClusterCommand())
	command.AddCommand(listClustersCommand())
	command.AddCommand(updateClusterCommand())
	command.AddCommand(manageClusterCommand())
	command.AddCommand(unmanageClusterCommand())
	command.AddCommand(createClusterCommand())
	command.AddCommand(getClusterCommand())
	command.AddCommand(deleteClusterCommand())
	command.AddCommand(ngupdateClusterCommand())
	return command
}

func checkForArgocd(c *cobra.Command, args []string) {
	conn, appIf := argocd.NewArgocdClientOrDie("").NewApplicationClientOrDie()
	defer io.Close(conn)
	_, err := appIf.List(context.Background(), &apppkg.ApplicationQuery{})
	if err != nil {
		fmt.Println("ArgoCD auth token has expired....Login to ArgoCD again")
		fmt.Println(err)
		os.Exit(1)
	}
}
