package profile

import (
	"fmt"
	"github.com/platform9/arlon/pkg/profile"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

import "github.com/argoproj/argo-cd/v2/util/cli"

func updateProfileCommand() *cobra.Command {
	var clientConfig clientcmd.ClientConfig
	var arlonNs string
	var argocdNs string
	var desc string
	var bundles string
	var tags string
	var clear bool
	command := &cobra.Command{
		Use:   "update",
		Short: "Update profile",
		Long:  "Update profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			config, err := clientConfig.ClientConfig()
			if err != nil {
				return fmt.Errorf("failed to get k8s client config: %s", err)
			}
			kubeClient := kubernetes.NewForConfigOrDie(config)
			bundlesPtr := &bundles
			if clear {
				if bundles != "" {
					return fmt.Errorf("bundles must not be specified when using --clear")
				}
				bundlesPtr = nil // change profile to empty bundle set
			} else if bundles == "" {
				return fmt.Errorf("bundles must be specified unless using --clear")
			}
			modified, err := profile.Update(kubeClient, argocdNs, arlonNs, args[0],
				bundlesPtr, desc, tags)
			if err != nil {
				return err
			}
			if !modified {
				fmt.Println("profile not modified")
			}
			return nil
		},
	}
	clientConfig = cli.AddKubectlFlagsToCmd(command)
	command.Flags().StringVar(&arlonNs, "arlon-ns", "arlon", "the arlon namespace")
	command.Flags().StringVar(&argocdNs, "argocd-ns", "argocd", "the ArgoCD namespace")
	command.Flags().StringVar(&desc, "desc", "", "description")
	command.Flags().StringVar(&bundles, "bundles", "", "comma separated list of bundles")
	command.Flags().StringVar(&tags, "tags", "", "comma separated list of tags")
	command.Flags().BoolVar(&clear, "clear", false, "set the bundle list to the empty set")
	return command
}
