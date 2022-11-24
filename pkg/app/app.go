package app

import (
	"context"
	"fmt"
	argoappv1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	appset "github.com/argoproj/argo-cd/v2/pkg/apis/applicationset/v1alpha1"
	"github.com/arlonproj/arlon/pkg/ctrlruntimeclient"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
) appset.ApplicationSet {
	aps := appset.ApplicationSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ApplicationSet", // can't use argoapp.ApplicationSetKind because "set" is not capitalized in that version ???
			APIVersion: appset.GroupVersion.Group + "/" + appset.GroupVersion.Version,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
			Labels: map[string]string{
				"arlon-type": "application",
				"managed-by": "arlon",
			},
		},
		Spec: appset.ApplicationSetSpec{
			Generators: []appset.ApplicationSetGenerator{
				{
					List: &appset.ListGenerator{
						Elements: []apiextensionsv1.JSON{},
					},
				},
			},
			Template: appset.ApplicationSetTemplate{
				ApplicationSetTemplateMeta: appset.ApplicationSetTemplateMeta{
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
