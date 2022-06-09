package bundle

import (
	"context"
	"fmt"
	arlonv1 "github.com/arlonproj/arlon/api/v1"
	"github.com/arlonproj/arlon/pkg/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1types "k8s.io/client-go/kubernetes/typed/core/v1"
)

type Bundle struct {
	Name string
	Data []byte
	// The following are only set on dynamic bundles
	RepoUrl      string
	RepoPath     string
	RepoRevision string
	SrcType      string
}

// -----------------------------------------------------------------------------

func GetBundlesFromProfile(
	profile *arlonv1.Profile,
	corev1 corev1types.CoreV1Interface,
	arlonNs string,
) (bundles []Bundle, err error) {
	secretsApi := corev1.Secrets(arlonNs)
	bundleList := profile.Spec.Bundles
	if bundleList == nil {
		return nil, nil
	}
	for _, bundleName := range bundleList {
		secr, err := secretsApi.Get(context.Background(), bundleName, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to get bundle secret %s: %s", bundleName, err)
		}
		bundles = append(bundles, Bundle{
			Name:         bundleName,
			Data:         secr.Data["data"],
			RepoUrl:      secr.Annotations[common.RepoUrlAnnotationKey],
			RepoPath:     secr.Annotations[common.RepoPathAnnotationKey],
			RepoRevision: secr.Annotations[common.RepoRevisionAnnotationKey],
			SrcType:      secr.Annotations[common.SrcTypeAnnotationKey],
		})
	}
	return
}
