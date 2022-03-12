package cluster

import (
	"arlon.io/arlon/pkg/common"
	"context"
	"fmt"
	argoapp "github.com/argoproj/argo-cd/v2/pkg/apiclient/application"
	argoappv1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	grpccodes "google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
	"k8s.io/client-go/kubernetes"
)

func Create(
	appIf argoapp.ApplicationServiceClient,
	kubeClient *kubernetes.Clientset,
	argocdNs,
	arlonNs,
	clusterName,
	repoUrl,
	repoBranch,
	basePath,
    clusterSpecName,
	profileName string,
	createInArgoCd bool,
) (*argoappv1.Application, error) {
	_, err := appIf.Get(context.Background(),
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
	if err != nil {
		return nil, fmt.Errorf("failed to get k8s client config: %s", err)
	}
	rootApp, err := ConstructRootApp(kubeClient, argocdNs, arlonNs,
		clusterName, repoUrl, repoBranch, basePath, clusterSpecName)
	if err != nil {
		return nil, fmt.Errorf("failed to construct root app: %s", err)
	}
	rootApp.ObjectMeta.Annotations[common.ProfileAnnotationKey] = profileName
	err = DeployToGit(kubeClient, argocdNs, arlonNs, clusterName,
		repoUrl, repoBranch, basePath, profileName)
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
