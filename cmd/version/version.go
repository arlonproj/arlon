package version

import (
	"fmt"
	"github.com/spf13/cobra"
)

var cliVersion string

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "check for arlon CLI version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("CLI Version: %s\n", cliVersion)
		},
	}
	return cmd
}
