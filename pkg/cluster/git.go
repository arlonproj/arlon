package cluster

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	gogit "github.com/go-git/go-git/v5"
	"github.com/platform9/arlon/pkg/argocd"
	"github.com/platform9/arlon/pkg/bundle"
	"github.com/platform9/arlon/pkg/gitutils"
	"github.com/platform9/arlon/pkg/log"
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
	bundles, err := bundle.GetBundlesFromProfile(prof, corev1, arlonNs)
	if err != nil {
		return fmt.Errorf("failed to get bundles: %s", err)
	}
	repo, tmpDir, auth, err := argocd.CloneRepo(kubeClient, argocdNs,
		repoUrl, repoBranch)
	if err != nil {
		return fmt.Errorf("failed to clone repo: %s", err)
	}
	clusterPath := clusterPathFromBasePath(basePath, clusterName)
	mgmtPath := mgmtPathFromClusterPath(clusterPath)
	workloadPath := workloadPathFromClusterPath(clusterPath)
	wt, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get repo worktree: %s", err)
	}
	// remove old data if directory exists, we'll regenerate everything
	fileInfo, err := wt.Filesystem.Lstat(clusterPath)
	if err == nil {
		if !fileInfo.IsDir() {
			return fmt.Errorf("unexpected file type for %s", clusterPath)
		}
		_, err = wt.Remove(clusterPath)
		if err != nil {
			return fmt.Errorf("failed to recursively delete cluster directory: %s", err)
		}
	}
	err = CopyManifests(wt, content, ".", mgmtPath)
	if err != nil {
		return fmt.Errorf("failed to copy embedded content: %s", err)
	}
	profRepoUrl := prof.Data["repo-url"]
	if profRepoUrl != "" {
		// dynamic profile: bundles not included in root app.
		// create an Application for the profile.
		profRepoPath := prof.Data["repo-path"]
		appPath := path.Join(mgmtPath, "templates", "profile.yaml")
		err = ProcessDynamicProfile(wt, clusterName, profileName, argocdNs,
			profRepoUrl, profRepoPath, appPath)
		if err != nil {
			return fmt.Errorf("failed to process dynamic profile: %s", err)
		}
	} else {
		// static profile: include bundles as individual Applications now
		err = ProcessBundles(wt, clusterName, repoUrl, mgmtPath, workloadPath, bundles)
		if err != nil {
			return fmt.Errorf("failed to process bundles: %s", err)
		}
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

// -----------------------------------------------------------------------------

func CopyManifests(wt *gogit.Worktree, fs embed.FS, root string, mgmtPath string) error {
	log := log.GetLogger()
	items, err := fs.ReadDir(root)
	if err != nil {
		return fmt.Errorf("failed to read embedded directory: %s", err)
	}
	for _, item := range items {
		filePath := path.Join(root, item.Name())
		if item.IsDir() {
			if err := CopyManifests(wt, fs, filePath, mgmtPath); err != nil {
				return err
			}
		} else {
			src, err := fs.Open(filePath)
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

const appTmpl = `
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: {{.AppName}}
  namespace: {{.AppNamespace}}
  finalizers:
  # This solves issue #17
  - resources-finalizer.argocd.argoproj.io/foreground
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
    targetRevision: {{.RepoRevision}}
    helm:
      parameters:
      # Pass cluster name to the bundle in case it needs it and is a Helm chart.
      # Example: this is required by the CAPI cluster autoscaler.
      # Use arlon prefix to avoid any conflicts with the bundle's own values.
      - name: arlon.clusterName
        value: {{.ClusterName}}
`

// This is used for a dynamic profile, which is an Application containing
// other Applications (one for each bundle), so the destination must always
// be the management cluster. Additionally, since the profile application
// is a Helm chart, clusterName is passed as a Helm parameter.
const dynProfTmpl = `
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: {{.AppName}}
  namespace: {{.AppNamespace}}
  finalizers:
  # This solves issue #17
  - resources-finalizer.argocd.argoproj.io/foreground
spec:
  syncPolicy:
    automated:
      prune: true
  destination:
    server: https://kubernetes.default.svc
    namespace: {{.DestinationNamespace}}
  project: default
  source:
    repoURL: {{.RepoUrl}}
    path: {{.RepoPath}}
    targetRevision: HEAD
    helm:
      parameters:
      - name: clusterName
        value: {{.ClusterName}}
      - name: profileAppName
        value: {{.AppName}}
`

type AppSettings struct {
	AppName              string
	ClusterName          string
	RepoUrl              string
	RepoPath             string
	RepoRevision         string
	AppNamespace         string
	DestinationNamespace string
}

// -----------------------------------------------------------------------------

func ProcessDynamicProfile(
	wt *gogit.Worktree,
	clusterName string,
	profileName string,
	argocdNs string,
	repoUrl string,
	repoPath string,
	appPath string,
) error {
	tmpl, err := template.New("app").Parse(dynProfTmpl)
	if err != nil {
		return fmt.Errorf("failed to create app template: %s", err)
	}
	mgmtPath := path.Join(repoPath, "mgmt")
	app := AppSettings{
		ClusterName:          clusterName,
		AppName:              fmt.Sprintf("%s-profile-%s", clusterName, profileName),
		AppNamespace:         argocdNs,
		DestinationNamespace: argocdNs,
		RepoUrl:              repoUrl,
		RepoPath:             mgmtPath,
	}
	dst, err := wt.Filesystem.Create(appPath)
	if err != nil {
		return fmt.Errorf("failed to create application file %s: %s", appPath, err)
	}
	err = tmpl.Execute(dst, &app)
	_ = dst.Close()
	if err != nil {
		return fmt.Errorf("failed to render application template %s: %s", appPath, err)
	}
	return nil
}

// -----------------------------------------------------------------------------

func ProcessBundles(
	wt *gogit.Worktree,
	clusterName string,
	repoUrl string,
	mgmtPath string,
	workloadPath string,
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
			ClusterName:          clusterName,
			AppName:              fmt.Sprintf("%s-%s", clusterName, b.Name),
			AppNamespace:         "argocd",
			DestinationNamespace: "default", // FIXME: make configurable
		}
		if b.RepoRevision == "" {
			app.RepoRevision = "HEAD"
		} else {
			app.RepoRevision = b.RepoRevision
		}
		if b.Data == nil {
			// dynamic bundle
			if b.RepoUrl == "" {
				return fmt.Errorf("b %s is neither static nor dynamic type", b.Name)
			}
			app.RepoUrl = b.RepoUrl
			app.RepoPath = b.RepoPath
		} else if b.RepoUrl != "" {
			return fmt.Errorf("b %s has both data and repoUrl set", b.Name)
		} else {
			// static bundle
			dirPath := path.Join(workloadPath, b.Name)
			err := wt.Filesystem.MkdirAll(dirPath, fs.ModeDir|0700)
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
				return fmt.Errorf("failed to copy static b %s: %s", b.Name, err)
			}
			app.RepoUrl = repoUrl
			app.RepoPath = path.Join(workloadPath, b.Name)
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

// -----------------------------------------------------------------------------
