package clusterspec

import "github.com/spf13/cobra"

func NewCommand() *cobra.Command {
	command := &cobra.Command{
		Use:               "clusterspec",
		Short:             "Manage cluster specifications",
		Long:              "Manage cluster specifications",
		DisableAutoGenTag: true,
		Run: func(c *cobra.Command, args []string) {
			_ = c.Usage()
		},
	}
	command.AddCommand(listClusterspecsCommand())
	command.AddCommand(createClusterspecCommand())
	command.AddCommand(updateClusterspecCommand())
	command.AddCommand(deleteClusterspecCommand())
	return command
}
