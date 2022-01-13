package cluster

import (
	"arlon.io/arlon/pkg/gitutils"
	"arlon.io/arlon/pkg/log"
	"context"
	"embed"
	"fmt"
	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"io"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"os"
	"path"
	"strings"
)

//go:embed manifests/*
var content embed.FS

type RepoCreds struct {
	Url string
	Username string
	Password string
}

func Deploy(
	config *restclient.Config,
	argocdNs string,
	arlonNs string,
	clusterName string,
	repoUrl string,
	repoBranch string,
	basePath string,
	clusterSpecName string,
	profileName string,
) error {
	log := log.GetLogger()
	kubeClient := kubernetes.NewForConfigOrDie(config)
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
		return fmt.Errorf("did not find secret matching repo url: %s", repoUrl)
	}

	//var bundleData [][]byte
	if profileName != "" {
		configMapsApi := corev1.ConfigMaps(arlonNs)
		profileConfigMap, err := configMapsApi.Get(context.Background(), profileName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("failed to get profile configmap: %s", err)
		}
		if profileConfigMap.Labels["arlon-type"] != "profile" {
			return fmt.Errorf("profile configmap does not have expected label")
		}
		bundles := profileConfigMap.Data["bundles"]
		if bundles == "" {
			return fmt.Errorf("profile has no bundles")
		}
		bundleItems := strings.Split(bundles, ",")
		secretsApi = corev1.Secrets(arlonNs)
		for _, bundleName := range bundleItems {
			_, err := secretsApi.Get(context.Background(), bundleName, metav1.GetOptions{})
			if err != nil {
				return fmt.Errorf("failed to get bundle secret %s: %s", bundleName, err)
			}
		}
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
	wt, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get repo worktree: %s", err)
	}
	err = copyContent(wt, ".", mgmtPath)
	if err != nil {
		return fmt.Errorf("failed to copy embedded content: %s", err)
	}
	changed, err := gitutils.CommitChanges(tmpDir, wt)
	if err != nil {
		return fmt.Errorf("failed to commit changes: %s", err)
	}
	if !changed {
		log.Info("no changed files, skipping commit")
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

func copyContent(wt *gogit.Worktree, root string, mgmtPath string) error {
	log := log.GetLogger()
	items, err := content.ReadDir(root)
	if err != nil {
		return fmt.Errorf("failed to read embedded directory: %s", err)
	}
	for _, item := range items {
		filePath := path.Join(root, item.Name())
		if item.IsDir() {
			if err := copyContent(wt, filePath, mgmtPath); err != nil {
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
			log.V(2).Info("copied embedded file", "destination", dstPath)
		}
	}
	return nil
}

