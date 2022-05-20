package list_clusters

import (
	"context"
	"fmt"
	clusterpkg "github.com/argoproj/argo-cd/v2/pkg/apiclient/cluster"
	argoappv1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/argoproj/argo-cd/v2/util/errors"
	"github.com/argoproj/argo-cd/v2/util/io"
	"github.com/arlonproj/arlon/pkg/argocd"
	"github.com/spf13/cobra"
	"os"
	"text/tabwriter"
)

func NewCommand() *cobra.Command {
	command := &cobra.Command{
		Use:               "listclusters",
		Short:             "List the clusters registered with ArgoCD",
		Long:              "List the clusters registered with ArgoCD",
		DisableAutoGenTag: true,
		Run: func(c *cobra.Command, args []string) {
			listClusters()
		},
	}
	return command
}

func listClusters() {
	conn, clusterIf := argocd.NewArgocdClientOrDie("").NewClusterClientOrDie()
	defer io.Close(conn)
	clusters, err := clusterIf.List(context.Background(), &clusterpkg.ClusterQuery{})
	errors.CheckError(err)
	printClusterTable(clusters.Items)
}

// Copied from argo-cd/cmd/argocd/commands/cluster.go
// Print table of cluster information
func printClusterTable(clusters []argoappv1.Cluster) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintf(w, "SERVER\tNAME\tVERSION\tSTATUS\tMESSAGE\n")
	for _, c := range clusters {
		server := c.Server
		if len(c.Namespaces) > 0 {
			server = fmt.Sprintf("%s (%d namespaces)", c.Server, len(c.Namespaces))
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", server, c.Name, c.ServerVersion, c.ConnectionState.Status, c.ConnectionState.Message)
	}
	_ = w.Flush()
}
