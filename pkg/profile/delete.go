package profile

import (
	"context"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func Delete(
	kubeClient *kubernetes.Clientset,
	ns string,
	profileName string,
) error {
	corev1 := kubeClient.CoreV1()
	configMapApi := corev1.ConfigMaps(ns)
	err := configMapApi.Delete(context.Background(), profileName, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete profile: %s", err)
	}
	return nil
}
