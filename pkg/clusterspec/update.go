package clusterspec

import (
	"context"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func Update(
	kubeClient *kubernetes.Clientset,
	arlonNs string,
	specName string,
	kubernetesVersion string,
	nodeType string,
	nodeCount int,
	masterNodeCount int,
	clusterAutoscalerEnabledPtr *bool,
	clusterAutoscalerMinNodes int,
	clusterAutoscalerMaxNodes int,
	desc string,
	tags string,
) (dirty bool, err error) {
	cs, err := Get(kubeClient, arlonNs, specName)
	if err != nil {
		return false, fmt.Errorf("failed to get clusterspec: %s", err)
	}
	if kubernetesVersion == "" {
		kubernetesVersion = cs.KubernetesVersion
	} else {
		dirty = true
	}
	if nodeType == "" {
		nodeType = cs.NodeType
	} else {
		dirty = true
	}
	if nodeCount == 0 {
		nodeCount = cs.NodeCount
	} else {
		dirty = true
	}
	if masterNodeCount == 0 {
		masterNodeCount = cs.MasterNodeCount
	} else {
		dirty = true
	}
	var clusterAutoscalerEnabled bool
	if clusterAutoscalerEnabledPtr == nil {
		clusterAutoscalerEnabled = cs.ClusterAutoscalerEnabled
	} else {
		clusterAutoscalerEnabled = *clusterAutoscalerEnabledPtr
		dirty = true
	}
	if clusterAutoscalerMinNodes == 0 {
		clusterAutoscalerMinNodes = cs.ClusterAutoscalerMinNodes
	} else {
		dirty = true
	}
	if clusterAutoscalerMaxNodes == 0 {
		clusterAutoscalerMaxNodes = cs.ClusterAutoscalerMaxNodes
	} else {
		dirty = true
	}
	if desc == "" {
		desc = cs.Description
	} else {
		dirty = true
	}
	if tags == "" {
		tags = cs.Tags
	} else {
		dirty = true
	}
	if !dirty {
		return
	}
	cm := ToConfigMap(specName, cs.ApiProvider, cs.CloudProvider, cs.Type,
		kubernetesVersion, nodeType, nodeCount, masterNodeCount,
		cs.Region, cs.PodCidrBlock, cs.SshKeyName, clusterAutoscalerEnabled,
		clusterAutoscalerMinNodes, clusterAutoscalerMaxNodes,
		tags, desc)
	corev1 := kubeClient.CoreV1()
	configMapApi := corev1.ConfigMaps(arlonNs)
	_, err = configMapApi.Update(context.Background(), cm, metav1.UpdateOptions{})
	if err != nil {
		return false, fmt.Errorf("failed to update clusterspec configmap: %s", err)
	}
	return
}
