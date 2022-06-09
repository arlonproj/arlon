package gitutils

import (
	"github.com/arlonproj/arlon/pkg/common"
	"strings"
	"testing"
	"text/template"
)

var expectedTemplateOutput = `
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: testApp
  namespace: yyy
  finalizers:
  # This solves issue #17
  - resources-finalizer.argocd.argoproj.io/foreground
spec:
  syncPolicy:
    automated:
      prune: true
  destination:
    name: testCluster
    namespace: zzz
  project: default
  source:
    repoURL: testRepoUrl
    path: testRepoPath
    targetRevision: testRepoRevision
    helm:
      parameters:
      # Pass cluster name to the bundle in case it needs it and is a Helm chart.
      # Example: this is required by the CAPI cluster autoscaler.
      # Use arlon prefix to avoid any conflicts with the bundle's own values.
      - name: arlon.clusterName
        value: testCluster
      - name: foo
        value: bar
      - name: goo
        value: gar
`

func TestAppTemplate(t *testing.T) {
	appSettings := AppSettings{
		AppName:              "testApp",
		ClusterName:          "testCluster",
		RepoUrl:              "testRepoUrl",
		RepoPath:             "testRepoPath",
		RepoRevision:         "testRepoRevision",
		SrcType:              "helm",
		AppNamespace:         "yyy",
		DestinationNamespace: "zzz",
		Overrides: []common.KVPair{
			{Key: "foo", Value: "bar"},
			{Key: "goo", Value: "gar"},
		},
	}
	tmpl, err := template.New("app").Parse(appTmpl)
	if err != nil {
		t.Fatalf("failed to create template: %s", err)
	}
	b := new(strings.Builder)
	err = tmpl.Execute(b, &appSettings)
	if err != nil {
		t.Fatalf("failed to execute template: %s", err)
	}
	if b.String() != expectedTemplateOutput {
		t.Fatalf("template output doesn't match: %s", b.String())
	}
}
