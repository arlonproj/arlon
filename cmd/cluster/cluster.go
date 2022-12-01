package cluster

import (
	"errors"

	"github.com/spf13/cobra"
)

var (
	ErrArgocdToken = errors.New("Login to ArgoCD again")
)

func NewCommand() *cobra.Command {
	command := &cobra.Command{
		Use:               "cluster",
		Short:             "Manage clusters",
		Long:              "Manage clusters",
		DisableAutoGenTag: true,
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
