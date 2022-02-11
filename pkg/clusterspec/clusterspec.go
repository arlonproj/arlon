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

const (
	ApiProviderKey = "apiProvider"
	CloudProviderKey = "cloudProvider"
	NodeTypeKey = "nodeType"
	ClusterTypeKey = "type"
	KubernetesVersionKey = "kubernetesVersion"
	NodeCountKey = "nodeCount"
	RegionKey = "region"
	PodCidrBlockKey = "podCidrBlock"
	SshKeyNameKey = "sshKeyName"
	TagsKey = "tags"
	DescriptionKey = "description"
)

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
		ApiProvider: cm.Data[ApiProviderKey],
		CloudProvider: cm.Data[CloudProviderKey],
		Type: cm.Data[ClusterTypeKey],
		KubernetesVersion: cm.Data[KubernetesVersionKey],
		NodeType: cm.Data[NodeTypeKey],
		Region: cm.Data[RegionKey],
		PodCidrBlock: cm.Data[PodCidrBlockKey],
		SshKeyName: cm.Data[SshKeyNameKey],
		Tags: cm.Data[TagsKey],
		Description: cm.Data[DescriptionKey],
	}
	var err error
	cs.NodeCount, err = strconv.Atoi(cm.Data[NodeCountKey])
	if err != nil {
		return nil, fmt.Errorf("could not parse clusterspec nodeCount: %s", err)
	}
	return cs, nil
}

func ToConfigMap(
	name string,
	apiProvider string,
	cloudProvider string,
	clusterType string,
	kubernetesVersion string,
	nodeType string,
	nodeCount int,
	region string,
	podCidrBlock string,
	sshKeyName string,
	tags string,
	description string,
) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"managed-by": "arlon",
				"arlon-type": "clusterspec",
			},
		},
		Data: map[string]string{
			ApiProviderKey: apiProvider,
			CloudProviderKey: cloudProvider,
			ClusterTypeKey: clusterType,
			KubernetesVersionKey: kubernetesVersion,
			NodeTypeKey: nodeType,
			NodeCountKey: strconv.Itoa(nodeCount),
			RegionKey: region,
			PodCidrBlockKey: podCidrBlock,
			SshKeyNameKey: sshKeyName,
			TagsKey: tags,
			DescriptionKey: description,
		},
	}
}
