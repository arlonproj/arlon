package profile

import (
	"context"
	"fmt"
	v1 "github.com/arlonproj/arlon/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func DeleteLegacy(
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

func Delete(cli client.Client, ns, profileName string) error {
	ctx := context.Background()
	prof := v1.Profile{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      profileName,
			Namespace: ns,
		},
	}
	return cli.Delete(ctx, &prof, &client.DeleteOptions{})
}
