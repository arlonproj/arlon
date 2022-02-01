package profile

import (
	"arlon.io/arlon/pkg/argocd"
	"arlon.io/arlon/pkg/bundle"
	"arlon.io/arlon/pkg/cluster"
	"arlon.io/arlon/pkg/gitutils"
	"arlon.io/arlon/pkg/log"
	"embed"
	"fmt"
	gogit "github.com/go-git/go-git/v5"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"path"
)

//go:embed manifests/*
var content embed.FS

func createInGit(
	kubeClient *kubernetes.Clientset,
	profileCm *v1.ConfigMap,
	argocdNs string,
	arlonNs string,
	repoUrl string,
	repoPath string,
	repoBranch string,
) error {
	log := log.GetLogger()
	corev1 := kubeClient.CoreV1()
	bundles, err := bundle.GetBundlesFromProfile(profileCm, corev1, arlonNs)
	if err != nil {
		return fmt.Errorf("failed to get bundles: %s", err)
	}
	repo, tmpDir, auth, err := argocd.CloneRepo(kubeClient, argocdNs,
		repoUrl, repoBranch)
	if err != nil {
		return fmt.Errorf("failed to clone repo: %s", err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get repo worktree: %s", err)
	}
	// remove old data if directory exists, we'll regenerate everything
	fileInfo, err := wt.Filesystem.Lstat(repoPath)
	if err == nil {
		if !fileInfo.IsDir() {
			return fmt.Errorf("unexpected file type for %s", repoPath)
		}
		_, err = wt.Remove(repoPath)
		if err != nil {
			return fmt.Errorf("failed to recursively delete cluster directory: %s", err)
		}
	}
	mgmtPath := path.Join(repoPath, "mgmt")
	err = cluster.CopyManifests(wt, content, ".", mgmtPath)
	if err != nil {
		return fmt.Errorf("failed to copy embedded content: %s", err)
	}
	workloadPath := path.Join(repoPath, "workload")
	err = cluster.ProcessBundles(wt, "{{ .Values.clusterName }}", repoUrl,
		mgmtPath, workloadPath, bundles)
	if err != nil {
		return fmt.Errorf("failed to process bundles: %s", err)
	}
	changed, err := gitutils.CommitChanges(tmpDir, wt)
	if err != nil {
		return fmt.Errorf("failed to commit changes: %s", err)
	}
	if !changed {
		log.Info("no changed files, skipping commit & push")
		return nil
	}
	err = repo.Push(&gogit.PushOptions{
		RemoteName: gogit.DefaultRemoteName,
		Auth:       auth,
		Progress:   nil,
		CABundle:   nil,
	})
	if err != nil {
		return fmt.Errorf("failed to push to remote repository: %s", err)
	}
	log.V(1).Info("succesfully pushed working tree", "tmpDir", tmpDir)
	return nil
}