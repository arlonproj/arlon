package cluster

import (
	"context"
	"fmt"

	argoapp "github.com/argoproj/argo-cd/v2/pkg/apiclient/application"
	argoappv1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	arlonv1 "github.com/arlonproj/arlon/api/v1"
	"github.com/arlonproj/arlon/pkg/argocd"
	"github.com/arlonproj/arlon/pkg/bundle"
	grpccodes "google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
)

func Create(
	appIf argoapp.ApplicationServiceClient,
	config *restclient.Config,
	argocdNs,
	arlonNs,
	clusterName,
	baseClusterName, // gen2 only, leave empty otherwise
	repoUrl, // gen1: target git repo, gen2: source git repo containing base
	repoBranch, // gen1: target git revision, gen2: source git revision containing base
	basePath, // gen1: target git path, gen2: source git path containing base
	clusterSpecName string, // empty for gen2
	prof *arlonv1.Profile,
	createInArgoCd bool,
	managementClusterUrl string,
) (*argoappv1.Application, error) {
	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to get kube client: %s", err)
	}
	_, err = appIf.Get(context.Background(),
		&argoapp.ApplicationQuery{Name: &clusterName})
	if err == nil {
		return nil, fmt.Errorf("arlon cluster already exists")
	}
	grpcStatus, ok := grpcstatus.FromError(err)
	if !ok {
		return nil, fmt.Errorf("failed to get grpc status from error")
	}
	if grpcStatus.Code() != grpccodes.NotFound {
		return nil, fmt.Errorf("unexpected cluster application error code: %d",
			grpcStatus.Code())
	}
	var cm *v1.ConfigMap
	if clusterSpecName != "" {
		corev1 := kubeClient.CoreV1()
		configMapsApi := corev1.ConfigMaps(arlonNs)
		cm, err = configMapsApi.Get(context.Background(), clusterSpecName, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to get clusterspec configmap: %s", err)
		}
	}
	repoPath := basePath // default for gen2
	if clusterSpecName != "" {
		repoPath = mgmtPathFromBasePath(basePath, clusterName) // gen1
	}
	profileName := "no-profile-for-gen2"
	if prof != nil {
		profileName = prof.Name
	}
	rootApp, err := ConstructRootApp(argocdNs, clusterName, baseClusterName, repoUrl, repoBranch,
		repoPath, clusterSpecName, cm, profileName, managementClusterUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to construct root app: %s", err)
	}
	if clusterSpecName != "" {
		// gen1 only: deploy cluster files to git
		creds, err := argocd.GetRepoCredsFromArgoCd(kubeClient, argocdNs, repoUrl)
		if err != nil {
			return nil, fmt.Errorf("failed to get repo credentials: %s", err)
		}
		corev1 := kubeClient.CoreV1()
		bundles, err := bundle.GetBundlesFromProfile(prof, corev1, arlonNs)
		if err != nil {
			return nil, fmt.Errorf("failed to get bundles from profile: %s", err)
		}
		err = DeployToGit(creds, argocdNs, bundles, clusterName,
			repoUrl, repoBranch, basePath, prof)
		if err != nil {
			return nil, fmt.Errorf("failed to deploy git tree: %s", err)
		}
	}
	if createInArgoCd {
		appCreateRequest := argoapp.ApplicationCreateRequest{
			Application: rootApp,
		}
		_, err := appIf.Create(context.Background(), &appCreateRequest)
		if err != nil {
			return nil, fmt.Errorf("failed to create ArgoCD root application: %s", err)
		}
	}
	return rootApp, nil
}
