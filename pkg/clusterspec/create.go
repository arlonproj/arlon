package clusterspec

import (
	"context"
	"fmt"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func Create(
	kubeClient *kubernetes.Clientset,
	arlonNs string,
	specName string,
	apiProvider string,
	cloudProvider string,
	clusterType string,
	kubernetesVersion string,
	nodeType string,
	nodeCount int,
	masterNodeCount int,
	sshKeyName string,
	clusterAutoscalerEnabled bool,
	clusterAutoscalerMinNodes int,
	clusterAutoscalerMaxNodes int,
	desc string,
	tags string,
) error {
	if err := ValidApiProvider(apiProvider); err != nil {
		return err
	}
	if err := ValidCloudProviderAndClusterType(cloudProvider, clusterType); err != nil {
		return err
	}
	_, err := Get(kubeClient, arlonNs, specName)
	if err == nil {
		return fmt.Errorf("a clusterspec with that name already exists")
	}
	if !apierr.IsNotFound(err) {
		return fmt.Errorf("failed to check for existence of clusterspec: %s", err)
	}
	cm := ToConfigMap(specName, apiProvider, cloudProvider, clusterType,
		kubernetesVersion, nodeType, nodeCount, masterNodeCount,
		"", "", sshKeyName, clusterAutoscalerEnabled,
		clusterAutoscalerMinNodes, clusterAutoscalerMaxNodes,
		tags, desc)
	corev1 := kubeClient.CoreV1()
	configMapApi := corev1.ConfigMaps(arlonNs)
	_, err = configMapApi.Create(context.Background(), cm, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create clusterspec configmap: %s", err)
	}
	return nil
}
