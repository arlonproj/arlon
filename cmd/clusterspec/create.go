package clusterspec

import (
	"fmt"
	cspec "github.com/arlonproj/arlon/pkg/clusterspec"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

import "github.com/argoproj/argo-cd/v2/util/cli"

func createClusterspecCommand() *cobra.Command {
	var clientConfig clientcmd.ClientConfig
	var arlonNs string
	var desc string
	var tags string
	var apiProvider string
	var cloudProvider string
	var clusterType string
	var kubernetesVersion string
	var nodeType string
	var nodeCount int
	var masterNodeCount int
	var sshKeyName string
	var clusterAutoscalerEnabled bool
	var clusterAutoscalerMinNodes int
	var clusterAutoscalerMaxNodes int
	command := &cobra.Command{
		Use:   "create",
		Short: "Create clusterspec",
		Long:  "Create clusterspec",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			config, err := clientConfig.ClientConfig()
			if err != nil {
				return fmt.Errorf("failed to get k8s client config: %s", err)
			}
			kubeClient := kubernetes.NewForConfigOrDie(config)
			return cspec.Create(kubeClient, arlonNs, args[0], apiProvider,
				cloudProvider, clusterType, kubernetesVersion,
				nodeType, nodeCount, masterNodeCount, sshKeyName,
				clusterAutoscalerEnabled,
				clusterAutoscalerMinNodes, clusterAutoscalerMaxNodes,
				desc, tags)
		},
	}
	clientConfig = cli.AddKubectlFlagsToCmd(command)
	command.Flags().StringVar(&arlonNs, "arlon-ns", "arlon", "the arlon namespace")
	command.Flags().StringVar(&desc, "desc", "", "description")
	command.Flags().StringVar(&apiProvider, "api", "capi", "the API provider (capi or xplane)")
	command.Flags().StringVar(&tags, "tags", "", "comma separated list of tags")
	command.Flags().StringVar(&cloudProvider, "cloud", "aws", "the cloud provider")
	command.Flags().StringVar(&clusterType, "type", "eks", "the cluster type (kubeadm or eks/aks)")
	command.Flags().StringVar(&kubernetesVersion, "kubeversion", "v1.18.16", "the kubernetes version")
	command.Flags().StringVar(&nodeType, "nodetype", "t3.large", "the cloud-specific node instance type")
	command.Flags().StringVar(&sshKeyName, "sshkey", "", "ssh key name for logging into nodes")
	command.Flags().IntVar(&nodeCount, "nodecount", 2, "the number of nodes")
	command.Flags().IntVar(&masterNodeCount, "masternodecount", 3, "the number of master nodes (3 or more required for HA)")
	command.Flags().BoolVar(&clusterAutoscalerEnabled, "casenabled", false, "enable cluster autoscaler")
	command.Flags().IntVar(&clusterAutoscalerMinNodes, "casmin", 1, "minimum number of nodes for cluster autoscaling")
	command.Flags().IntVar(&clusterAutoscalerMaxNodes, "casmax", 9, "maximum number of nodes for cluster autoscaling")
	command.MarkFlagRequired("sshkey")
	return command
}
