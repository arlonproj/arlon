package clusterspec

import (
	"context"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func Delete(
	kubeClient *kubernetes.Clientset,
	ns string,
	clusterspecName string,
) error {
	corev1 := kubeClient.CoreV1()
	configMapApi := corev1.ConfigMaps(ns)
	err := configMapApi.Delete(context.Background(), clusterspecName, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete clusterspec: %s", err)
	}
	return nil
}
