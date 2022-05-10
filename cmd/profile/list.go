package profile

import (
	"fmt"
	"github.com/arlonproj/arlon/pkg/profile"
	"github.com/spf13/cobra"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"text/tabwriter"
)

import "github.com/argoproj/argo-cd/v2/util/cli"

func listProfilesCommand() *cobra.Command {
	var clientConfig clientcmd.ClientConfig
	var ns string
	command := &cobra.Command{
		Use:   "list",
		Short: "List configuration profiles",
		Long:  "List configuration profiles",
		RunE: func(c *cobra.Command, args []string) error {
			config, err := clientConfig.ClientConfig()
			if err != nil {
				return fmt.Errorf("failed to get k8s client config: %s", err)
			}
			return listProfiles(config, ns)
		},
	}
	clientConfig = cli.AddKubectlFlagsToCmd(command)
	command.Flags().StringVar(&ns, "ns", "arlon", "the arlon namespace")
	return command
}

func listProfiles(config *restclient.Config, ns string) error {
	plist, err := profile.List(config, ns)
	if err != nil {
		return err
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintf(w, "NAME\tGEN\tTYPE\tBUNDLES\tREPO-URL\tREPO-PATH\tOVRDS\tTAGS\tDESCRIPTION\n")
	for _, prof := range plist {
		profileType := "dynamic"
		bundles := prof.Spec.Bundles
		repoUrl := prof.Spec.RepoUrl
		if repoUrl == "" {
			repoUrl = "(N/A)"
			profileType = "static"
		}
		repoPath := prof.Spec.RepoPath
		if repoPath == "" {
			repoPath = "(N/A)"
		}
		tags := prof.Spec.Tags
		desc := prof.Spec.Description
		gen := "2"
		if prof.Legacy {
			gen = "1"
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%d\t%s\t%s\n",
			prof.Name, gen, profileType, bundles, repoUrl, repoPath,
			len(prof.Spec.Overrides),
			tags, desc)
	}
	_ = w.Flush()
	return nil
}
