package cluster

import (
	"context"
	"fmt"

	argoclient "github.com/argoproj/argo-cd/v2/pkg/apiclient"
	apppkg "github.com/argoproj/argo-cd/v2/pkg/apiclient/application"
	argoapp "github.com/argoproj/argo-cd/v2/pkg/apiclient/application"
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

	app, err := appIf.Get(context.Background(),
		&apppkg.ApplicationQuery{
			Name: &name,
		})
	if err != nil {
		return fmt.Errorf("failed to get argocd application: %s", err)
	}
	typ := app.Labels["arlon-type"]
	if typ == "cluster-app" {
		log := logpkg.GetLogger()
		RepoUrl := app.Annotations[baseClusterRepoUrlAnnotation]
		RepoRevision := app.Annotations[baseClusterRepoRevisionAnnotation]
		RepoPath := app.Annotations[baseClusterRepoPathAnnotation] + "/" + name
		overRiden := app.Annotations[baseClusterOverriden]
		if overRiden == "true" {
			creds, err := argocd.GetRepoCredsFromArgoCd(kubeClient, argocdNs, RepoUrl)
			repo, tmpDir, auth, err := argocd.CloneRepo(creds, RepoUrl, RepoRevision)
			if err != nil {
				return fmt.Errorf("failed to clone repo: %s", err)
			}
			wt, err := repo.Worktree()
			fileInfo, err := wt.Filesystem.Lstat(RepoPath)
			if err == nil {
				if !fileInfo.IsDir() {
					return fmt.Errorf("unexpected file type for %s", RepoPath)
				}
				_, err := wt.Remove(RepoPath)
				if err != nil {
					return fmt.Errorf("failed to recursively delete cluster directory: %s", err)
				}
			}

			commitMsg := "Deleted the files regarding to " + RepoPath
			changed, err := gitutils.CommitDeletechanges(tmpDir, wt, commitMsg)
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
		}
	}

	for _, app := range apps.Items {
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
