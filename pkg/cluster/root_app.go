package cluster

import (
	"fmt"
	"github.com/argoproj/argo-cd/v2/pkg/apis/application"
	argoappv1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/arlonproj/arlon/pkg/clusterspec"
	"github.com/arlonproj/arlon/pkg/common"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ConstructRootApp(
	argocdNs string,
	clusterName string,
	innerClusterName string, // only used for gen2, ok to leave empty
	repoUrl string, // target repo for gen1, source repo for gen2
	repoBranch string, // target revision for gen1, source revision for gen2
	repoPath string, // target path for gen1, source path for gen2
	clusterSpecName string, // empty for gen2
	clusterSpecCm *corev1.ConfigMap, // nil for gen2
	profileName string,
	managementClusterUrl string,
) (*argoappv1.Application, error) {
	appName := clusterName // gen1 default
	arlonType := "cluster" // gen1 default
	if clusterSpecName == "" {
		// gen2
		appName = fmt.Sprintf("%s-arlon", appName)
		arlonType = "arlon-app"
	}
	app := &argoappv1.Application{
		TypeMeta: v1.TypeMeta{
			Kind:       application.ApplicationKind,
			APIVersion: application.Group + "/v1alpha1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      appName,
			Namespace: argocdNs,
			Labels:    map[string]string{"managed-by": "arlon", "arlon-type": arlonType},
			Annotations: map[string]string{
				common.ClusterSpecAnnotationKey: clusterSpecName,
				common.ProfileAnnotationKey:     profileName,
			},
			Finalizers: []string{argoappv1.ForegroundPropagationPolicyFinalizer},
		},
	}
	var apiProvider string
	var cs *clusterspec.ClusterSpec
	if clusterSpecCm != nil {
		// gen1
		var err error
		cs, err = clusterspec.FromConfigMap(clusterSpecCm)
		if err != nil {
			return nil, fmt.Errorf("failed to read clusterspec from configmap: %s", err)
		}
		apiProvider = cs.ApiProvider
	} else {
		// gen2
		app.ObjectMeta.Labels["arlon-cluster"] = clusterName
		// assume CAPI for now
		apiProvider = "capi"
	}
	helmParams := []argoappv1.HelmParameter{
		{
			Name:  "global.clusterName",
			Value: clusterName,
		},
		{
			Name:  "global.kubeconfigSecretKeyName",
			Value: clusterspec.KubeconfigSecretKeyNameByApiProvider[apiProvider],
		},
		{
			Name:  "global.managementClusterUrl",
			Value: managementClusterUrl,
		},
	}
	if innerClusterName != "" {
		innerClusterNameWithDashSuffix := innerClusterName + "-"
		helmParams = append(helmParams, argoappv1.HelmParameter{
			Name:  "global.innerClusterNameWithDashSuffix",
			Value: innerClusterNameWithDashSuffix,
		})
	}
	if clusterSpecCm != nil {
		// gen1
		for _, key := range clusterspec.ValidHelmParamKeys {
			val := clusterSpecCm.Data[key]
			if val != "" {
				helmParams = append(helmParams, argoappv1.HelmParameter{
					Name:  fmt.Sprintf("global.%s", key),
					Value: val,
				})
			}
		}
		subchartName, err := clusterspec.SubchartNameFromClusterSpec(cs)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve subchart name: %s", err)
		}
		helmParams = append(helmParams, argoappv1.HelmParameter{
			Name:  fmt.Sprintf("tags.%s", subchartName),
			Value: "true",
		})
		var ignoreDiffs []argoappv1.ResourceIgnoreDifferences
		if cs.ClusterAutoscalerEnabled {
			casSubchartName, err := clusterspec.ClusterAutoscalerSubchartNameFromClusterSpec(cs)
			if err != nil {
				return nil, fmt.Errorf("failed to get cluster autoscaler subchart name: %s", err)
			}
			helmParams = append(helmParams, argoappv1.HelmParameter{
				Name:  fmt.Sprintf("tags.%s", casSubchartName),
				Value: "true",
			})
			// Cluster autoscaler will change replicas so ignore it
			ignoreDiffs = append(ignoreDiffs, argoappv1.ResourceIgnoreDifferences{
				Group:        "cluster.x-k8s.io",
				Kind:         "MachineDeployment",
				JSONPointers: []string{"/spec/replicas"},
			})
		}
		// Ignore CAPI EKS control plane's spec.version because the AWS controller(s)
		// appear to update it with a value that is less precise than the requested
		// one, for e.g. the spec might specify v1.18.16, and get updated with v1.18,
		// causing ArgoCD to report the resource as OutOfSync
		ignoreDiffs = append(ignoreDiffs, argoappv1.ResourceIgnoreDifferences{
			Group:        "controlplane.cluster.x-k8s.io",
			Kind:         "AWSManagedControlPlane",
			JSONPointers: []string{"/spec/version"},
		})
		app.Spec.IgnoreDifferences = ignoreDiffs
	}
	app.Spec.Source.Helm = &argoappv1.ApplicationSourceHelm{Parameters: helmParams}
	app.Spec.Source.RepoURL = repoUrl
	app.Spec.Source.TargetRevision = repoBranch
	app.Spec.Source.Path = repoPath
	app.Spec.Destination.Server = "https://kubernetes.default.svc"
	app.Spec.Destination.Namespace = "default"
	app.Spec.SyncPolicy = &argoappv1.SyncPolicy{
		Automated: &argoappv1.SyncPolicyAutomated{
			Prune: true,
		},
		SyncOptions: []string{"Prune=true"},
	}
	return app, nil
}
