package app

import (
	"context"
	"fmt"
	appset "github.com/argoproj/argo-cd/v2/pkg/apis/applicationset/v1alpha1"
	"github.com/arlonproj/arlon/pkg/ctrlruntimeclient"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	restclient "k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const ProfileLabelKey = "arlon.io/profile"
const ProfilesAnnotationKey = "arlon.io/profiles"

func List(config *restclient.Config, ns string) (apslist []appset.ApplicationSet, err error) {
	cli, err := ctrlruntimeclient.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to get controller runtime client: %s", err)
	}
	var asl appset.ApplicationSetList
	req, err := labels.NewRequirement("arlon-type", selection.In, []string{"application"})
	if err != nil {
		return nil, fmt.Errorf("failed to create requirement: %s", err)
	}
	sel := labels.NewSelector().Add(*req)
	err = cli.List(context.Background(), &asl, &client.ListOptions{
		Namespace:     ns,
		LabelSelector: sel,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list applicationsets: %s", err)
	}
	return asl.Items, nil
}
