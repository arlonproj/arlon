package bundle

import (
	"bytes"
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"io"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
)

import "github.com/argoproj/argo-cd/v2/util/cli"

func dumpBundleCommand() *cobra.Command {
	var clientConfig clientcmd.ClientConfig
	var ns string
	command := &cobra.Command{
		Use:   "dump",
		Short: "Dump content of static configuration bundle",
		Long:  "Dump content of static configuration bundle",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			config, err := clientConfig.ClientConfig()
			if err != nil {
				return fmt.Errorf("failed to get k8s client config: %s", err)
			}
			return dumpBundle(config, ns, args[0])
		},
	}
	clientConfig = cli.AddKubectlFlagsToCmd(command)
	command.Flags().StringVar(&ns, "ns", "arlon", "the arlon namespace")
	return command
}

func dumpBundle(config *restclient.Config, ns string, bundleName string) error {
	kubeClient := kubernetes.NewForConfigOrDie(config)
	corev1 := kubeClient.CoreV1()
	secretsApi := corev1.Secrets(ns)
	secret, err := secretsApi.Get(context.Background(), bundleName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get bundle secret: %s", err)
	}
	if secret.Labels["arlon-type"] != "config-bundle" {
		return fmt.Errorf("secret is missing expected label")
	}
	if secret.Labels["bundle-type"] != "static" {
		return fmt.Errorf("bundle is not of static type")
	}
	if secret.Data["data"] == nil {
		return fmt.Errorf("bundle has no data")
	}
	_, err = io.Copy(os.Stdout, bytes.NewReader(secret.Data["data"]))
	if err != nil {
		return fmt.Errorf("failed to copy secret data: %s", err)
	}
	return nil
}
