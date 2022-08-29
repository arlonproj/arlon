package cluster

import (
	"context"
	"fmt"

	argoapp "github.com/argoproj/argo-cd/v2/pkg/apiclient/application"
	argoappv1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/arlonproj/arlon/pkg/profile"
	restclient "k8s.io/client-go/rest"
)

func NgUpdate(
	appIf argoapp.ApplicationServiceClient,
	config *restclient.Config,
	argocdNs,
	arlonNs,
	clusterName,
	profileName string,
	updateInArgoCd bool,
) (*argoappv1.Application, error) {

	prof, err := profile.Get(config, profileName, arlonNs)
	if err != nil {
		return nil, fmt.Errorf("failed to get profile: %s", err)
	}
	apps, err := appIf.List(context.Background(),
		&argoapp.ApplicationQuery{Selector: "arlon-cluster=" + clusterName + ",arlon-type=profile-app"})
	if len(apps.Items) != 0 {
		DestructProfileApp(appIf, clusterName)
	}
	profileAppName := fmt.Sprintf("%s-profile-%s", clusterName, prof.Name)
	profileApp, err := CreateProfileApp(profileAppName,
		appIf, argocdNs, clusterName, prof, updateInArgoCd)
	if err != nil {
		return nil, fmt.Errorf("failed to create profile app: %s", err)
	}
	return profileApp, nil
}
