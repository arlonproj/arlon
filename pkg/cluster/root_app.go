package cluster

import (
	"arlon.io/arlon/pkg/clusterspec"
	"arlon.io/arlon/pkg/common"
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
			Labels: map[string]string{"managed-by": "arlon","arlon-type":"cluster"},
			Annotations: map[string]string{common.ClusterSpecAnnotationKey: clusterSpecName},
			Finalizers: []string{argoappv1.ForegroundPropagationPolicyFinalizer},
		},
	}
	cs, err := clusterspec.FromConfigMap(cm)
	if err != nil {
		return nil, fmt.Errorf("failed to read clusterspec from configmap: %s", err)
	}
	helmParams := [] argoappv1.HelmParameter{
		{
			Name:  "global.clusterName",
			Value: clusterName,
		},
		{
			Name: "global.kubeconfigSecretKeyName",
			Value: clusterspec.KubeconfigSecretKeyNameByApiProvider[cs.ApiProvider],
		},
	}
	for _, key := range clusterspec.ValidHelmParamKeys {
		val := cm.Data[key]
		if val != "" {
			helmParams = append(helmParams, argoappv1.HelmParameter{
				Name: fmt.Sprintf("global.%s", key),
				Value: val,
			})
		}
	}
	subchartName, err := clusterspec.SubchartName(cm)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve subchart name: %s", err)
	}
	helmParams = append(helmParams, argoappv1.HelmParameter{
		Name: fmt.Sprintf("tags.%s", subchartName),
		Value: "true",
	})
	app.Spec.Source.Helm = &argoappv1.ApplicationSourceHelm{Parameters: helmParams}
	app.Spec.Source.RepoURL = repoUrl
	app.Spec.Source.TargetRevision = repoBranch
	app.Spec.Source.Path = path.Join(basePath, clusterName, "mgmt")
	app.Spec.Destination.Server = "https://kubernetes.default.svc"
	app.Spec.Destination.Namespace = "default"
	app.Spec.SyncPolicy = &argoappv1.SyncPolicy{
		Automated: &argoappv1.SyncPolicyAutomated{
			Prune: true,
		},
		SyncOptions: []string{"Prune=true"},
	}
	// Ignore CAPI EKS control plane's spec.version because the AWS controller(s)
	// appear to update it with a value that is less precise than the requested
	// one, for e.g. the spec might specify v1.18.16, and get updated with v1.18,
	// causing ArgoCD to report the resource as OutOfSync
	app.Spec.IgnoreDifferences = []argoappv1.ResourceIgnoreDifferences{
		{
			Group: "controlplane.cluster.x-k8s.io",
			Kind: "AWSManagedControlPlane",
			JSONPointers: []string{"/spec/version"},
		},
	}
	return app, nil
}
