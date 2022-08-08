package bundle

import (
	"context"
	"errors"
	"fmt"
	"github.com/arlonproj/arlon/pkg/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
)

type ListItem struct {
	Name        string `json:"name,omitempty"`
	Type        string `json:"type,omitempty"`
	Tags        string `json:"tags,omitempty"`
	Repo        string `json:"repo,omitempty"`
	Path        string `json:"path,omitempty"`
	Revision    string `json:"revision,omitempty"`
	SrcType     string `json:"src_type,omitempty"`
	Description string `json:"description,omitempty"`
}

var (
	ErrFailedList = errors.New("failed to list secrets")
	ErrNoBundles  = errors.New("no bundles found")
)

func List(config *restclient.Config, namespace string) ([]ListItem, error) {
	kubeClient := kubernetes.NewForConfigOrDie(config)
	corev1 := kubeClient.CoreV1()
	secretsApi := corev1.Secrets(namespace)
	opts := metav1.ListOptions{
		LabelSelector: "managed-by=arlon,arlon-type=config-bundle",
	}
	secrets, err := secretsApi.List(context.Background(), opts)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrFailedList, err)
	}
	if len(secrets.Items) == 0 {
		return nil, ErrNoBundles
	}
	var bundles = make([]ListItem, 0)
	for _, secret := range secrets.Items {
		labels := secret.GetLabels()
		bundleType := labels["bundle-type"]
		if bundleType == "" {
			bundleType = "(undefined)"
		}
		repoUrl := secret.Annotations[common.RepoUrlAnnotationKey]
		repoPath := secret.Annotations[common.RepoPathAnnotationKey]
		srcType := secret.Annotations[common.SrcTypeAnnotationKey]
		if bundleType != "dynamic" {
			repoUrl = "(N/A)"
			repoPath = "(N/A)"
		}
		bundle := ListItem{
			Name:        secret.Name,
			Type:        bundleType,
			Tags:        string(secret.Data["tags"]),
			Repo:        repoUrl,
			Path:        repoPath,
			Revision:    secret.Annotations[common.RepoRevisionAnnotationKey],
			SrcType:     srcType,
			Description: string(secret.Data["description"]),
		}
		bundles = append(bundles, bundle)
	}
	return bundles, nil
}
