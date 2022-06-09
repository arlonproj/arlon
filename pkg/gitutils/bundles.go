package gitutils

import (
	"bytes"
	"fmt"
	"github.com/arlonproj/arlon/pkg/bundle"
	"github.com/arlonproj/arlon/pkg/common"
	gogit "github.com/go-git/go-git/v5"
	"io"
	"io/fs"
	"path"
	"text/template"
)

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
{{- if eq .SrcType "helm" }}
    helm:
      parameters:
      # Pass cluster name to the bundle in case it needs it and is a Helm chart.
      # Example: this is required by the CAPI cluster autoscaler.
      # Use arlon prefix to avoid any conflicts with the bundle's own values.
      - name: arlon.clusterName
        value: {{.ClusterName}}
	{{- range .Overrides }}
      - name: {{ .Key }}
        value: {{ .Value }}
	{{- end }}
{{- else if eq .SrcType "kustomize" }}
    kustomize: {}
{{- else if eq .SrcType "ksonnet" }}
    ksonnet: {}
{{- else if eq .SrcType "directory" }}
    directory: {}
{{- end }}
`

type AppSettings struct {
	AppName              string
	ClusterName          string
	RepoUrl              string
	RepoPath             string
	RepoRevision         string
	SrcType              string
	AppNamespace         string
	DestinationNamespace string
	Overrides            []common.KVPair
}

func ProcessBundles(
	wt *gogit.Worktree,
	clusterName string,
	repoUrl string,
	mgmtPath string,
	workloadPath string,
	bundles []bundle.Bundle,
	overrides common.KVPairMap,
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
			app.SrcType = b.SrcType
			o := overrides[b.Name]
			app.Overrides = append(app.Overrides, o...)
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
