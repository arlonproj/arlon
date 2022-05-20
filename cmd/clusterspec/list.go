package clusterspec

import (
	"context"
	"fmt"
	"github.com/argoproj/argo-cd/v2/util/cli"
	"github.com/arlonproj/arlon/pkg/clusterspec"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"text/tabwriter"
)

func listClusterspecsCommand() *cobra.Command {
	var clientConfig clientcmd.ClientConfig
	var ns string
	command := &cobra.Command{
		Use:   "list",
		Short: "List configuration clusterspecs",
		Long:  "List configuration clusterspecs",
		RunE: func(c *cobra.Command, args []string) error {
			config, err := clientConfig.ClientConfig()
			if err != nil {
				return fmt.Errorf("failed to get k8s client config: %s", err)
			}
			return listClusterspecs(config, ns)
		},
	}
	clientConfig = cli.AddKubectlFlagsToCmd(command)
	command.Flags().StringVar(&ns, "ns", "arlon", "the arlon namespace")
	return command
}

func listClusterspecs(config *restclient.Config, ns string) error {
	kubeClient := kubernetes.NewForConfigOrDie(config)
	corev1 := kubeClient.CoreV1()
	configMapsApi := corev1.ConfigMaps(ns)
	opts := metav1.ListOptions{
		LabelSelector: "managed-by=arlon,arlon-type=clusterspec",
	}
	configMaps, err := configMapsApi.List(context.Background(), opts)
	if err != nil {
		return fmt.Errorf("failed to list configMaps: %s", err)
	}
	if len(configMaps.Items) == 0 {
		fmt.Println("no clusterspecs found")
		return nil
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintf(w, "NAME\tAPIPROV\tCLOUDPROV\tTYPE\tKUBEVERSION\tNODETYPE\tNODECNT\tMSTNODECNT\tSSHKEY\tCAS\tCASMIN\tCASMAX\tTAGS\tDESCRIPTION\n")
	for _, configMap := range configMaps.Items {
		cs, err := clusterspec.FromConfigMap(&configMap)
		if err != nil {
			fmt.Fprintf(w, "%s\t(corrupt data)\n", configMap.Name)
			continue
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%d\t%d\t%s\t%t\t%d\t%d\t%s\t%s\n",
			cs.Name, cs.ApiProvider, cs.CloudProvider,
			cs.Type, cs.KubernetesVersion, cs.NodeType, cs.NodeCount,
			cs.MasterNodeCount, cs.SshKeyName, cs.ClusterAutoscalerEnabled,
			cs.ClusterAutoscalerMinNodes, cs.ClusterAutoscalerMaxNodes,
			cs.Tags, cs.Description)
	}
	_ = w.Flush()
	return nil
}
