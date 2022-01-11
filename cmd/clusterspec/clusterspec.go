package clusterspec

import "github.com/spf13/cobra"

func NewCommand() *cobra.Command {
	command := &cobra.Command{
		Use:               "clusterspec",
		Short:             "Manage cluster specifications",
		Long:              "Manage cluster specifications",
		DisableAutoGenTag: true,
		Run: func(c *cobra.Command, args []string) {
		},
	}
	command.AddCommand(listClusterspecsCommand())
	return command
}

