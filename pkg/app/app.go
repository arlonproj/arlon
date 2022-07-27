package app

import (
	"context"
	"fmt"
	appset "github.com/argoproj-labs/applicationset/api/v1alpha1"
	"github.com/arlonproj/arlon/pkg/controller"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	restclient "k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func List(config *restclient.Config, ns string) (apslist []appset.ApplicationSet, err error) {
	cli, err := controller.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to get controller runtime client: %s", err)
	}
	var asl appset.ApplicationSetList
	sel := labels.NewSelector()
	req, err := labels.NewRequirement("managed-by", selection.In, []string{"arlon"})
	if err != nil {
		return nil, fmt.Errorf("failed to create requirement: %s", err)
	}
	sel.Add(*req)
	err = cli.List(context.Background(), &asl, &client.ListOptions{
		Namespace:     ns,
		LabelSelector: sel,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list applicationsets: %s", err)
	}
	return asl.Items, nil
}
