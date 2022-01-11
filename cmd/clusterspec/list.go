package clusterspec

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"text/tabwriter"
)

import "github.com/argoproj/argo-cd/v2/util/cli"

func listClusterspecsCommand() *cobra.Command {
	var clientConfig clientcmd.ClientConfig
	var ns string
	command := &cobra.Command{
		Use:               "list",
		Short:             "List configuration clusterspecs",
		Long:              "List configuration clusterspecs",
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
	_, _ = fmt.Fprintf(w, "NAME\tTYPE\tKUBEVERSION\tNODETYPE\tNODECOUNT\tTAGS\tDESCRIPTION\n")
	for _, configMap := range configMaps.Items {
		clusterType := configMap.Data["type"]
		kubernetesVersion := configMap.Data["kubernetesVersion"]
		nodeType := configMap.Data["nodeType"]
		nodeCount := configMap.Data["nodeCount"]
		tags := configMap.Data["tags"]
		desc := string(configMap.Data["description"])
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n", configMap.Name,
			clusterType, kubernetesVersion, nodeType, nodeCount, tags, desc)
	}
	_ = w.Flush()
	return nil
}

