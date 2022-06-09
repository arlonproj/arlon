package profile

import (
	"fmt"
	"github.com/arlonproj/arlon/pkg/profile"
	"github.com/spf13/cobra"
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
	var overrides []string
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
			var bundlesPtr *string
			if clear {
				if bundles != "" {
					return fmt.Errorf("bundles must not be specified when using --clear")
				}
				bundlesPtr = &bundles // change profile to empty bundle set
			} else if bundles != "" {
				bundlesPtr = &bundles //
			} // bundles == "", meaning no change, so leave bundlesPtr as nil

			o, err := processOverrides(overrides)
			if err != nil {
				return fmt.Errorf("failed to process overrides: %s", err)
			}
			modified, err := profile.Update(config, argocdNs, arlonNs, args[0],
				bundlesPtr, desc, tags, o)
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
	command.Flags().StringArrayVarP(&overrides, "param", "p", nil, "add a single parameter override of the form bundle,key,value ... can be repeated")
	return command
}
