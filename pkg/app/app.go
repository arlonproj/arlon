package app

import (
	"context"
	"fmt"

	argoappv1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/arlonproj/arlon/pkg/ctrlruntimeclient"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	restclient "k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const ProfilesAnnotationKey = "arlon.io/profiles"

func List(config *restclient.Config, ns string) (apslist []argoappv1.ApplicationSet, err error) {
	cli, err := ctrlruntimeclient.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to get controller runtime client: %s", err)
	}
	var asl argoappv1.ApplicationSetList
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

// -----------------------------------------------------------------------------

func Create(
	ns string,
	name string,
	destNs string,
	project string,
	srcPath string,
	srcRepoUrl string,
	srcTargetRevision string,
	autoSync bool,
	autoPrune bool,
) argoappv1.ApplicationSet {
	aps := argoappv1.ApplicationSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ApplicationSet", // can't use argoapp.ApplicationSetKind because "set" is not capitalized in that version ???
			APIVersion: argoappv1.SchemeGroupVersion.Group + "/" + argoappv1.SchemeGroupVersion.Version,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
			Labels: map[string]string{
				"arlon-type": "application",
				"managed-by": "arlon",
			},
		},
		Spec: argoappv1.ApplicationSetSpec{
			Generators: []argoappv1.ApplicationSetGenerator{
				{
					List: &argoappv1.ListGenerator{
						Elements: []apiextensionsv1.JSON{},
					},
				},
			},
			Template: argoappv1.ApplicationSetTemplate{
				ApplicationSetTemplateMeta: argoappv1.ApplicationSetTemplateMeta{
					Name: fmt.Sprintf("{{cluster_name}}-app-%s", name),
				},
				Spec: argoappv1.ApplicationSpec{
					Destination: argoappv1.ApplicationDestination{
						Namespace: destNs,
						Server:    "{{cluster_server}}",
					},
					Project: project,
					Source: argoappv1.ApplicationSource{
						Path:           srcPath,
						RepoURL:        srcRepoUrl,
						TargetRevision: srcTargetRevision,
					},
					SyncPolicy: &argoappv1.SyncPolicy{},
				},
			},
		},
	}
	if autoSync {
		aps.Spec.Template.Spec.SyncPolicy.Automated = &argoappv1.SyncPolicyAutomated{
			Prune: autoPrune,
		}
	}
	return aps
}

// -----------------------------------------------------------------------------

func Delete(config *restclient.Config, ns string, name string) error {
	cli, err := ctrlruntimeclient.NewClient(config)
	if err != nil {
		return fmt.Errorf("failed to get controller runtime client: %s", err)
	}
	var app argoappv1.ApplicationSet
	err = cli.Get(context.Background(), client.ObjectKey{
		Name:      name,
		Namespace: ns,
	}, &app)
	if err != nil {
		return fmt.Errorf("failed to get applicationset: %s", err)
	}
	if app.Labels["arlon-type"] != "application" {
		return fmt.Errorf("applicationset %s is not an arlon app", name)
	}
	err = cli.Delete(context.Background(), &app)
	if err != nil {
		return fmt.Errorf("failed to delete applicationset: %s", err)
	}
	return nil
}
