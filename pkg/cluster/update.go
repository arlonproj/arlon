package cluster

import (
	"context"
	"fmt"
	argoapp "github.com/argoproj/argo-cd/v2/pkg/apiclient/application"
	argoappv1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/arlonproj/arlon/pkg/clusterspec"
	"github.com/arlonproj/arlon/pkg/common"
	"github.com/arlonproj/arlon/pkg/profile"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
)

// Update modifies a cluster to use a different cluster spec or profile,
// or both. Only some specific changes are allowed. For example, the API
// provider, cloud provider, or cluster type cannot change, meaning if a
// new cluster spec is chosen, it must preserve those values.
// There are no restrictions on the new profile, if one is specified.
// Bundles associated with the old profile will automatically be removed from
// the cluster.
func Update(
	appIf argoapp.ApplicationServiceClient,
	config *restclient.Config,
	argocdNs,
	arlonNs,
	clusterName,
	clusterSpecName string,
	profileName string,
	updateInArgoCd bool,
	managementClusterUrl string,
) (*argoappv1.Application, error) {
	oldApp, err := appIf.Get(context.Background(),
		&argoapp.ApplicationQuery{Name: &clusterName})
	if err != nil {
		return nil, fmt.Errorf("failed to get argocd app: %s", err)
	}
	if clusterSpecName == "" {
		clusterSpecName = oldApp.Annotations[common.ClusterSpecAnnotationKey]
		if clusterSpecName == "" {
			return nil, fmt.Errorf("existing cluster root app is missing clusterspec annotation")
		}
	}
	if profileName == "" {
		profileName = oldApp.Annotations[common.ProfileAnnotationKey]
		if profileName == "" {
			return nil, fmt.Errorf("existing cluster root app is missing profile annotation")
		}
	}
	prof, err := profile.Get(config, profileName, arlonNs)
	if err != nil {
		return nil, fmt.Errorf("failed to get profile: %s", err)
	}
	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to get kube client: %s", err)
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
		repoBranch, repoPath, clusterSpecName, clusterSpecCm, prof.Name,
		managementClusterUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to construct root app: %s", err)
	}
	if oldApp.Spec.Source.RepoURL != rootApp.Spec.Source.RepoURL ||
		oldApp.Spec.Source.Path != rootApp.Spec.Source.Path {
		return nil, fmt.Errorf("git repo reference cannot change")
	}
	err = DeployToGit(config, argocdNs, arlonNs, clusterName,
		repoUrl, repoBranch, basePath, prof)
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
