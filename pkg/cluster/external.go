package cluster

import (
	"context"
	"fmt"
	argoclient "github.com/argoproj/argo-cd/v2/pkg/apiclient"
	argoapp "github.com/argoproj/argo-cd/v2/pkg/apiclient/application"
	argocluster "github.com/argoproj/argo-cd/v2/pkg/apiclient/cluster"
	"github.com/argoproj/argo-cd/v2/pkg/apis/application"
	argoappv1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	arlonv1 "github.com/arlonproj/arlon/api/v1"
	"github.com/arlonproj/arlon/pkg/common"
	"github.com/arlonproj/arlon/pkg/controller"
	logpkg "github.com/arlonproj/arlon/pkg/log"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"path"
	patchclient "sigs.k8s.io/controller-runtime/pkg/client"
)

//------------------------------------------------------------------------------

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
	cli, err := controller.NewClient(config)
	if err != nil {
		return fmt.Errorf("failed to get controller runtime client: %s", err)
	}
	profileAppName := fmt.Sprintf("%s-profile-%s", clusterName, prof.Name)
	app := constructProfileApp(profileAppName, argocdNs, clusterName, prof)
	_, err = appIf.Create(context.Background(), &argoapp.ApplicationCreateRequest{
		Application:          *app,
	})
	if err != nil {
		return fmt.Errorf("failed to get create profile app: %s", err)
	}
	newSecr := secrPtr.DeepCopy()
	newSecr.Labels[clusterTypeLabelKey] = "external"
	newSecr.Annotations[common.ProfileAnnotationKey] = prof.Name
	newSecr.Annotations[common.ProfileAppAnnotationKey] = profileAppName
	err = cli.Patch(context.Background(), newSecr, patchclient.MergeFrom(secrPtr))
	if err != nil {
		cascade := true
		_,err = appIf.Delete(context.Background(), &argoapp.ApplicationDeleteRequest{
			Name: &profileAppName, Cascade: &cascade,
		})
		if err != nil {
			log.Error(err, "failed to delete profile app after failed secret patch")
		}
		return fmt.Errorf("failed to patch secret: %s", err)
	}
	return nil
}

//------------------------------------------------------------------------------

func constructProfileApp(
	appName string,
	argocdNs string,
	clusterName string,
	prof *arlonv1.Profile,
) *argoappv1.Application {
	repoPath := path.Join(prof.Spec.RepoPath, "mgmt")
	return &argoappv1.Application{
		TypeMeta: metav1.TypeMeta{
			Kind:       application.ApplicationKind,
			APIVersion: application.Group + "/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      appName,
			Namespace: argocdNs,
			Labels:    map[string]string{"managed-by": "arlon", "arlon-type": "profile-app"},
			Annotations: map[string]string{
				common.ProfileAnnotationKey: prof.Name,
			},
			Finalizers: []string{argoappv1.ForegroundPropagationPolicyFinalizer},
		},
		Spec: argoappv1.ApplicationSpec{
			SyncPolicy: &argoappv1.SyncPolicy{
				Automated: &argoappv1.SyncPolicyAutomated{
					Prune: true,
				},
			},
			Destination: argoappv1.ApplicationDestination{
				Server: "https://kubernetes.default.svc",
				Namespace: argocdNs,
			},
			Source: argoappv1.ApplicationSource{
				RepoURL: prof.Spec.RepoUrl,
				Path: repoPath,
				TargetRevision: prof.Spec.RepoRevision,
				Helm: &argoappv1.ApplicationSourceHelm{
					Parameters: []argoappv1.HelmParameter{
						{
							Name: "clusterName", Value: clusterName,
						},
						{
							Name: "profileAppName", Value: appName,
						},
					},
				},
			},
		},
	}
}

//------------------------------------------------------------------------------

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
	foundCluster, err := Get(appIf, config, argocdNs, clusterName)
	if err != nil {
		return fmt.Errorf("failed to get existing cluster: %s", err)
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
	cli, err := controller.NewClient(config)
	if err != nil {
		return fmt.Errorf("failed to get controller runtime client: %s", err)
	}

	// First delete the associated profile app
	profileAppName := secr.Annotations[common.ProfileAppAnnotationKey]
	if profileAppName == "" {
		return fmt.Errorf("secret does not contain profile app name annotation")
	}
	cascade := true
	_,err = appIf.Delete(context.Background(), &argoapp.ApplicationDeleteRequest{
		Name: &profileAppName, Cascade: &cascade,
	})
	if err != nil {
		return fmt.Errorf("failed to delete profile app: %s", err)
	}

	// Patch the secret to remove the arlon annotations and label
	newSecr := secr.DeepCopy()
	delete(newSecr.Labels, clusterTypeLabelKey)
	delete(newSecr.Annotations, common.ProfileAnnotationKey)
	delete(newSecr.Annotations, common.ProfileAppAnnotationKey)
	err = cli.Patch(context.Background(), newSecr, patchclient.MergeFrom(secr))
	if err != nil {
		return fmt.Errorf("failed to patch secret: %s", err)
	}
	return nil
}
