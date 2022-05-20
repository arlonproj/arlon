package bundle

import (
	"context"
	"fmt"
	"github.com/arlonproj/arlon/pkg/common"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"text/tabwriter"
)

import "github.com/argoproj/argo-cd/v2/util/cli"

func listBundlesCommand() *cobra.Command {
	var clientConfig clientcmd.ClientConfig
	var ns string
	command := &cobra.Command{
		Use:   "list",
		Short: "List configuration bundles",
		Long:  "List configuration bundles",
		RunE: func(c *cobra.Command, args []string) error {
			config, err := clientConfig.ClientConfig()
			if err != nil {
				return fmt.Errorf("failed to get k8s client config: %s", err)
			}
			return listBundles(config, ns)
		},
	}
	clientConfig = cli.AddKubectlFlagsToCmd(command)
	command.Flags().StringVar(&ns, "ns", "arlon", "the arlon namespace")
	return command
}

func listBundles(config *restclient.Config, ns string) error {
	kubeClient := kubernetes.NewForConfigOrDie(config)
	corev1 := kubeClient.CoreV1()
	secretsApi := corev1.Secrets(ns)
	opts := metav1.ListOptions{
		LabelSelector: "managed-by=arlon,arlon-type=config-bundle",
	}
	secrets, err := secretsApi.List(context.Background(), opts)
	if err != nil {
		return fmt.Errorf("failed to list secrets: %s", err)
	}
	if len(secrets.Items) == 0 {
		fmt.Println("no bundles found")
		return nil
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintf(w, "NAME\tTYPE\tTAGS\tREPO\tPATH\tREVISION\tSRCTYPE\tDESCRIPTION\n")
	for _, secret := range secrets.Items {
		bundleType := secret.Labels["bundle-type"]
		if bundleType == "" {
			bundleType = "(undefined)"
		}
		repoUrl := secret.Annotations[common.RepoUrlAnnotationKey]
		repoPath := secret.Annotations[common.RepoPathAnnotationKey]
		repoRevision := secret.Annotations[common.RepoRevisionAnnotationKey]
		srcType := secret.Annotations[common.SrcTypeAnnotationKey]
		if bundleType != "dynamic" {
			repoUrl = "(N/A)"
			repoPath = "(N/A)"
		}
		tags := string(secret.Data["tags"])
		desc := string(secret.Data["description"])
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n", secret.Name,
			bundleType, tags, repoUrl, repoPath, repoRevision, srcType, desc)
	}
	_ = w.Flush()
	return nil
}
