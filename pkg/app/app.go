package app

import (
	"context"
	"fmt"
	appset "github.com/argoproj/argo-cd/v2/pkg/apis/applicationset/v1alpha1"
	"github.com/arlonproj/arlon/pkg/controller"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	restclient "k8s.io/client-go/rest"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"text/tabwriter"
)

const ProfileLabelKey = "arlon.io/profile"

func List(config *restclient.Config, ns string) (apslist []appset.ApplicationSet, err error) {
	cli, err := controller.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to get controller runtime client: %s", err)
	}
	var asl appset.ApplicationSetList
	req, err := labels.NewRequirement("managed-by", selection.In, []string{"arlon"})
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

func ListToStdout(config *restclient.Config, ns string) error {
	apps, err := List(config, ns)
	if err != nil {
		return fmt.Errorf("failed to list apps: %s", err)
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintf(w, "NAME\tREPO\tPATH\tREVISION\tPROFILES\n")
	for _, app := range apps {
		profiles := "(none)"
		if len(app.Spec.Generators) == 1 &&
			app.Spec.Generators[0].Clusters != nil &&
			len(app.Spec.Generators[0].Clusters.Selector.MatchExpressions) >= 1 {
			if app.Spec.Generators[0].Clusters.Selector.MatchExpressions[0].Key != ProfileLabelKey ||
				app.Spec.Generators[0].Clusters.Selector.MatchExpressions[0].Operator != metav1.LabelSelectorOpIn {
				profiles = "(invalid data)"
			} else {
				profiles = fmt.Sprintf("%s", app.Spec.Generators[0].Clusters.Selector.MatchExpressions[0].Values)
			}
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			app.Name,
			app.Spec.Template.Spec.Source.RepoURL,
			app.Spec.Template.Spec.Source.Path,
			app.Spec.Template.Spec.Source.TargetRevision,
			profiles,
		)
	}
	_ = w.Flush()
	return nil
}
