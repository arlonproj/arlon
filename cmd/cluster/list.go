package cluster

import (
	"fmt"
	"github.com/argoproj/argo-cd/v2/util/cli"
	"github.com/argoproj/argo-cd/v2/util/io"
	"github.com/arlonproj/arlon/pkg/argocd"
	"github.com/arlonproj/arlon/pkg/cluster"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"text/tabwriter"
)

func listClustersCommand() *cobra.Command {
	var clientConfig clientcmd.ClientConfig
	var argocdNs string
	command := &cobra.Command{
		Use:               "list",
		Short:             "List the clusters managed by Arlon",
		Long:              "List the clusters managed by Arlon",
		DisableAutoGenTag: true,
		RunE: func(c *cobra.Command, args []string) error {
			return listClusters(clientConfig, argocdNs)
		},
	}
	clientConfig = cli.AddKubectlFlagsToCmd(command)
	command.Flags().StringVar(&argocdNs, "argocd-ns", "argocd", "the argocd namespace")
	return command
}

func listClusters(clientConfig clientcmd.ClientConfig, argocdNs string) error {
	conn, appIf := argocd.NewArgocdClientOrDie("").NewApplicationClientOrDie()
	defer io.Close(conn)
	config, err := clientConfig.ClientConfig()
	if err != nil {
		return fmt.Errorf("failed to get k8s client config: %s", err)
	}
	clist, err := cluster.List(appIf, config, argocdNs)
	if err != nil {
		return fmt.Errorf("failed to list clusters: %s", err)
	}
	printClusterTable(clist)
	return nil
}

func printClusterTable(clist []cluster.Cluster) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintf(w, "NAME\tEXTERNAL\tCLUSTERSPEC\tPROFILE\t\n")
	for _, c := range clist {
		_, _ = fmt.Fprintf(w, "%s\t%v\t%s\t%s\n", c.Name, c.IsExternal,
			c.ClusterSpecName, c.ProfileName)
	}
	_ = w.Flush()
}
