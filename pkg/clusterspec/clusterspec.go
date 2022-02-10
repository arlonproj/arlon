package clusterspec

import (
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"strconv"
)

type ClusterSpec struct {
	Name string
	ApiProvider string
	CloudProvider string
	Type string
	KubernetesVersion string
	NodeType string
	NodeCount int
	Region string
	PodCidrBlock string
	SshKeyName string
	Tags string
	Description string
}

func Get(
	kubeClient *kubernetes.Clientset,
	arlonNs string,
	specName string,
) (cs *ClusterSpec, err error) {
	corev1 := kubeClient.CoreV1()
	configMapApi := corev1.ConfigMaps(arlonNs)
	cm, err := configMapApi.Get(context.Background(), specName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	cs, err = FromConfigMap(cm)
	if err != nil {
		return nil, err
	}
	return
}

func FromConfigMap(cm *corev1.ConfigMap) (*ClusterSpec, error) {
	cs := &ClusterSpec{
		Name: cm.Name,
		ApiProvider: cm.Data["apiProvider"],
		CloudProvider: cm.Data["cloudProvider"],
		Type: cm.Data["type"],
		KubernetesVersion: cm.Data["kubernetesVersion"],
		NodeType: cm.Data["nodeType"],
		Region: cm.Data["region"],
		PodCidrBlock: cm.Data["podCidrBlock"],
		SshKeyName: cm.Data["sshKeyName"],
		Tags: cm.Data["tags"],
		Description: cm.Data["description"],
	}
	var err error
	cs.NodeCount, err = strconv.Atoi(cm.Data["nodeCount"])
	if err != nil {
		return nil, fmt.Errorf("could not parse clusterspec nodeCount: %s", err)
	}
	return cs, nil
}
