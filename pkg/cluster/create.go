package cluster

import (
	"context"
	"fmt"
	argoapp "github.com/argoproj/argo-cd/v2/pkg/apiclient/application"
	argoappv1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	arlonv1 "github.com/arlonproj/arlon/api/v1"
	grpccodes "google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
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
	repoUrl,
	repoBranch,
	basePath,
	clusterSpecName string,
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
	corev1 := kubeClient.CoreV1()
	configMapsApi := corev1.ConfigMaps(arlonNs)
	cm, err := configMapsApi.Get(context.Background(), clusterSpecName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get clusterspec configmap: %s", err)
	}
	repoPath := mgmtPathFromBasePath(basePath, clusterName)
	rootApp, err := ConstructRootApp(argocdNs, clusterName, repoUrl, repoBranch,
		repoPath, clusterSpecName, cm, prof.Name, managementClusterUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to construct root app: %s", err)
	}
	err = DeployToGit(config, argocdNs, arlonNs, clusterName,
		repoUrl, repoBranch, basePath, prof)
	if err != nil {
		return nil, fmt.Errorf("failed to deploy git tree: %s", err)
	}
	if createInArgoCd {
		appCreateRequest := argoapp.ApplicationCreateRequest{
			Application: *rootApp,
		}
		_, err := appIf.Create(context.Background(), &appCreateRequest)
		if err != nil {
			return nil, fmt.Errorf("failed to create ArgoCD root application: %s", err)
		}
	}
	return rootApp, nil
}
