package ngprofile

import "github.com/spf13/cobra"

func NewCommand() *cobra.Command {
	command := &cobra.Command{
		Use:               "ngprofile",
		Short:             "Manage next-generation profiles",
		Long:              "Manage next-generation profiles",
		DisableAutoGenTag: true,
		Run: func(c *cobra.Command, args []string) {
			c.Usage()
		},
	}
	command.AddCommand(listProfilesCommand())
	command.AddCommand(attachCommand())
	return command
}
