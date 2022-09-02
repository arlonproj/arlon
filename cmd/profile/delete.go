package profile

import (
	"context"
	"fmt"
	"github.com/argoproj/argo-cd/v2/util/errors"
	v1 "github.com/arlonproj/arlon/api/v1"
	"github.com/arlonproj/arlon/pkg/controller"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/types"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

import "github.com/argoproj/argo-cd/v2/util/cli"

func deleteProfileCommand() *cobra.Command {
	var clientConfig clientcmd.ClientConfig
	var ns string
	command := &cobra.Command{
		Use:   "delete",
		Short: "Delete profile",
		Long:  "Delete profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			config, err := clientConfig.ClientConfig()
			if err != nil {
				return fmt.Errorf("failed to get k8s client config: %s", err)
			}
			return deleteProfile(config, ns, args[0])
		},
	}
	clientConfig = cli.AddKubectlFlagsToCmd(command)
	command.Flags().StringVar(&ns, "ns", "arlon", "the arlon namespace")
	return command
}

func deleteProfile(config *restclient.Config, ns string, profileName string) error {
	ctrl, err := controller.NewClient(config)
	errors.CheckError(err)
	ctx := context.Background()
	var prof v1.Profile
	err = ctrl.Get(ctx, types.NamespacedName{
		Namespace: ns,
		Name:      profileName,
	}, &prof)
	errors.CheckError(err)
	return ctrl. /*ALT*/ Delete(ctx, &prof, nil)
	//kubeClient := kubernetes.NewForConfigOrDie(config)
	//return profile.Delete(kubeClient, ns, profileName)
}
