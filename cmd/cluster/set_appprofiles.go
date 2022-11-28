package cluster

import (
	_ "embed"
	"fmt"
	"github.com/arlonproj/arlon/pkg/argocd"
	"github.com/arlonproj/arlon/pkg/cluster"
	"github.com/spf13/cobra"
)

func setAppProfilesCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "setappprofiles <clustername> <comma_separated_app_profiles>",
		Short: "set a cluster's list of application profiles ",
		Long:  "set a cluster's list of application profiles ",
		Args:  cobra.ExactArgs(2),
		RunE: func(c *cobra.Command, args []string) error {
			argoIf := argocd.NewArgocdClientOrDie("")
			conn, appIf := argoIf.NewApplicationClientOrDie()
			defer conn.Close()
			clusterName := args[0]
			commaSeparatedAppProfiles := args[1]
			err := cluster.SetAppProfiles(appIf, clusterName, commaSeparatedAppProfiles)
			if err != nil {
				return fmt.Errorf("failed to set cluster's app profiles list: %s", err)
			}
			return nil
		},
	}
	return command
}
