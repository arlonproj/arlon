package cluster

import (
	"context"
	"fmt"

	argoclient "github.com/argoproj/argo-cd/v2/pkg/apiclient"
	argoapp "github.com/argoproj/argo-cd/v2/pkg/apiclient/application"
	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/arlonproj/arlon/pkg/argocd"
	"github.com/arlonproj/arlon/pkg/gitutils"
	logpkg "github.com/arlonproj/arlon/pkg/log"
	gogit "github.com/go-git/go-git/v5"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
)

//------------------------------------------------------------------------------

func Delete(
	// appIf argoapp.ApplicationServiceClient,
	argoIf argoclient.Client,
	config *restclient.Config,
	argocdNs string,
	name string,
) error {
	//log := logpkg.GetLogger()
	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to get kube client: %s", err)
	}
	conn, appIf, err := argoIf.NewApplicationClient()
	if err != nil {
		return fmt.Errorf("failed to get argocd application client: %s", err)
	}
	defer conn.Close()
	clust, err := Get(appIf, config, argocdNs, name)
	if err != nil {
		return fmt.Errorf("failed to get existing cluster: %s", err)
	}
	if clust.IsExternal {
		return UnmanageExternal(argoIf, config, argocdNs, name)
	}
	if clust.BaseCluster == nil {
		cascade := true
		_, err = appIf.Delete(
			context.Background(),
			&argoapp.ApplicationDeleteRequest{
				Name:    &name,
				Cascade: &cascade,
			})
		return err
	}

	clusterQuery := "arlon-cluster=" + name
	apps, err := appIf.List(context.Background(),
		&argoapp.ApplicationQuery{Selector: &clusterQuery})
	if err != nil {
		return fmt.Errorf("failed to list apps related to cluster: %s", err)
	}

	for _, app := range apps.Items {
		if app.Labels["arlon-type"] == "cluster-app" {
			overridden := app.Annotations[baseClusterOverridden]
			if overridden == "true" {
				err = DeleteOverridesDir(&app, kubeClient, argocdNs, name)
				if err != nil {
					return fmt.Errorf("failed to delete the overrides directory: %s", err)
				}
			}
		}
		cascade := true
		_, err = appIf.Delete(
			context.Background(),
			&argoapp.ApplicationDeleteRequest{
				Name:    &app.Name,
				Cascade: &cascade,
			})
		if err != nil {
			return fmt.Errorf("failed to delete related app %s: %s",
				app.Name, err)
		}
		fmt.Println("deleted related app:", app.Name)
	}
	return nil
}

func DeleteOverridesDir(app *v1alpha1.Application, kubeClient *kubernetes.Clientset, argocdNs string, clusterName string) error {
	log := logpkg.GetLogger()
	repoUrl := app.Annotations[baseClusterRepoUrlAnnotation]
	repoRevision := app.Annotations[baseClusterRepoRevisionAnnotation]
	repoPath := app.Annotations[baseClusterRepoPathAnnotation] + "/" + clusterName
	creds, err := argocd.GetRepoCredsFromArgoCd(kubeClient, argocdNs, repoUrl)
	if err != nil {
		return fmt.Errorf("failed to get repo credentials: %s", err)
	}
	repo, tmpDir, auth, err := argocd.CloneRepo(creds, repoUrl, repoRevision)
	if err != nil {
		return fmt.Errorf("failed to clone repo: %s", err)
	}
	wt, err := repo.Worktree()
	fileInfo, err := wt.Filesystem.Lstat(repoPath)
	if err == nil {
		if !fileInfo.IsDir() {
			return fmt.Errorf("unexpected file type for %s", repoPath)
		}
		_, err := wt.Remove(repoPath)
		if err != nil {
			return fmt.Errorf("failed to recursively delete cluster directory: %s", err)
		}
	} else {
		return fmt.Errorf("Failed to find the directory to delete: %s", err)
	}
	commitMsg := "Deleted the files regarding to " + repoPath
	changed, err := gitutils.CommitDeleteChanges(tmpDir, wt, commitMsg)
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
	log.V(1).Info("successfully pushed working tree", "tmpDir", tmpDir)
	return nil
}
