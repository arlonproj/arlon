package basecluster

import (
	"fmt"
	bcl "github.com/arlonproj/arlon/pkg/basecluster"
	"github.com/spf13/cobra"
	// "k8s.io/client-go/tools/clientcmd"
)

func prepareBaseClusterCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "prepare <filename> [flags]",
		Short: "prepare base cluster",
		Long:  "prepare base cluster",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			fileName := args[0]
			clusterName, err := bcl.Prepare(fileName)
			if err != nil {
				return err
			}
			fmt.Println("preparation successful, cluster name:", clusterName)
			return nil
		},
	}
	return command
}
