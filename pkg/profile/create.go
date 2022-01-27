package profile

import (
	"context"
	"fmt"
	v1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func Create(
	kubeClient *kubernetes.Clientset,
	ns string, profileName string,
	repoUrl string,
	repoPath string,
	bundles string,
	desc string,
	tags string,
) error {
	corev1 := kubeClient.CoreV1()
	configMapApi := corev1.ConfigMaps(ns)
	_, err := configMapApi.Get(context.Background(), profileName, metav1.GetOptions{})
	if err == nil {
		return fmt.Errorf("a profile with that name already exists")
	}
	if !apierr.IsNotFound(err) {
		return fmt.Errorf("failed to check for existence of profile: %s", err)
	}
	cm := v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: profileName,
			Labels: map[string]string{
				"managed-by": "arlon",
				"arlon-type": "profile",
				"profile-type": "configuration",
			},
		},
		Data: map[string]string{
			"description": desc,
			"bundles": bundles,
			"tags": tags,
			"repo-url": repoUrl,
			"repo-path": repoPath,
		},
	}
	_, err = configMapApi.Create(context.Background(), &cm, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create profile configmap: %s", err)
	}
	return nil
}

