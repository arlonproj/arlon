package webhook

import (
	"flag"
	"github.com/arlonproj/arlon/pkg/webhook"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	port := 9443
	command := &cobra.Command{
		Use:               "webhook",
		Short:             "Run the Arlon webhook",
		Long:              "Run the Arlon webhook",
		DisableAutoGenTag: true,
		RunE: func(c *cobra.Command, args []string) error {
			return webhook.Start(port)
		},
	}
	flag.IntVar(&port, "port", port, "webhook listening port")
	return command
}
