package app

import "github.com/spf13/cobra"

func NewCommand() *cobra.Command {
	command := &cobra.Command{
		Use:               "app",
		Short:             "Manage apps",
		Long:              "Manage apps",
		DisableAutoGenTag: true,
		Aliases:           []string{"apps"},
		Run: func(c *cobra.Command, args []string) {
			_ = c.Usage()
		},
	}
	command.AddCommand(listAppsCommand())
	command.AddCommand(createAppCommand())
	command.AddCommand(deleteAppCommand())
	return command
}
