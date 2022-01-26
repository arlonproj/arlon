package bundle

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
)

import "github.com/argoproj/argo-cd/v2/util/cli"

func createBundleCommand() *cobra.Command {
	var clientConfig clientcmd.ClientConfig
	var ns string
	var fromFile string
	var repoUrl string
	var repoPath string
	var desc string
	var tags string
	command := &cobra.Command{
		Use:               "create",
		Short:             "Create configuration bundle",
		Long:              "Create configuration bundle",
		Args: cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			config, err := clientConfig.ClientConfig()
			if err != nil {
				return fmt.Errorf("failed to get k8s client config: %s", err)
			}
			return createBundle(config, ns, args[0], fromFile, repoUrl, repoPath, desc, tags)
		},
	}
	clientConfig = cli.AddKubectlFlagsToCmd(command)
	command.Flags().StringVar(&ns, "ns", "arlon", "the arlon namespace")
	command.Flags().StringVar(&fromFile, "from-file", "", "create inline bundle from this file")
	command.Flags().StringVar(&repoUrl, "repo-url", "", "create a reference bundle from this repo URL")
	command.Flags().StringVar(&repoPath, "repo-path", "", "optional path in repo specified by --from-repo")
	command.Flags().StringVar(&desc, "desc", "", "description")
	command.Flags().StringVar(&tags, "tags", "", "comma separated list of tags")
	return command
}


func createBundle(config *restclient.Config, ns string, bundleName string, fromFile string, repoUrl string, repoPath string, desc string, tags string) error {
	kubeClient := kubernetes.NewForConfigOrDie(config)
	corev1 := kubeClient.CoreV1()
	secretsApi := corev1.Secrets(ns)
	_, err := secretsApi.Get(context.Background(), bundleName, metav1.GetOptions{})
	if err == nil {
		return fmt.Errorf("a bundle with that name already exists")
	}
	if !apierr.IsNotFound(err) {
		return fmt.Errorf("failed to check for existence of bundle: %s", err)
	}
	secr := v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: bundleName,
			Labels: map[string]string{
				"managed-by": "arlon",
				"arlon-type": "config-bundle",
			},
			Annotations: map[string]string{},
		},
		Data: map[string][]byte{
			"description": []byte(desc),
			"tags": []byte(tags),
		},
	}
	if fromFile != "" && repoUrl != "" {
		return fmt.Errorf("file and repo cannot both be specified")
	}
	if fromFile != "" {
		data, err := os.ReadFile(fromFile)
		if err != nil {
			return fmt.Errorf("failed to read file: %s", err)
		}
		secr.Labels["bundle-type"] = "inline"
		secr.Data["data"] = data
	} else if repoUrl != "" {
		secr.Labels["bundle-type"] = "reference"
		secr.ObjectMeta.Annotations["arlon.io/repo-url"] = repoUrl
		secr.ObjectMeta.Annotations["arlon.io/repo-path"] = repoPath
	} else {
		return fmt.Errorf("the bundle must be created from a file or repo URL")
	}
	_, err = secretsApi.Create(context.Background(), &secr, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create secret: %s", err)
	}
	return nil
}


