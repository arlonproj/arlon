package profile

import (
	"context"
	"fmt"
	arlonv1 "github.com/arlonproj/arlon/api/v1"
	"github.com/arlonproj/arlon/pkg/controller"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"path"
)

func Create(
	config *restclient.Config,
	argocdNs string,
	arlonNs string,
	profileName string,
	repoUrl string,
	repoBasePath string,
	repoRevision string,
	bundles string,
	desc string,
	tags string,
	overrides []arlonv1.Override,
) error {
	cli, err := controller.NewClient(config)
	if err != nil {
		return fmt.Errorf("failed to get controller runtime client: %s", err)
	}

	var repoPath string
	if repoUrl == "" {
		repoRevision = ""
	} else {
		repoPath = path.Join(repoBasePath, profileName)
	}
	bundleList := StringListFromCommaSeparated(bundles)
	tagList := StringListFromCommaSeparated(tags)
	p := arlonv1.Profile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      profileName,
			Namespace: arlonNs,
		},
		Spec: arlonv1.ProfileSpec{
			Description:  desc,
			Bundles:      bundleList,
			Tags:         tagList,
			RepoUrl:      repoUrl,
			RepoPath:     repoPath,
			RepoRevision: repoRevision,
			Overrides:    overrides,
		},
	}
	if repoUrl != "" {
		kubeClient, err := kubernetes.NewForConfig(config)
		if err != nil {
			return fmt.Errorf("failed to get kubernetes client: %s", err)
		}
		err = createInGit(kubeClient, &p, argocdNs, arlonNs)
		if err != nil {
			return fmt.Errorf("failed to create dynamic profile in git: %s", err)
		}
	}
	err = cli.Create(context.Background(), &p)
	if err != nil {
		return fmt.Errorf("failed to create profile configmap: %s", err)
	}
	return nil
}
