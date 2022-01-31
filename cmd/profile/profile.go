package profile

import "github.com/spf13/cobra"

func NewCommand() *cobra.Command {
	command := &cobra.Command{
		Use:               "profile",
		Short:             "Manage configuration profiles",
		Long:              "Manage configuration profiles",
		DisableAutoGenTag: true,
		Run: func(c *cobra.Command, args []string) {
		},
	}
	command.AddCommand(listProfilesCommand())
	command.AddCommand(createProfileCommand())
	command.AddCommand(deleteProfileCommand())
	command.AddCommand(updateProfileCommand())
	return command
}

