package app

import "github.com/spf13/cobra"

func NewCommand() *cobra.Command {
	command := &cobra.Command{
		Use:               "app",
		Short:             "Manage apps",
		Long:              "Manage apps",
		DisableAutoGenTag: true,
		Run: func(c *cobra.Command, args []string) {
			c.Usage()
		},
	}
	command.AddCommand(listAppsCommand())
	command.AddCommand(addToProfileCommand())
	command.AddCommand(removeFromProfileCommand())
	return command
}
