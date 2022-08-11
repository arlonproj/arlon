package gitrepo

import (
	"fmt"
	"github.com/spf13/cobra"
)

func register() *cobra.Command {
	var (
		repoUrl  string
		userName string
		password string
	)
	command := &cobra.Command{
		Use:   "register",
		Short: "register a git repository configuration",
		Long:  "register a git repository configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println(repoUrl, password, userName)
			return nil
		},
	}
	command.Flags().StringVar(&repoUrl, "repo-url", "", "url of the repository to register")
	command.Flags().StringVar(&userName, "user", "", "username for the repository configuration")
	command.Flags().StringVar(&password, "password", "", "password of the user")
	command.MarkFlagRequired("repo-url")
	command.MarkFlagRequired("user")
	command.MarkFlagRequired("password")
	return command
}
