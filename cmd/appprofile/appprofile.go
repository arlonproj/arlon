package appprofile

import "github.com/spf13/cobra"

func NewCommand() *cobra.Command {
	command := &cobra.Command{
		Use:               "appprofile",
		Short:             "Manage application profiles",
		Long:              "Manage application profiles",
		DisableAutoGenTag: true,
		Aliases:           []string{"appprofiles"},
		Run: func(c *cobra.Command, args []string) {
			_ = c.Usage()
		},
	}
	command.AddCommand(listAppProfilesCommand())
	return command
}
