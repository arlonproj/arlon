package cluster

import (
	"context"
	"fmt"
	argoclient "github.com/argoproj/argo-cd/v2/pkg/apiclient"
	argocluster "github.com/argoproj/argo-cd/v2/pkg/apiclient/cluster"
	arlonv1 "github.com/arlonproj/arlon/api/v1"
	"github.com/arlonproj/arlon/pkg/common"
	"github.com/arlonproj/arlon/pkg/controller"
	logpkg "github.com/arlonproj/arlon/pkg/log"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	patchclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func ManageExternal(
	argoIf argoclient.Client,
	config *restclient.Config,
	argocdNs,
	clusterName string,
	prof *arlonv1.Profile,
) error {
	log := logpkg.GetLogger()
	conn, appIf, err := argoIf.NewApplicationClient()
	if err != nil {
		return fmt.Errorf("failed to get argocd application client: %s", err)
	}
	defer conn.Close()
	clist, err := List(appIf, config, argocdNs)
	if err != nil {
		return fmt.Errorf("failed to list existing clusters: %s", err)
	}
	for _, existingCluster := range clist {
		if existingCluster.Name == clusterName {
			return fmt.Errorf("cluster is already created or managed by arlon")
		}
	}
	conn2, clusterIf, err := argoIf.NewClusterClient()
	if err != nil {
		return fmt.Errorf("failed to get argocd cluster client: %s", err)
	}
	defer conn2.Close()
	_, err = clusterIf.Get(context.Background(), &argocluster.ClusterQuery{
		Name:               clusterName,
	})
	if err != nil {
		return fmt.Errorf("cluster is not registered in argocd")
	}
	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to get kube client: %s", err)
	}
	corev1 := kubeClient.CoreV1()
	secrApi := corev1.Secrets(argocdNs)
	secrs, err := secrApi.List(context.Background(), metav1.ListOptions{
		LabelSelector: "argocd.argoproj.io/secret-type=cluster",
	})
	if err != nil {
		return fmt.Errorf("failed to list cluster secrets: %s", err)
	}
	var secrPtr *v1.Secret
	for _, secr := range secrs.Items {
		name := secr.Data["name"]
		if name == nil {
			log.V(1).Info("cluster secret skipped because missing name",
				"secretName", secr.Name)
		}
		if string(name) == clusterName {
			secrPtr = &secr
			break
		}
	}
	if secrPtr == nil {
		return fmt.Errorf("failed to find matching cluster secret")
	}
	lb := secrPtr.Labels["arlon.io/cluster-type"]
	if lb != "" {
		return fmt.Errorf("unexpectedly found arlon label in secret")
	}
	newSecr := secrPtr.DeepCopy()
	newSecr.Labels["arlon.io/cluster-type"] = "external"
	newSecr.Annotations[common.ProfileAnnotationKey] = prof.Name
	cli, err := controller.NewClient(config)
	if err != nil {
		return fmt.Errorf("failed to get controller runtime client: %s", err)
	}
	err = cli.Patch(context.Background(), newSecr, patchclient.MergeFrom(secrPtr))
	if err != nil {
		return fmt.Errorf("failed to patch secret: %s", err)
	}
	return nil
}
