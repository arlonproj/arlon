package app

import (
	"fmt"
	"github.com/arlonproj/arlon/pkg/app"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"text/tabwriter"
)

import "github.com/argoproj/argo-cd/v2/util/cli"

func listAppsCommand() *cobra.Command {
	var clientConfig clientcmd.ClientConfig
	var ns string
	command := &cobra.Command{
		Use:   "list",
		Short: "List apps",
		Long:  "List apps",
		RunE: func(c *cobra.Command, args []string) error {
			config, err := clientConfig.ClientConfig()
			if err != nil {
				return fmt.Errorf("failed to get k8s client config: %s", err)
			}
			return listApps(config, ns)
		},
	}
	clientConfig = cli.AddKubectlFlagsToCmd(command)
	command.Flags().StringVar(&ns, "ns", "argocd", "the argo-cd namespace")
	return command
}

func listApps(config *restclient.Config, ns string) error {
	apps, err := app.List(config, ns)
	if err != nil {
		return fmt.Errorf("failed to list apps: %s", err)
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintf(w, "NAME\tREPO\tPATH\tREVISION\tPROFILES\n")
	for _, app := range apps {
		profiles := "(none)"
		if len(app.Spec.Generators) == 1 &&
			app.Spec.Generators[0].Clusters != nil &&
			len(app.Spec.Generators[0].Clusters.Selector.MatchExpressions) >= 1 {
			if app.Spec.Generators[0].Clusters.Selector.MatchExpressions[0].Key != "arlon.io/profile" ||
				app.Spec.Generators[0].Clusters.Selector.MatchExpressions[0].Operator != metav1.LabelSelectorOpIn {
				profiles = "(invalid data)"
			} else {
				profiles = fmt.Sprintf("%s", app.Spec.Generators[0].Clusters.Selector.MatchExpressions[0].Values)
			}
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			app.Name,
			app.Spec.Template.Spec.Source.RepoURL,
			app.Spec.Template.Spec.Source.Path,
			app.Spec.Template.Spec.Source.TargetRevision,
			profiles,
		)
	}
	_ = w.Flush()
	return nil
}
