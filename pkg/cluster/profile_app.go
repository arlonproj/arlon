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
			Application: *app,
		}
		_, err := appIf.Create(context.Background(), &appCreateRequest)
		if err != nil {
			return nil, fmt.Errorf("failed to create profile app: %s", err)
		}
	}
	return app, nil
}
