package bundle

import (
	"context"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"os"
)

func Update(
	kubeClient *kubernetes.Clientset,
	ns string, bundleName string,
	fromFile string,
	repoUrl string,
	repoPath string,
	desc string,
	tags string,
) error {
	if !IsValidK8sName(bundleName) {
		return fmt.Errorf("%w: %s", ErrInvalidName, bundleName)
	}
	corev1 := kubeClient.CoreV1()
	secretsApi := corev1.Secrets(ns)
	secr, err := secretsApi.Get(context.Background(), bundleName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get bundle secret: %s", err)
	}
	var dirty bool
	if desc != "" && desc != string(secr.Data["description"]) {
		secr.Data["description"] = []byte(desc)
		dirty = true
	}
	if tags != "" && tags != string(secr.Data["tags"]) {
		secr.Data["tags"] = []byte(tags)
		dirty = true
	}
	if fromFile != "" && repoUrl != "" {
		return fmt.Errorf("file and repo cannot both be specified")
	}
	if fromFile != "" {
		if secr.Labels["bundle-type"] != "static" {
			return fmt.Errorf("manifest content can only be changed if bundle is static")
		}
		data, err := os.ReadFile(fromFile)
		if err != nil {
			return fmt.Errorf("failed to read file: %s", err)
		}
		secr.Data["data"] = data
		dirty = true
	} else if repoUrl != "" || repoPath != "" {
		if secr.Labels["bundle-type"] == "dynamic" {
			return fmt.Errorf("cannot update git reference of a dynamic bundle")
		}
		return fmt.Errorf("cannot specify repo URL or path for an existing static bundle")
	}
	if !dirty {
		return nil
	}
	_, err = secretsApi.Update(context.Background(), secr, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update secret: %s", err)
	}
	return nil
}
