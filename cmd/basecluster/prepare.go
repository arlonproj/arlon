package basecluster

import (
	"fmt"
	bcl "github.com/arlonproj/arlon/pkg/basecluster"
	"github.com/spf13/cobra"
	// "k8s.io/client-go/tools/clientcmd"
)

func prepareBaseClusterCommand() *cobra.Command {
	var validateOnly bool
	var casMin int
	var casMax int
	command := &cobra.Command{
		Use:   "prepare <filename> [flags]",
		Short: "prepare base cluster",
		Long:  "prepare base cluster",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			fileName := args[0]
			clusterName, modifiedYaml, err := bcl.Prepare(fileName, validateOnly, casMax, casMin)
			if err != nil {
				return err
			}
			if validateOnly {
				fmt.Println("validation successful, cluster name:", clusterName)
			} else {
				fmt.Println("preparation successful, cluster name:", clusterName)
				if modifiedYaml == nil {
					fmt.Println("manifest is already compliant, no changes necessary")
				} else {
					fmt.Println("at least one namespace removed, modified YAML:")
					fmt.Println("---")
					fmt.Println(string(modifiedYaml))
				}
			}
			return nil
		},
	}
	command.Flags().IntVar(&casMin, "cas-min", 1, "set minimum number of nodes for capi-cluster autoscaler, for MachineDeployment based clusters")
	command.Flags().IntVar(&casMax, "cas-min", 9, "set maximum number of nodes for capi-cluster autoscaler, for MachineDeployment based clusters")
	command.Flags().BoolVar(&validateOnly, "validate-only", false, "validate only, don't modify")
	return command
}
