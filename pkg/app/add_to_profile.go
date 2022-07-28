package app

import (
	"context"
	"fmt"
	appset "github.com/argoproj-labs/applicationset/api/v1alpha1"
	"github.com/arlonproj/arlon/pkg/controller"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	restclient "k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// -----------------------------------------------------------------------------

func AddToProfile(
	config *restclient.Config,
	ns string,
	appName string,
	profName string,
) (modified bool, err error) {
	cli, as, err := getAndValidateApplicationSet(config, ns, appName)
	if err != nil {
		err = fmt.Errorf("failed to get applicationset: %s", err)
		return
	}
	meList := as.Spec.Generators[0].Clusters.Selector.MatchExpressions
	me := meList[0]
	for _, val := range me.Values {
		if val == profName {
			return // already on profile
		}
	}
	me.Values = append(me.Values, profName)
	meList[0] = me
	err = cli.Update(context.Background(), as)
	if err != nil {
		err = fmt.Errorf("failed to update applicationset: %s", err)
		return
	}
	modified = true
	return
}

// -----------------------------------------------------------------------------

func RemoveFromProfile(
	config *restclient.Config,
	ns string,
	appName string,
	profName string,
) (found bool, err error) {
	cli, as, err := getAndValidateApplicationSet(config, ns, appName)
	if err != nil {
		err = fmt.Errorf("failed to get and validate applicationset: %s", err)
		return
	}
	meList := as.Spec.Generators[0].Clusters.Selector.MatchExpressions
	me := meList[0]
	var newValues []string
	for _, val := range me.Values {
		if val == profName {
			found = true
		} else {
			newValues = append(newValues, val)
		}
	}
	if !found {
		return
	}
	me.Values = newValues
	meList[0] = me
	err = cli.Update(context.Background(), as)
	if err != nil {
		err = fmt.Errorf("failed to update applicationset: %s", err)
		return
	}
	return
}

// -----------------------------------------------------------------------------

func getAndValidateApplicationSet(config *restclient.Config,
	ns string,
	appName string,
) (
	cli client.Client,
	as *appset.ApplicationSet,
	err error,
) {
	cli, err = controller.NewClient(config)
	if err != nil {
		err = fmt.Errorf("failed to get controller runtime client: %s", err)
		return
	}
	as = &appset.ApplicationSet{}
	err = cli.Get(context.Background(), client.ObjectKey{
		Namespace: ns, Name: appName}, as)
	if err != nil {
		err = fmt.Errorf("failed to get applicationset: %s", err)
		return
	}
	err = validateApplicationSet(as)
	if err != nil {
		return
	}
	return
}

// -----------------------------------------------------------------------------

func validateApplicationSet(as *appset.ApplicationSet) error {
	if as.Labels["managed-by"] != "arlon" {
		return fmt.Errorf("applicationset not managed by arlon")
	}
	if len(as.Spec.Generators) != 1 {
		return fmt.Errorf("malformed applicationset: wrong number of generators")
	}
	gen := as.Spec.Generators[0]
	if gen.Clusters == nil {
		return fmt.Errorf("malformed applicationset: wrong generator type")
	}
	meList := gen.Clusters.Selector.MatchExpressions
	if len(meList) != 1 {
		return fmt.Errorf("malformed applicationset: wrong number of matchExpressions")
	}
	me := meList[0]
	if me.Key != ProfileLabelKey {
		return fmt.Errorf("malformed applicationset: wrong matchExpression key")
	}
	if me.Operator != metav1.LabelSelectorOpIn {
		return fmt.Errorf("malformed applicationset: wrong matchExpression operator")
	}
	return nil
}
