package basecluster

import (
	"fmt"
	"github.com/arlonproj/arlon/pkg/argocd"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"os"
	"path"
)

// Validate verifies whether the resources in the specified file contain one and
// only one cluster, and that no resources have a namespace specified.
// If successful, the function returns the name of the cluster.
func Validate(fileName string) (clusterName string, err error) {
	bld := resource.NewLocalBuilder()
	opts := resource.FilenameOptions{
		Filenames: []string{fileName},
	}
	res := bld.Unstructured().FilenameParam(false, &opts).Do()
	infos, err := res.Infos()
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrBuilderFailedRun, err)
	}
	for _, info := range infos {
		gvk := info.Object.GetObjectKind().GroupVersionKind()
		if info.Namespace != "" {
			return "", fmt.Errorf("%w: resource: %s, kind: %s", ErrResourceHasNamespace, info.Name, gvk.Kind)
		}
		if gvk.Kind == "Cluster" {
			if clusterName != "" {
				return "", ErrMultipleClusters
			}
			clusterName = info.Name
		}
	}
	if clusterName == "" {
		return "", ErrNoClusterResource
	}
	return
}

// -----------------------------------------------------------------------------

func ValidateGitDir(
	config *restclient.Config,
	argocdNs string,
	repoUrl string,
	repoRevision string,
	repoPath string,
) (clusterName string, err error) {
	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return "", fmt.Errorf("failed to get kubernetes client: %s", err)
	}
	repo, tmpDir, _, err := argocd.CloneRepo(kubeClient, argocdNs,
		repoUrl, repoRevision)
	defer os.RemoveAll(tmpDir)
	if err != nil {
		return "", fmt.Errorf("failed to clone repo: %s", err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		return "", fmt.Errorf("failed to get repo worktree: %s", err)
	}
	fs := wt.Filesystem
	infos, err := fs.ReadDir(repoPath)
	if err != nil {
		return "", fmt.Errorf("failed to list repo directory: %s", err)
	}
	dirPath := path.Join(tmpDir, repoPath)
	return validateDir(dirPath, infos)
}

// -----------------------------------------------------------------------------

// Given a list of file entries from a directory, validates whether
// conditions are met for using the directory as a base cluster.
func validateDir(dirPath string, infos []os.FileInfo) (clusterName string, err error) {
	var kustomizationFound bool
	var configurationsFound bool
	var manifestFile string
	for _, info := range infos {
		if info.IsDir() {
			return "", fmt.Errorf("found subdirectory: %s", info.Name())
		}
		if info.Name() == "kustomization.yaml" {
			kustomizationFound = true
			continue
		}
		if info.Name() == "configurations.yaml" {
			configurationsFound = true
			continue
		}
		if manifestFile != "" {
			return "", ErrMultipleManifests
		}
		manifestFile = info.Name()
	}
	if manifestFile == "" {
		return "", ErrNoManifest
	}
	if !kustomizationFound {
		return "", ErrNoKustomizationYaml
	}
	if !configurationsFound {
		return "", ErrNoConfigurationsYaml
	}
	manifestPath := path.Join(dirPath, manifestFile)
	return Validate(manifestPath)
}
