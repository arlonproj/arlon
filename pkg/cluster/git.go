package cluster

import (
	"arlon.io/arlon/pkg/common"
	"arlon.io/arlon/pkg/gitutils"
	"arlon.io/arlon/pkg/log"
	"bytes"
	"context"
	"embed"
	"fmt"
	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"io"
	"io/fs"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	corev1types "k8s.io/client-go/kubernetes/typed/core/v1"
	"os"
	"path"
	"strings"
	"text/template"
)

//go:embed manifests/*
var content embed.FS

type RepoCreds struct {
	Url string
	Username string
	Password string
}

type bundle struct {
	name string
	data []byte
	// The following are only set on reference type bundles
	repoUrl string
	repoPath string
}

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
	secretsApi := corev1.Secrets(argocdNs)
	opts := metav1.ListOptions{
		LabelSelector: "argocd.argoproj.io/secret-type=repository",
	}
	secrets, err := secretsApi.List(context.Background(), opts)
	if err != nil {
		return fmt.Errorf("failed to list secrets: %s", err)
	}
	var creds *RepoCreds
	for _, repoSecret := range secrets.Items {
		if strings.Compare(repoUrl, string(repoSecret.Data["url"])) == 0 {
			creds = &RepoCreds{
				Url: string(repoSecret.Data["url"]),
				Username: string(repoSecret.Data["username"]),
				Password: string(repoSecret.Data["password"]),
			}
			break
		}
	}
	if creds == nil {
		return fmt.Errorf("did not find argocd repository matching %s (did you register it?)", repoUrl)
	}

	prof, err := getProfileConfigMap(profileName, corev1, arlonNs)
	if err != nil {
		return fmt.Errorf("failed to get profile: %s", err)
	}
	bundles, err := getBundles(prof, corev1, arlonNs)
	if err != nil {
		return fmt.Errorf("failed to get bundles: %s", err)
	}
	tmpDir, err := os.MkdirTemp("", "arlon-")
	branchRef := plumbing.NewBranchReferenceName(repoBranch)
	auth := &http.BasicAuth{
		Username: creds.Username,
		Password: creds.Password,
	}
	repo, err := gogit.PlainCloneContext(context.Background(), tmpDir, false, &gogit.CloneOptions{
		URL:           repoUrl,
		Auth:          auth,
		RemoteName:    gogit.DefaultRemoteName,
		ReferenceName: branchRef,
		SingleBranch:  true,
		NoCheckout: false,
		Progress:   nil,
		Tags:       gogit.NoTags,
		CABundle:   nil,
	})
	if err != nil {
		return fmt.Errorf("failed to clone repository: %s", err)
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
) (bundles []bundle, err error) {
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
		bundles = append(bundles, bundle{
			name: bundleName,
			data: secr.Data["data"],
			repoUrl: string(secr.Annotations[common.RepoUrlAnnotationKey]),
			repoPath: string(secr.Annotations[common.RepoPathAnnotationKey]),
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
	bundles []bundle,
) error {
	if len(bundles) == 0 {
		return nil
	}
	tmpl, err := template.New("app").Parse(appTmpl)
	if err != nil {
		return fmt.Errorf("failed to create app template: %s", err)
	}
	for _, bundle := range bundles {
		bundleFileName := fmt.Sprintf("%s.yaml", bundle.name)
		app := AppSettings{
			ClusterName: clusterName,
			AppName: fmt.Sprintf("%s-%s", clusterName, bundle.name),
			AppNamespace: "argocd",
			DestinationNamespace: "default", // FIXME: make configurable
		}
		if bundle.data == nil {
			// reference type bundle
			if bundle.repoUrl == "" {
				return fmt.Errorf("bundle %s is neither inline nor reference type", bundle.name)
			}
			app.RepoUrl = bundle.repoUrl
			app.RepoPath = bundle.repoPath
		} else if bundle.repoUrl != "" {
			return fmt.Errorf("bundle %s has both data and repoUrl set", bundle.name)
		} else {
			// inline bundle
			dirPath := path.Join(repoPath, bundle.name)
			err := wt.Filesystem.MkdirAll(dirPath, fs.ModeDir | 0700)
			if err != nil {
				return fmt.Errorf("failed to create directory in working tree: %s", err)
			}
			bundlePath := path.Join(dirPath, bundleFileName)
			dst, err := wt.Filesystem.Create(bundlePath)
			if err != nil {
				return fmt.Errorf("failed to create file in working tree: %s", err)
			}
			_, err = io.Copy(dst, bytes.NewReader(bundle.data))
			_ = dst.Close()
			if err != nil {
				return fmt.Errorf("failed to copy inline bundle %s: %s", bundle.name, err)
			}
			app.RepoUrl = repoUrl
			app.RepoPath = path.Join(repoPath, bundle.name)
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
