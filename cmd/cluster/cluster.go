package cluster

import "github.com/spf13/cobra"

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
	return command
}
