package profile

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
)

import "github.com/argoproj/argo-cd/v2/util/cli"

func createProfileCommand() *cobra.Command {
	var clientConfig clientcmd.ClientConfig
	var ns string
	var desc string
	var bundles string
	var tags string
	command := &cobra.Command{
		Use:               "create",
		Short:             "Create profile",
		Long:              "Create profile",
		Args: cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			config, err := clientConfig.ClientConfig()
			if err != nil {
				return fmt.Errorf("failed to get k8s client config: %s", err)
			}
			return createProfile(config, ns, args[0], bundles, desc, tags)
		},
	}
	clientConfig = cli.AddKubectlFlagsToCmd(command)
	command.Flags().StringVar(&ns, "ns", "arlon", "the arlon namespace")
	command.Flags().StringVar(&desc, "desc", "", "description")
	command.Flags().StringVar(&bundles, "bundles", "", "comma separated list of bundles")
	command.Flags().StringVar(&tags, "tags", "", "comma separated list of tags")
	command.MarkFlagRequired("bundles")
	return command
}


func createProfile(config *restclient.Config, ns string, profileName string, bundles string, desc string, tags string) error {
	kubeClient := kubernetes.NewForConfigOrDie(config)
	corev1 := kubeClient.CoreV1()
	configMapApi := corev1.ConfigMaps(ns)
	_, err := configMapApi.Get(context.Background(), profileName, metav1.GetOptions{})
	if err == nil {
		return fmt.Errorf("a profile with that name already exists")
	}
	if !apierr.IsNotFound(err) {
		return fmt.Errorf("failed to check for existence of profile: %s", err)
	}
	cm := v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: profileName,
			Labels: map[string]string{
				"managed-by": "arlon",
				"arlon-type": "profile",
				"profile-type": "configuration",
			},
		},
		Data: map[string]string{
			"description": desc,
			"bundles": bundles,
			"tags": tags,
		},
	}
	_, err = configMapApi.Create(context.Background(), &cm, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create profile: %s", err)
	}
	return nil
}


