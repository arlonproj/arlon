package profile

import (
	"context"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func Update(
	kubeClient *kubernetes.Clientset,
	argocdNs string,
	arlonNs string,
	profileName string,
	bundles string,
	desc string,
	tags string,
) (dirty bool, err error) {
	corev1 := kubeClient.CoreV1()
	configMapApi := corev1.ConfigMaps(arlonNs)
	cm, err := configMapApi.Get(context.Background(), profileName, metav1.GetOptions{})
	if err != nil {
		return false, fmt.Errorf("failed to lookup profile: %s", err)
	}
	if desc != "" && desc != cm.Data["description"] {
		cm.Data["description"] = desc
		dirty = true
	}
	if tags != "" && tags != cm.Data["tags"] {
		cm.Data["tags"] = tags
		dirty = true
	}
	if bundles != "" && bundles != cm.Data["bundles"] {
		cm.Data["bundles"] = bundles
		repoUrl := cm.Data["repo-url"]
		dirty = true
		if repoUrl != "" {
			err = createInGit(kubeClient, cm, argocdNs, arlonNs,
				repoUrl, cm.Data["repo-path"], cm.Data["repo-branch"])
			if err != nil {
				return false, fmt.Errorf("failed to update dynamic profile in git: %s", err)
			}
		}
	}
	if !dirty {
		return
	}
	_, err = configMapApi.Update(context.Background(), cm, metav1.UpdateOptions{})
	if err != nil {
		return false, fmt.Errorf("failed to update profile configmap: %s", err)
	}
	return
}

