package clusterspec

import (
	"fmt"
	cspec "github.com/arlonproj/arlon/pkg/clusterspec"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

import "github.com/argoproj/argo-cd/v2/util/cli"

func updateClusterspecCommand() *cobra.Command {
	var clientConfig clientcmd.ClientConfig
	var arlonNs string
	var desc string
	var tags string
	var kubernetesVersion string
	var nodeType string
	var nodeCount int
	var masterNodeCount int
	var clusterAutoscalerEnabledPtr *bool
	var disableClusterAutoscaler bool
	var enableClusterAutoscaler bool
	var clusterAutoscalerMinNodes int
	var clusterAutoscalerMaxNodes int
	command := &cobra.Command{
		Use:   "update <clusterspec name> [flags]",
		Short: "Update clusterspec",
		Long:  "Update clusterspec",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			config, err := clientConfig.ClientConfig()
			if err != nil {
				return fmt.Errorf("failed to get k8s client config: %s", err)
			}
			kubeClient := kubernetes.NewForConfigOrDie(config)
			if enableClusterAutoscaler {
				if disableClusterAutoscaler {
					return fmt.Errorf("--disablecas and --enablecas cannot both be set")
				}
				clusterAutoscalerEnabledPtr = &enableClusterAutoscaler // true
			} else if disableClusterAutoscaler {
				clusterAutoscalerEnabledPtr = &enableClusterAutoscaler // false
			}
			changed, err := cspec.Update(kubeClient, arlonNs, args[0], kubernetesVersion,
				nodeType, nodeCount, masterNodeCount, clusterAutoscalerEnabledPtr,
				clusterAutoscalerMinNodes, clusterAutoscalerMaxNodes,
				desc, tags)
			if err != nil {
				return err
			}
			if !changed {
				fmt.Println("no changes were made")
			}
			return nil
		},
	}
	clientConfig = cli.AddKubectlFlagsToCmd(command)
	command.Flags().StringVar(&arlonNs, "arlon-ns", "arlon", "the arlon namespace")
	command.Flags().StringVar(&desc, "desc", "", "description")
	command.Flags().StringVar(&tags, "tags", "", "comma separated list of tags")
	command.Flags().StringVar(&kubernetesVersion, "kubeversion", "", "the kubernetes version")
	command.Flags().StringVar(&nodeType, "nodetype", "", "the cloud-specific node instance type")
	command.Flags().IntVar(&nodeCount, "nodecount", 0, "the number of nodes")
	command.Flags().IntVar(&masterNodeCount, "masternodecount", 0, "the number of master nodes")
	command.Flags().BoolVar(&disableClusterAutoscaler, "disablecas", false, "disable cluster autoscaler")
	command.Flags().BoolVar(&enableClusterAutoscaler, "enablecas", false, "enable cluster autoscaler")
	command.Flags().IntVar(&clusterAutoscalerMinNodes, "casmin", 1, "minimum number of nodes for cluster autoscaling")
	command.Flags().IntVar(&clusterAutoscalerMaxNodes, "casmax", 9, "maximum number of nodes for cluster autoscaling")
	return command
}
