package cluster

import (
	"context"
	"fmt"
	argoapp "github.com/argoproj/argo-cd/v2/pkg/apiclient/application"
	argoappv1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/platform9/arlon/pkg/clusterspec"
	"github.com/platform9/arlon/pkg/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func Update(
	appIf argoapp.ApplicationServiceClient,
	kubeClient *kubernetes.Clientset,
	argocdNs,
	arlonNs,
	clusterName,
	clusterSpecName,
	profileName string,
	updateInArgoCd bool,
	managementClusterUrl string,
) (*argoappv1.Application, error) {
	oldApp, err := appIf.Get(context.Background(),
		&argoapp.ApplicationQuery{Name: &clusterName})
	if err != nil {
		return nil, fmt.Errorf("failed to get argocd app: %s", err)
	}
	if profileName == "" {
		profileName = oldApp.Annotations[common.ProfileAnnotationKey]
		if profileName == "" {
			return nil, fmt.Errorf("existing cluster root app is missing profile annotation")
		}
	}
	if clusterSpecName == "" {
		clusterSpecName = oldApp.Annotations[common.ClusterSpecAnnotationKey]
		if clusterSpecName == "" {
			return nil, fmt.Errorf("existing cluster root app is missing clusterspec annotation")
		}
	}
	corev1 := kubeClient.CoreV1()
	configMapsApi := corev1.ConfigMaps(arlonNs)
	clusterSpecCm, err := configMapsApi.Get(context.Background(), clusterSpecName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get clusterspec configmap: %s", err)
	}
	// Ensure subchart name (api, cloud, clustertype) hasn't changed
	subchartName, err := clusterspec.SubchartName(clusterSpecCm)
	helmParamName := fmt.Sprintf("tags.%s", subchartName)
	found := false
	for _, param := range oldApp.Spec.Source.Helm.Parameters {
		found = param.Name == helmParamName && param.Value == "true"
		if found {
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("the api provider, cloud provider, or cluster type cannot change")
	}
	repoUrl := oldApp.Spec.Source.RepoURL
	repoBranch := oldApp.Spec.Source.TargetRevision
	repoPath := oldApp.Spec.Source.Path
	basePath, clstName, err := decomposePath(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to decompose repo path: %s", err)
	}
	if clstName != clusterName {
		return nil, fmt.Errorf("unexpected cluster name extracted from repo path: %s",
			clstName)
	}
	rootApp, err := ConstructRootApp(argocdNs, clusterName, repoUrl,
		repoBranch, repoPath, clusterSpecName, clusterSpecCm, profileName,
		managementClusterUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to construct root app: %s", err)
	}
	if oldApp.Spec.Source.RepoURL != rootApp.Spec.Source.RepoURL ||
		oldApp.Spec.Source.Path != rootApp.Spec.Source.Path {
		return nil, fmt.Errorf("git repo reference cannot change")
	}
	err = DeployToGit(kubeClient, argocdNs, arlonNs, clusterName,
		repoUrl, repoBranch, basePath, profileName)
	if err != nil {
		return nil, fmt.Errorf("failed to deploy git tree: %s", err)
	}
	if updateInArgoCd {
		appUpdateRequest := argoapp.ApplicationUpdateRequest{
			Application: rootApp,
		}
		_, err := appIf.Update(context.Background(), &appUpdateRequest)
		if err != nil {
			return nil, fmt.Errorf("failed to update ArgoCD root application: %s", err)
		}
	}
	return rootApp, nil
}
