package cluster

import (
	"context"
	"fmt"
	"github.com/argoproj/argo-cd/v2/util/cli"
	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/spf13/cobra"
	"io"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	_ "embed"
	"path"
	"path/filepath"
	"strings"
	"time"
)

//go:embed manifests/Chart.yaml
var chartYaml string

func deployClusterCommand() *cobra.Command {
	var clientConfig clientcmd.ClientConfig
	var ns string
	var repoUrl string
	var repoBranch string
	var basePath string
	var clusterName string
	command := &cobra.Command{
		Use:               "deploy",
		Short:             "Deploy cluster",
		Long:              "Deploy cluster",
		RunE: func(c *cobra.Command, args []string) error {
			config, err := clientConfig.ClientConfig()
			if err != nil {
				return fmt.Errorf("failed to get k8s client config: %s", err)
			}
			return deployCluster(config, ns, clusterName, repoUrl, repoBranch, basePath)
		},
	}
	clientConfig = cli.AddKubectlFlagsToCmd(command)
	command.Flags().StringVar(&ns, "ns", "argocd", "the argocd namespace")
	command.Flags().StringVar(&repoUrl, "repo-url", "", "the git repository url")
	command.Flags().StringVar(&repoBranch, "repo-branch", "main", "the git branch")
	command.Flags().StringVar(&clusterName, "cluster-name", "", "the cluster name")
	command.Flags().StringVar(&basePath, "path", "arlon", "the git repository base path")
	command.MarkFlagRequired("repo-url")
	command.MarkFlagRequired("cluster-name")
	return command
}

type RepoCreds struct {
	Url string
	Username string
	Password string
}

func deployCluster(config *restclient.Config, ns string, clusterName string, repoUrl string, repoBranch string, basePath string) error {
	kubeClient := kubernetes.NewForConfigOrDie(config)
	corev1 := kubeClient.CoreV1()
	secretsApi := corev1.Secrets(ns)
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
	chartPath := path.Join(mgmtPath, "Chart.yaml")
	wt, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get repo worktree: %s", err)
	}
	f, err := wt.Filesystem.Create(chartPath)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, strings.NewReader(chartYaml))
	if err != nil {
		return fmt.Errorf("failed to copy chart YAML to worktree: %s", err)
	}
	status, err := wt.Status()
	if err != nil {
		return fmt.Errorf("failed to get worktree status: %s", err)
	}

	// The following was copied from flux2/internal/bootstrap/git/gogit/gogit.go:
	//
	// go-git has [a bug](https://github.com/go-git/go-git/issues/253)
	// whereby it thinks broken symlinks to absolute paths are
	// modified. There's no circumstance in which we want to commit a
	// change to a broken symlink: so, detect and skip those.
	var changed bool
	for file, _ := range status {
		abspath := filepath.Join(tmpDir, file)
		info, err := os.Lstat(abspath)
		if err != nil {
			return fmt.Errorf("failed to check if %s is a symlink: %w", file, err)
		}
		if info.Mode()&os.ModeSymlink > 0 {
			// symlinks are OK; broken symlinks are probably a result
			// of the bug mentioned above, but not of interest in any
			// case.
			if _, err := os.Stat(abspath); os.IsNotExist(err) {
				continue
			}
		}
		_, _ = wt.Add(file)
		changed = true
	}

	if !changed {
		return nil
	}
	commitOpts := &gogit.CommitOptions{
		Author: &object.Signature{
			Name:  "arlon automation",
			Email: "arlon@arlon.io",
			When:  time.Now(),
		},
	}
	commitMsg := "add arlon manifests"
	_, err = wt.Commit(commitMsg, commitOpts)
	if err != nil {
		return fmt.Errorf("failed to commit change: %s", err)
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
	return nil
}

