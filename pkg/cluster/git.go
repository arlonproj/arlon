package cluster

import (
	"embed"
	"fmt"
	"path"
	"text/template"

	arlonv1 "github.com/arlonproj/arlon/api/v1"
	"github.com/arlonproj/arlon/pkg/argocd"
	bcl "github.com/arlonproj/arlon/pkg/basecluster"
	"github.com/arlonproj/arlon/pkg/bundle"
	"github.com/arlonproj/arlon/pkg/gitutils"
	logpkg "github.com/arlonproj/arlon/pkg/log"
	"github.com/arlonproj/arlon/pkg/profile"
	"github.com/go-git/go-billy"
	gogit "github.com/go-git/go-git/v5"
)

//go:embed manifests/*
var content embed.FS

// -----------------------------------------------------------------------------

func DeployToGit(
	creds *argocd.RepoCreds,
	argocdNs string,
	bundles []bundle.Bundle,
	clusterName string,
	repoUrl string,
	repoBranch string,
	basePath string,
	prof *arlonv1.Profile,
) error {
	log := logpkg.GetLogger()
	repo, tmpDir, auth, err := argocd.CloneRepo(creds, repoUrl, repoBranch)
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
	err = gitutils.CopyManifests(wt, content, ".", mgmtPath)
	if err != nil {
		return fmt.Errorf("failed to copy embedded content: %s", err)
	}
	profRepoUrl := prof.Spec.RepoUrl
	if profRepoUrl != "" {
		// dynamic profile: bundles not included in root app.
		// create an Application for the profile.
		profRepoPath := prof.Spec.RepoPath
		appPath := path.Join(mgmtPath, "templates", "profile.yaml")
		err = ProcessDynamicProfile(wt, clusterName, prof.Name, argocdNs,
			profRepoUrl, profRepoPath, appPath)
		if err != nil {
			return fmt.Errorf("failed to process dynamic profile: %s", err)
		}
	} else {
		// static profile: include bundles as individual Applications now
		om := profile.MakeOverridesMap(prof)
		err = gitutils.ProcessBundles(wt, clusterName, repoUrl, mgmtPath, workloadPath, bundles, om)
		if err != nil {
			return fmt.Errorf("failed to process bundles: %s", err)
		}
	}
	changed, err := gitutils.CommitChanges(tmpDir, wt, "deploy arlon cluster "+clusterPath)
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

// -----------------------------------------------------------------------------

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
	app := gitutils.AppSettings{
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

func DeployPatchToGit(
	creds *argocd.RepoCreds,
	clusterName string,
	repoUrl string,
	patchRepoRevision string,
	baseRepoRevision string,
	basePath string,
	patchContent []byte,
	baseRepoUrl string,
	baseRepoPath string,
) error {
	log := logpkg.GetLogger()
	repo, tmpDir, auth, err := argocd.CloneRepo(creds, repoUrl, patchRepoRevision)
	if err != nil {
		return fmt.Errorf("failed to clone repo: %s", err)
	}
	clusterPath := clusterPathFromBasePath(basePath, clusterName)
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
	err = gitutils.CopyPatchManifests(wt, patchContent, clusterPath, baseRepoUrl, baseRepoPath, baseRepoRevision)
	if err != nil {
		return fmt.Errorf("failed to copy embedded content: %s", err)
	}
	var file billy.File
	fs := wt.Filesystem
	file, err = fs.Create(path.Join(clusterPath, "configurations.yaml"))
	if err != nil {
		return fmt.Errorf("failed to create configurations.yaml: %s", err)
	}
	_, err = file.Write([]byte(bcl.ConfigurationsYaml))
	_ = file.Close()
	if err != nil {
		return fmt.Errorf("failed to write to configurations.yaml: %s", err)
	}

	_, err = gitutils.CommitChanges(tmpDir, wt, "deploy patches of the arlon cluster in "+clusterPath)
	if err != nil {
		return fmt.Errorf("failed to commit changes: %s", err)
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
