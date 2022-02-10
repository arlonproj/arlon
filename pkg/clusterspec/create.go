package clusterspec

import (
	"context"
	"fmt"
	v1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"strconv"
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
	desc string,
	tags string,
) error {
	if err := ValidApiProvider(apiProvider); err != nil {
		return err
	}
	if err := ValidCloudProviderAndClusterType(cloudProvider, clusterType); err != nil {
		return err
	}
	corev1 := kubeClient.CoreV1()
	configMapApi := corev1.ConfigMaps(arlonNs)
	_, err := configMapApi.Get(context.Background(), specName, metav1.GetOptions{})
	if err == nil {
		return fmt.Errorf("a profile with that name already exists")
	}
	if !apierr.IsNotFound(err) {
		return fmt.Errorf("failed to check for existence of profile: %s", err)
	}
	cm := v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: specName,
			Labels: map[string]string{
				"managed-by": "arlon",
				"arlon-type": "clusterspec",
			},
		},
		Data: map[string]string{
			"description": desc,
			"apiProvider": apiProvider,
			"tags": tags,
			"cloudProvider": cloudProvider,
			"type": clusterType,
			"kubernetesVersion": kubernetesVersion,
			"nodeType": nodeType,
			"nodeCount": strconv.Itoa(nodeCount),
		},
	}
	_, err = configMapApi.Create(context.Background(), &cm, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create clusterspec configmap: %s", err)
	}
	return nil
}

