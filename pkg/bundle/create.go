package bundle

import (
	"context"
	"fmt"
	"github.com/arlonproj/arlon/pkg/common"
	v1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"os"
)

func Create(
	kubeClient *kubernetes.Clientset,
	ns string, bundleName string,
	fromFile string,
	repoUrl string,
	repoPath string,
	repoRevision string,
	srcType string,
	desc string,
	tags string,
) error {
	corev1 := kubeClient.CoreV1()
	secretsApi := corev1.Secrets(ns)
	_, err := secretsApi.Get(context.Background(), bundleName, metav1.GetOptions{})
	if err == nil {
		return fmt.Errorf("a bundle with that name already exists")
	}
	if !apierr.IsNotFound(err) {
		return fmt.Errorf("failed to check for existence of bundle: %s", err)
	}
	secr := v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: bundleName,
			Labels: map[string]string{
				"managed-by": "arlon",
				"arlon-type": "config-bundle",
			},
			Annotations: map[string]string{},
		},
		Data: map[string][]byte{
			"description": []byte(desc),
			"tags":        []byte(tags),
		},
	}
	if fromFile != "" && repoUrl != "" {
		return fmt.Errorf("file and repo cannot both be specified")
	}
	if fromFile != "" {
		data, err := os.ReadFile(fromFile)
		if err != nil {
			return fmt.Errorf("failed to read file: %s", err)
		}
		secr.Labels["bundle-type"] = "static"
		secr.Data["data"] = data
	} else if repoUrl != "" {
		secr.Labels["bundle-type"] = "dynamic"
		secr.ObjectMeta.Annotations[common.RepoUrlAnnotationKey] = repoUrl
		secr.ObjectMeta.Annotations[common.RepoPathAnnotationKey] = repoPath
		secr.ObjectMeta.Annotations[common.RepoRevisionAnnotationKey] = repoRevision
		secr.ObjectMeta.Annotations[common.SrcTypeAnnotationKey] = srcType
	} else {
		return fmt.Errorf("the bundle must be created from a file or repo URL")
	}
	_, err = secretsApi.Create(context.Background(), &secr, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create secret: %s", err)
	}
	return nil
}
