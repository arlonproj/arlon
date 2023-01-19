package basecluster

import "github.com/spf13/cobra"

func NewCommand() *cobra.Command {
	command := &cobra.Command{
		Use:               "clustertemplate",
		Short:             "Manage cluster templates",
		Long:              "Manage cluster templates",
		Aliases:           []string{"basecluster", "clustertemplates", "baseclusters"},
		DisableAutoGenTag: true,
		Run: func(c *cobra.Command, args []string) {
			_ = c.Usage()
		},
	}
	command.AddCommand(validateBaseClusterCommand())
	command.AddCommand(validateGitBaseClusterCommand())
	command.AddCommand(prepareBaseClusterCommand())
	command.AddCommand(prepareGitBaseClusterCommand())
	return command
}
