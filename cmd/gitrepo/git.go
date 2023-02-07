package gitrepo

import "github.com/spf13/cobra"

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "git",
		Short: "manage git configurations",
		Long:  "manage git configurations",
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Usage()
		},
	}
	cmd.AddCommand(register(), unregister())
	return cmd
}
