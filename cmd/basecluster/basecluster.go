package basecluster

import "github.com/spf13/cobra"

func NewCommand() *cobra.Command {
	command := &cobra.Command{
		Use:               "basecluster",
		Short:             "Manage cluster templates",
		Long:              "Manage cluster templates",
		Aliases:           []string{"clustertemplate", "clustertemplates", "baseclusters"},
		DisableAutoGenTag: true,
		Run: func(c *cobra.Command, args []string) {
			c.Usage()
		},
	}
	command.AddCommand(validateBaseClusterCommand())
	command.AddCommand(validateGitBaseClusterCommand())
	command.AddCommand(prepareBaseClusterCommand())
	command.AddCommand(prepareGitBaseClusterCommand())
	return command
}
