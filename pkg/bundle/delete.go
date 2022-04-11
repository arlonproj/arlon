package bundle

import (
	"context"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func Delete(
	kubeClient *kubernetes.Clientset,
	ns string,
	bundleName string,
) error {
	corev1 := kubeClient.CoreV1()
	secretsApi := corev1.Secrets(ns)
	err := secretsApi.Delete(context.Background(), bundleName, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete bundle: %s", err)
	}
	return nil
}
