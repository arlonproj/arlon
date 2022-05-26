package cluster

import (
	"context"
	"fmt"
	apppkg "github.com/argoproj/argo-cd/v2/pkg/apiclient/application"
	argoapp "github.com/argoproj/argo-cd/v2/pkg/apiclient/application"
	"github.com/arlonproj/arlon/pkg/common"
	"k8s.io/client-go/kubernetes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	restclient "k8s.io/client-go/rest"
)

func List(
	appIf argoapp.ApplicationServiceClient,
	config *restclient.Config,
	argocdNs string,
) (clist []Cluster, err error) {
	apps, err := appIf.List(context.Background(),
		&apppkg.ApplicationQuery{Selector: "managed-by=arlon,arlon-type=cluster"})
	if err != nil {
		return nil, fmt.Errorf("failed to list argocd applications: %s", err)
	}
	for _, a := range apps.Items {
		clist = append(clist, Cluster{
			Name: a.Name,
			ClusterSpecName: a.Annotations[common.ClusterSpecAnnotationKey],
			ProfileName: a.Annotations[common.ProfileAnnotationKey],
		})
	}
	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to get kube client: %s", err)
	}
	corev1 := kubeClient.CoreV1()
	secrApi := corev1.Secrets(argocdNs)
	secrs, err := secrApi.List(context.Background(), metav1.ListOptions{
		LabelSelector: "argocd.argoproj.io/secret-type=cluster,arlon.io/cluster-type=unmanaged",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list cluster secrets: %s", err)
	}
	for _, secr := range secrs.Items {
		clist = append(clist, Cluster{
			Name: secr.Name,
			ProfileName: secr.Annotations[common.ProfileAnnotationKey],
			IsUnmanaged: true,
		})
	}
	return
}
