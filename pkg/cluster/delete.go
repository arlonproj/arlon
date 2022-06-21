package cluster

import (
	"context"
	"fmt"
	argoclient "github.com/argoproj/argo-cd/v2/pkg/apiclient"
	argoapp "github.com/argoproj/argo-cd/v2/pkg/apiclient/application"
	restclient "k8s.io/client-go/rest"
)

//------------------------------------------------------------------------------

func Delete(
	// appIf argoapp.ApplicationServiceClient,
	argoIf argoclient.Client,
	config *restclient.Config,
	argocdNs string,
	name string,
) error {
	//log := logpkg.GetLogger()
	conn, appIf, err := argoIf.NewApplicationClient()
	if err != nil {
		return fmt.Errorf("failed to get argocd application client: %s", err)
	}
	defer conn.Close()
	clust, err := Get(appIf, config, argocdNs, name)
	if err != nil {
		return fmt.Errorf("failed to get existing cluster: %s", err)
	}
	if clust.IsExternal {
		return UnmanageExternal(argoIf, config, argocdNs, name)
	}
	if clust.BaseCluster == nil {
		cascade := true
		_, err = appIf.Delete(
			context.Background(),
			&argoapp.ApplicationDeleteRequest{
				Name:    &name,
				Cascade: &cascade,
			})
		return err
	}
	apps, err := appIf.List(context.Background(),
		&argoapp.ApplicationQuery{Selector: "arlon-cluster=" + name})
	if err != nil {
		return fmt.Errorf("failed to list apps related to cluster: %s", err)
	}
	for _, app := range apps.Items {
		cascade := true
		_, err = appIf.Delete(
			context.Background(),
			&argoapp.ApplicationDeleteRequest{
				Name:    &app.Name,
				Cascade: &cascade,
			})
		if err != nil {
			return fmt.Errorf("failed to delete related app %s: %s",
				app.Name, err)
		}
		fmt.Println("deleted related app:", app.Name)
	}
	return nil
}
