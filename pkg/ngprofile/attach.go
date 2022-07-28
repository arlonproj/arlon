package ngprofile

import (
	"context"
	"fmt"
	argoclient "github.com/argoproj/argo-cd/v2/pkg/apiclient"
	clusterpkg "github.com/argoproj/argo-cd/v2/pkg/apiclient/cluster"
	"github.com/arlonproj/arlon/pkg/app"
)

func AttachToCluster(argoIf argoclient.Client, profName string, clusterName string,
) (modified bool, err error) {
	conn, clustIf, err := argoIf.NewClusterClient()
	if err != nil {
		err = fmt.Errorf("failed to get argocd cluster client: %s", err)
		return
	}
	defer conn.Close()
	clust, err := clustIf.Get(context.Background(), &clusterpkg.ClusterQuery{Name: clusterName})
	if err != nil {
		err = fmt.Errorf("failed to get argocd cluster: %s", err)
		return
	}
	if clust.Labels[app.ProfileLabelKey] == profName {
		// already has it
		return
	}
	clust.Labels[app.ProfileLabelKey] = profName
	_, err = clustIf.Update(context.Background(), &clusterpkg.ClusterUpdateRequest{
		Cluster: clust,
	})
	if err != nil {
		err = fmt.Errorf("failed to update argocd cluster: %s", err)
		return
	}
	modified = true
	return
}
