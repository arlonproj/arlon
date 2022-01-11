package cluster

import (
	"context"
	"fmt"
	"github.com/argoproj/argo-cd/v2/util/cli"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"strings"
)

func deployClusterCommand() *cobra.Command {
	var clientConfig clientcmd.ClientConfig
	var ns string
	var repoUrl string
	var basePath string
	command := &cobra.Command{
		Use:               "deploy",
		Short:             "Deploy cluster",
		Long:              "Deploy cluster",
		RunE: func(c *cobra.Command, args []string) error {
			config, err := clientConfig.ClientConfig()
			if err != nil {
				return fmt.Errorf("failed to get k8s client config: %s", err)
			}
			return deployCluster(config, ns, repoUrl, basePath)
		},
	}
	clientConfig = cli.AddKubectlFlagsToCmd(command)
	command.Flags().StringVar(&ns, "ns", "argocd", "the argocd namespace")
	command.Flags().StringVar(&repoUrl, "repo-url", "", "the git repository url")
	command.Flags().StringVar(&basePath, "path", "arlon", "the git repository base path")
	command.MarkFlagRequired("repo-url")
	return command
}

type RepoCreds struct {
	Url string
	Username string
	Password string
}

func deployCluster(config *restclient.Config, ns string, repoUrl string, basePath string) error {
	kubeClient := kubernetes.NewForConfigOrDie(config)
	corev1 := kubeClient.CoreV1()
	secretsApi := corev1.Secrets(ns)
	opts := metav1.ListOptions{
		LabelSelector: "argocd.argoproj.io/secret-type=repository",
	}
	secrets, err := secretsApi.List(context.Background(), opts)
	if err != nil {
		return fmt.Errorf("failed to list secrets: %s", err)
	}
	var creds *RepoCreds
	for _, repoSecret := range secrets.Items {
		if strings.Compare(repoUrl, string(repoSecret.Data["url"])) == 0 {
			creds = &RepoCreds{
				Url: string(repoSecret.Data["url"]),
				Username: string(repoSecret.Data["username"]),
				Password: string(repoSecret.Data["password"]),
			}
			break
		}
	}
	if creds == nil {
		return fmt.Errorf("did not find secret matching repo url: %s", repoUrl)
	}
	fmt.Println("repo username:", creds.Username)
	fmt.Println("repo password:", creds.Password)
	return nil
}

