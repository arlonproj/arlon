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

func AddToProfile(
	config *restclient.Config,
	ns string,
	appName string,
	profName string,
) (modified bool, err error) {
	cli, err := controller.NewClient(config)
	if err != nil {
		err = fmt.Errorf("failed to get controller runtime client: %s", err)
		return
	}
	var as appset.ApplicationSet
	err = cli.Get(context.Background(), client.ObjectKey{
		Namespace: ns, Name: appName}, &as)
	if err != nil {
		err = fmt.Errorf("failed to get applicationset: %s", err)
		return
	}
	if as.Labels["managed-by"] != "arlon" {
		err = fmt.Errorf("applicationset not managed by arlon")
		return
	}
	if len(as.Spec.Generators) != 1 {
		err = fmt.Errorf("malformed applicationset: wrong number of generators")
		return
	}
	gen := as.Spec.Generators[0]
	if gen.Clusters == nil {
		err = fmt.Errorf("malformed applicationset: wrong generator type")
		return
	}
	meList := gen.Clusters.Selector.MatchExpressions
	if len(meList) != 1 {
		err = fmt.Errorf("malformed applicationset: wrong number of matchExpressions")
		return
	}
	me := meList[0]
	if me.Key != ProfileLabelKey {
		err = fmt.Errorf("malformed applicationset: wrong matchExpression key")
		return
	}
	if me.Operator != metav1.LabelSelectorOpIn {
		err = fmt.Errorf("malformed applicationset: wrong matchExpression operator")
		return
	}
	for _, val := range me.Values {
		if val == profName {
			return // already on profile
		}
	}
	me.Values = append(me.Values, profName)
	meList[0] = me
	err = cli.Update(context.Background(), &as)
	if err != nil {
		err = fmt.Errorf("failed to update applicationset: %s", err)
		return
	}
	modified = true
	return
}
