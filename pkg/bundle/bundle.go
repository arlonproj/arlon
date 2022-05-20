package bundle

import (
	"context"
	"fmt"
	"github.com/arlonproj/arlon/pkg/common"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1types "k8s.io/client-go/kubernetes/typed/core/v1"
	"strings"
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
	profileConfigMap *v1.ConfigMap,
	corev1 corev1types.CoreV1Interface,
	arlonNs string,
) (bundles []Bundle, err error) {
	secretsApi := corev1.Secrets(arlonNs)
	bundleList := profileConfigMap.Data["bundles"]
	if bundleList == "" {
		return nil, nil
	}
	bundleItems := strings.Split(bundleList, ",")
	for _, bundleName := range bundleItems {
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
