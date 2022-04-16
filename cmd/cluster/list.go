package cluster

import (
	"context"
	"fmt"
	apppkg "github.com/argoproj/argo-cd/v2/pkg/apiclient/application"
	argoappv1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/argoproj/argo-cd/v2/util/errors"
	"github.com/argoproj/argo-cd/v2/util/io"
	"github.com/platform9/arlon/pkg/argocd"
	"github.com/platform9/arlon/pkg/common"
	"github.com/spf13/cobra"
	"os"
	"text/tabwriter"
)

func listClustersCommand() *cobra.Command {
	command := &cobra.Command{
		Use:               "list",
		Short:             "List the clusters managed by Arlon",
		Long:              "List the clusters managed by Arlon",
		DisableAutoGenTag: true,
		Run: func(c *cobra.Command, args []string) {
			listClusters()
		},
	}
	return command
}

func listClusters() {
	conn, appIf := argocd.NewArgocdClientOrDie("").NewApplicationClientOrDie()
	defer io.Close(conn)
	apps, err := appIf.List(context.Background(),
		&apppkg.ApplicationQuery{Selector: "managed-by=arlon,arlon-type=cluster"})
	errors.CheckError(err)
	printClusterTable(apps.Items)
}

func printClusterTable(apps []argoappv1.Application) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintf(w, "NAME\tCLUSTERSPEC\tPROFILE\t\n")
	for _, a := range apps {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", a.Name,
			a.Annotations[common.ClusterSpecAnnotationKey],
			a.Annotations[common.ProfileAnnotationKey])
	}
	_ = w.Flush()
}
