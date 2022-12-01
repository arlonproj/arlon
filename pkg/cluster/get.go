package cluster

import (
	"context"
	"fmt"

	apppkg "github.com/argoproj/argo-cd/v2/pkg/apiclient/application"
	argoapp "github.com/argoproj/argo-cd/v2/pkg/apiclient/application"
	"github.com/arlonproj/arlon/pkg/common"
	logpkg "github.com/arlonproj/arlon/pkg/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
)

//------------------------------------------------------------------------------

func Get(
	appIf argoapp.ApplicationServiceClient,
	config *restclient.Config,
	argocdNs string,
	name string,
) (cl *Cluster, err error) {
	log := logpkg.GetLogger()
	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to get kube client: %s", err)
	}
	corev1 := kubeClient.CoreV1()
	secrApi := corev1.Secrets(argocdNs)
	secrs, err := secrApi.List(context.Background(), metav1.ListOptions{
		LabelSelector: argoClusterSecretTypeLabel + "," + externalClusterTypeLabel,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list cluster secrets: %s", err)
	}
	for _, secr := range secrs.Items {
		clusterName := secr.Data["name"]
		if clusterName == nil {
			log.V(1).Info("cluster secret skipped because missing cluster name",
				"secretName", secr.Name)
		}
		if string(clusterName) == name {
			return &Cluster{
				Name:        name,
				ProfileName: secr.Annotations[common.ProfileAnnotationKey],
				IsExternal:  true,
				SecretName:  secr.Name,
			}, nil
		}
	}
	app, err := appIf.Get(context.Background(),
		&apppkg.ApplicationQuery{
			Name: &name,
		})
	if err != nil {
		return nil, fmt.Errorf("failed to get argocd application: %s, %s", err, ErrArgocdToken)
	}
	typ := app.Labels["arlon-type"]
	if typ == "cluster" {
		return &Cluster{
			Name:            app.Name,
			ClusterSpecName: app.Annotations[common.ClusterSpecAnnotationKey],
			ProfileName:     app.Annotations[common.ProfileAnnotationKey],
		}, nil
	}
	if typ == "cluster-app" {
		profileName, err := getMatchingProfileName(appIf, app.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to matching profile name: %s", err)
		}
		return &Cluster{
			Name:        app.Name,
			ProfileName: profileName,
			BaseCluster: &BaseClusterInfo{
				Name:         app.Annotations[baseClusterNameAnnotation],
				RepoUrl:      app.Annotations[baseClusterRepoUrlAnnotation],
				RepoRevision: app.Annotations[baseClusterRepoRevisionAnnotation],
				RepoPath:     app.Annotations[baseClusterRepoPathAnnotation],
			},
		}, nil
	}
	return nil, fmt.Errorf("not an arlon cluster")
}

//------------------------------------------------------------------------------

func (c *Cluster) String() string {
	s := "Name: " + c.Name
	if c.IsExternal {
		s = s + ", Type: external"
	} else if c.BaseCluster != nil {
		s = s + ", Type: next-gen, Base Cluster Repo Url: " + c.BaseCluster.RepoUrl +
			", Base Cluster Repo Path: " + c.BaseCluster.RepoPath
	} else {
		s = s + ", Type: previous gen, Cluster Spec: " + c.ClusterSpecName
	}
	if c.ProfileName != "" {
		s = s + ", Profile: " + c.ProfileName
	}
	return s
}
