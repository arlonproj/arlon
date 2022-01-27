package cluster

import (
	"arlon.io/arlon/pkg/argocd"
	"arlon.io/arlon/pkg/bundle"
	"arlon.io/arlon/pkg/common"
	"arlon.io/arlon/pkg/gitutils"
	"arlon.io/arlon/pkg/log"
	"bytes"
	"context"
	"embed"
	"fmt"
	gogit "github.com/go-git/go-git/v5"
	"io"
	"io/fs"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	corev1types "k8s.io/client-go/kubernetes/typed/core/v1"
	"path"
	"strings"
	"text/template"
)

//go:embed manifests/*
var content embed.FS

// -----------------------------------------------------------------------------

func DeployToGit(
	kubeClient *kubernetes.Clientset,
	argocdNs string,
	arlonNs string,
	clusterName string,
	repoUrl string,
	repoBranch string,
	basePath string,
	profileName string,
) error {
	log := log.GetLogger()
	corev1 := kubeClient.CoreV1()
	prof, err := getProfileConfigMap(profileName, corev1, arlonNs)
	if err != nil {
		return fmt.Errorf("failed to get profile: %s", err)
	}
	bundles, err := getBundles(prof, corev1, arlonNs)
	if err != nil {
		return fmt.Errorf("failed to get bundles: %s", err)
	}
	repo, tmpDir, auth, err := argocd.CloneRepo(kubeClient, argocdNs,
		repoUrl, repoBranch)
	if err != nil {
		return fmt.Errorf("failed to clone repo: %s", err)
	}
	mgmtPath := path.Join(basePath, clusterName, "mgmt")
	repoPath := path.Join(basePath, clusterName, "workload")
	wt, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get repo worktree: %s", err)
	}
	err = copyManifests(wt, ".", mgmtPath)
	if err != nil {
		return fmt.Errorf("failed to copy embedded content: %s", err)
	}
	err = processBundles(wt, clusterName, repoUrl, mgmtPath, repoPath, bundles)
	if err != nil {
		return fmt.Errorf("failed to copy inline bundles: %s", err)
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
	log.Info("succesfully pushed working tree", "tmpDir", tmpDir)
	return nil
}

// -----------------------------------------------------------------------------

func copyManifests(wt *gogit.Worktree, root string, mgmtPath string) error {
	log := log.GetLogger()
	items, err := content.ReadDir(root)
	if err != nil {
		return fmt.Errorf("failed to read embedded directory: %s", err)
	}
	for _, item := range items {
		filePath := path.Join(root, item.Name())
		if item.IsDir() {
			if err := copyManifests(wt, filePath, mgmtPath); err != nil {
				return err
			}
		} else {
			src, err := content.Open(filePath)
			if err != nil {
				return fmt.Errorf("failed to open embedded file %s: %s", filePath, err)
			}
			// remove manifests/ prefix
			components := strings.Split(filePath, "/")
			dstPath := path.Join(components[1:]...)
			dstPath = path.Join(mgmtPath, dstPath)
			dst, err := wt.Filesystem.Create(dstPath)
			if err != nil {
				_ = src.Close()
				return fmt.Errorf("failed to create destination file %s: %s", dstPath, err)
			}
			_, err = io.Copy(dst, src)
			_ = src.Close()
			_ = dst.Close()
			if err != nil {
				return fmt.Errorf("failed to copy embedded file: %s", err)
			}
			log.V(1).Info("copied embedded file", "destination", dstPath)
		}
	}
	return nil
}

// -----------------------------------------------------------------------------

func getProfileConfigMap(
	profileName string,
	corev1 corev1types.CoreV1Interface,
	arlonNs string,
) (prof *v1.ConfigMap, err error) {
	if profileName == "" {
		return nil, fmt.Errorf("profile name not specified")
	}
	configMapsApi := corev1.ConfigMaps(arlonNs)
	profileConfigMap, err := configMapsApi.Get(context.Background(), profileName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get profile configmap: %s", err)
	}
	if profileConfigMap.Labels["arlon-type"] != "profile" {
		return nil, fmt.Errorf("profile configmap does not have expected label")
	}
	return profileConfigMap, nil
}

// -----------------------------------------------------------------------------

func getBundles(
	profileConfigMap *v1.ConfigMap,
	corev1 corev1types.CoreV1Interface,
	arlonNs string,
) (bundles []bundle.Bundle, err error) {
	secretsApi := corev1.Secrets(arlonNs)
	log := log.GetLogger()
	bundleList := profileConfigMap.Data["bundles"]
	if bundleList == "" {
		return nil, fmt.Errorf("profile has no bundles")
	}
	bundleItems := strings.Split(bundleList, ",")
	for _, bundleName := range bundleItems {
		secr, err := secretsApi.Get(context.Background(), bundleName, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to get bundle secret %s: %s", bundleName, err)
		}
		bundles = append(bundles, bundle.Bundle{
			Name: bundleName,
			Data: secr.Data["data"],
			RepoUrl: string(secr.Annotations[common.RepoUrlAnnotationKey]),
			RepoPath: string(secr.Annotations[common.RepoPathAnnotationKey]),
		})
		log.V(1).Info("adding bundle", "bundleName", bundleName)
	}
	return
}

// -----------------------------------------------------------------------------

const appTmpl = `
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: {{.AppName}}
  namespace: {{.AppNamespace}}
spec:
  syncPolicy:
    automated:
      prune: true
  destination:
    name: {{.ClusterName}}
    namespace: {{.DestinationNamespace}}
  project: default
  source:
    repoURL: {{.RepoUrl}}
    path: {{.RepoPath}}
    targetRevision: HEAD
`

type AppSettings struct {
	AppName string
	ClusterName string
	RepoUrl string
	RepoPath string
	AppNamespace string
	DestinationNamespace string
}

func processBundles(
	wt *gogit.Worktree,
	clusterName string,
	repoUrl string,
	mgmtPath string,
	repoPath string,
	bundles []bundle.Bundle,
) error {
	if len(bundles) == 0 {
		return nil
	}
	tmpl, err := template.New("app").Parse(appTmpl)
	if err != nil {
		return fmt.Errorf("failed to create app template: %s", err)
	}
	for _, b := range bundles {
		bundleFileName := fmt.Sprintf("%s.yaml", b.Name)
		app := AppSettings{
			ClusterName: clusterName,
			AppName: fmt.Sprintf("%s-%s", clusterName, b.Name),
			AppNamespace: "argocd",
			DestinationNamespace: "default", // FIXME: make configurable
		}
		if b.Data == nil {
			// reference type b
			if b.RepoUrl == "" {
				return fmt.Errorf("b %s is neither inline nor reference type", b.Name)
			}
			app.RepoUrl = b.RepoUrl
			app.RepoPath = b.RepoPath
		} else if b.RepoUrl != "" {
			return fmt.Errorf("b %s has both data and repoUrl set", b.Name)
		} else {
			// inline bundle
			dirPath := path.Join(repoPath, b.Name)
			err := wt.Filesystem.MkdirAll(dirPath, fs.ModeDir | 0700)
			if err != nil {
				return fmt.Errorf("failed to create directory in working tree: %s", err)
			}
			bundlePath := path.Join(dirPath, bundleFileName)
			dst, err := wt.Filesystem.Create(bundlePath)
			if err != nil {
				return fmt.Errorf("failed to create file in working tree: %s", err)
			}
			_, err = io.Copy(dst, bytes.NewReader(b.Data))
			_ = dst.Close()
			if err != nil {
				return fmt.Errorf("failed to copy inline b %s: %s", b.Name, err)
			}
			app.RepoUrl = repoUrl
			app.RepoPath = path.Join(repoPath, b.Name)
		}
		appPath := path.Join(mgmtPath, "templates", bundleFileName)
		dst, err := wt.Filesystem.Create(appPath)
		if err != nil {
			return fmt.Errorf("failed to create application file %s: %s", appPath, err)
		}
		err = tmpl.Execute(dst, &app)
		if err != nil {
			dst.Close()
			return fmt.Errorf("failed to render application template %s: %s", appPath, err)
		}
		dst.Close()
	}
	return nil
}
