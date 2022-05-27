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
	if prof.Spec.RepoUrl == "" {
		return fmt.Errorf("the profile is static, only dynamic is supported")
	}
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
		LabelSelector: argoClusterSecretTypeLabel,
	})
	if err != nil {
		return fmt.Errorf("failed to list argo cluster secrets: %s", err)
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
	lb := secrPtr.Labels[clusterTypeLabelKey]
	if lb != "" {
		return fmt.Errorf("unexpectedly found arlon label in secret")
	}
	newSecr := secrPtr.DeepCopy()
	newSecr.Labels[clusterTypeLabelKey] = "external"
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


func UnmanageExternal(
	argoIf argoclient.Client,
	config *restclient.Config,
	argocdNs,
	clusterName string,
) error {
	conn, appIf, err := argoIf.NewApplicationClient()
	if err != nil {
		return fmt.Errorf("failed to get argocd application client: %s", err)
	}
	defer conn.Close()
	clist, err := List(appIf, config, argocdNs)
	if err != nil {
		return fmt.Errorf("failed to list existing clusters: %s", err)
	}
	var foundCluster *Cluster
	for _, existingCluster := range clist {
		if existingCluster.Name == clusterName {
			foundCluster = &existingCluster
			break
		}
	}
	if foundCluster == nil {
		return fmt.Errorf("cluster does not exist")
	}
	if !foundCluster.IsExternal {
		return fmt.Errorf("cluster is not external")
	}
	if foundCluster.SecretName == "" {
		return fmt.Errorf("cluster is missing secret information")
	}
	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to get kube client: %s", err)
	}
	corev1 := kubeClient.CoreV1()
	secrApi := corev1.Secrets(argocdNs)
	secr, err := secrApi.Get(context.Background(), foundCluster.SecretName,
		metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get argo cluster secret: %s", err)
	}
	if string(secr.Data["name"]) != clusterName {
		return fmt.Errorf("secret data does not match cluster name")
	}
	if secr.Labels[clusterTypeLabelKey] != "external" {
		return fmt.Errorf("secret does not have arlon cluster label")
	}
	newSecr := secr.DeepCopy()
	delete(newSecr.Labels, clusterTypeLabelKey)
	cli, err := controller.NewClient(config)
	if err != nil {
		return fmt.Errorf("failed to get controller runtime client: %s", err)
	}
	err = cli.Patch(context.Background(), newSecr, patchclient.MergeFrom(secr))
	if err != nil {
		return fmt.Errorf("failed to patch secret: %s", err)
	}
	return nil
}
