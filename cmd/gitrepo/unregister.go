package gitrepo

import (
	"fmt"
	"github.com/spf13/cobra"
)

func unregister() *cobra.Command {
	var (
		repoUrl string
	)
	command := &cobra.Command{
		Use:   "unregister",
		Args:  cobra.ExactArgs(1),
		Short: "unregister a previously registered configuration",
		Long:  "unregister a previously registered configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			repoUrl = args[0]
			fmt.Println(repoUrl)
			return nil
		},
	}
	return command
}
