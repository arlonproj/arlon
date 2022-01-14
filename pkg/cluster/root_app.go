package cluster

import (
	"context"
	"fmt"
	"github.com/argoproj/argo-cd/v2/pkg/apis/application"
	argoappv1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"path"
)

func ConstructRootApp(
	kubeClient *kubernetes.Clientset,
	argocdNs string,
	arlonNs string,
	clusterName string,
	repoUrl string,
	repoBranch string,
	basePath string,
	clusterSpecName string,
) (*argoappv1.Application, error) {
	corev1 := kubeClient.CoreV1()
	configMapsApi := corev1.ConfigMaps(arlonNs)
	cm, err := configMapsApi.Get(context.Background(), clusterSpecName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get clusterspec configmap: %s", err)
	}
	app := &argoappv1.Application{
		TypeMeta: v1.TypeMeta{
			Kind:       application.ApplicationKind,
			APIVersion: application.Group + "/v1alpha1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name: clusterName,
			Namespace: argocdNs,
		},
	}
	keys := []string{
		"region", "sshKeyName", "kubernetesVersion", "podCidrBlock", "nodeCount", "nodeType",
	}
	var helmParams [] argoappv1.HelmParameter
	for _, key := range keys {
		val := cm.Data[key]
		if val != "" {
			helmParams = append(helmParams, argoappv1.HelmParameter{
				Name: key,
				Value: val,
			})
		}
	}
	app.Spec.Source.Helm = &argoappv1.ApplicationSourceHelm{Parameters: helmParams}
	app.Spec.Source.RepoURL = repoUrl
	app.Spec.Source.TargetRevision = repoBranch
	app.Spec.Source.Path = path.Join(basePath, clusterName, "mgmt")
	app.Spec.Destination.Server = "https://kubernetes.default.svc"
	app.Spec.Destination.Namespace = argocdNs
	app.Spec.SyncPolicy = &argoappv1.SyncPolicy{
		Automated: &argoappv1.SyncPolicyAutomated{
			Prune: true,
		},
		SyncOptions: []string{"Prune=true"},
	}
	return app, nil
}
