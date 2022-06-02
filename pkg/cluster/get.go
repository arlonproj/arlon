package cluster

import (
	"context"
	"fmt"
	apppkg "github.com/argoproj/argo-cd/v2/pkg/apiclient/application"
	argoapp "github.com/argoproj/argo-cd/v2/pkg/apiclient/application"
	"github.com/arlonproj/arlon/pkg/common"
	logpkg "github.com/arlonproj/arlon/pkg/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
)

//------------------------------------------------------------------------------

func Get(
	appIf argoapp.ApplicationServiceClient,
	config *restclient.Config,
	argocdNs string,
	name string,
) (cl *Cluster, err error) {
	log := logpkg.GetLogger()
	app, err := appIf.Get(context.Background(),
		&apppkg.ApplicationQuery{
			Name: &name,
			Selector: "managed-by=arlon,arlon-type=cluster",
		})
	if err == nil {
		return &Cluster{
			Name: app.Name,
			ClusterSpecName: app.Annotations[common.ClusterSpecAnnotationKey],
			ProfileName: app.Annotations[common.ProfileAnnotationKey],
		}, nil
	}
	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to get kube client: %s", err)
	}
	corev1 := kubeClient.CoreV1()
	secrApi := corev1.Secrets(argocdNs)
	secrs, err := secrApi.List(context.Background(), metav1.ListOptions{
		LabelSelector: argoClusterSecretTypeLabel + "," + externalClusterTypeLabel,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list cluster secrets: %s", err)
	}
	for _, secr := range secrs.Items {
		clusterName := secr.Data["name"]
		if clusterName == nil {
			log.V(1).Info("cluster secret skipped because missing cluster name",
				"secretName", secr.Name)
		}
		if string(clusterName) == name {
			return &Cluster{
				Name:        name,
				ProfileName: secr.Annotations[common.ProfileAnnotationKey],
				IsExternal:  true,
				SecretName: secr.Name,
			}, nil
		}
	}
	return nil, fmt.Errorf("not found")
}
