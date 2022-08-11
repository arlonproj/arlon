package bundle

import (
	"fmt"
	"github.com/arlonproj/arlon/pkg/bundle"
	"github.com/spf13/cobra"
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
	bundles, err := bundle.List(config, ns)
	if err != nil {
		return err
	}
	if len(bundles) == 0 {
		fmt.Println("no bundles found")
		return nil
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintf(w, "NAME\tTYPE\tTAGS\tREPO\tPATH\tREVISION\tSRCTYPE\tDESCRIPTION\n")
	for _, bundleItm := range bundles {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n", bundleItm.Name,
			bundleItm.Type, bundleItm.Tags, bundleItm.Repo, bundleItm.Path, bundleItm.Revision, bundleItm.SrcType, bundleItm.Description)
	}
	_ = w.Flush()
	return nil
}
