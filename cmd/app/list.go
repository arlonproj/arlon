package app

import (
	"fmt"
	"github.com/arlonproj/arlon/pkg/app"
	"github.com/arlonproj/arlon/pkg/appprofile"
	"github.com/spf13/cobra"
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
			return listToStdout(config, ns)
		},
	}
	clientConfig = cli.AddKubectlFlagsToCmd(command)
	command.Flags().StringVar(&ns, "ns", "argocd", "the argo-cd namespace")
	return command
}

func listToStdout(config *restclient.Config, ns string) error {
	apps, err := app.List(config, ns)
	if err != nil {
		return fmt.Errorf("failed to list apps: %s", err)
	}
	profiles, err := appprofile.List(config, "arlon")
	if err != nil {
		return fmt.Errorf("failed to list application profiles: %s", err)
	}
	appToProf := make(map[string][]string)
	for _, prof := range profiles {
		for _, appName := range prof.Spec.AppNames {
			appToProf[appName] = append(appToProf[appName], prof.Name)
		}
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintf(w, "NAME\tREPO\tPATH\tREVISION\tAPP_PROFILES\n")
	for _, app := range apps {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			app.Name,
			app.Spec.Template.Spec.Source.RepoURL,
			app.Spec.Template.Spec.Source.Path,
			app.Spec.Template.Spec.Source.TargetRevision,
			appToProf[app.Name],
		)
	}
	_ = w.Flush()
	return nil
}
