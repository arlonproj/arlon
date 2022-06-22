package basecluster

import "github.com/spf13/cobra"

func NewCommand() *cobra.Command {
	command := &cobra.Command{
		Use:               "basecluster",
		Short:             "Manage base clusters",
		Long:              "Manage base clusters",
		DisableAutoGenTag: true,
		Run: func(c *cobra.Command, args []string) {
			c.Usage()
		},
	}
	command.AddCommand(validateBaseClusterCommand())
	return command
}
