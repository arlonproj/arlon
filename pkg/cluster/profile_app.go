package cluster

import (
	"context"
	"fmt"

	argoapp "github.com/argoproj/argo-cd/v2/pkg/apiclient/application"
	argoappv1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	arlonv1 "github.com/arlonproj/arlon/api/v1"
)

// CreateProfileApp creates a profile-app that accompanies an arlon-app for gen2 clusters
func CreateProfileApp(
	profileAppName string,
	appIf argoapp.ApplicationServiceClient,
	argocdNs string,
	clusterName string,
	prof *arlonv1.Profile,
	createInArgoCd bool,
) (*argoappv1.Application, error) {
	app := constructProfileApp(profileAppName, argocdNs, clusterName, prof)
	if createInArgoCd {
		appCreateRequest := argoapp.ApplicationCreateRequest{
			Application: app,
		}
		_, err := appIf.Create(context.Background(), &appCreateRequest)
		if err != nil {
			return nil, fmt.Errorf("failed to create profile app: %s", err)
		}
	}
	return app, nil
}
func DestructProfileApp(
	appIf argoapp.ApplicationServiceClient,
	clusterName string,
) error {
	var err error
	apps, err := appIf.List(context.Background(),
		&argoapp.ApplicationQuery{Selector: "arlon-cluster=" + clusterName + ",arlon-type=profile-app"})
	for _, app := range apps.Items {
		cascade := true
		_, err = appIf.Delete(
			context.Background(),
			&argoapp.ApplicationDeleteRequest{
				Name:    &app.Name,
				Cascade: &cascade,
			})
		if err != nil {
			return fmt.Errorf("failed to delete related profile app %s: %s",
				app.Name, err)
		}
		fmt.Println("deleted related app:", app.Name)
	}
	return err
}
