package gitutils

import (
	"embed"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/arlonproj/arlon/pkg/log"
	"github.com/go-git/go-billy"
	gogit "github.com/go-git/go-git/v5"
	"gopkg.in/yaml.v2"
)

// // -----------------------------------------------------------------------------

type kustomizeyaml struct {
	APIVersion     string   `yaml:"apiVersion"`
	Kind           string   `yaml:"kind"`
	Resources      []string `yaml:"resources"`
	Configurations []string `yaml:"configurations"`
	Patches        []string `yaml:"patchesStrategicMerge"`
}

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

func CopyPatchManifests(wt *gogit.Worktree, filePath string, clusterPath string,
	baseRepoUrl string, baseRepoPath string, baseRepoRevision string) error {
	log := log.GetLogger()
	src, err := os.OpenFile(filePath, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to open the patch file %s:", err)
	}
	defer src.Close()
	_, fileName := filepath.Split(filePath)
	resourcestring := "git::" + baseRepoUrl + "//" + baseRepoPath + "?ref=" + baseRepoRevision
	kustomizeresult := kustomizeyaml{
		APIVersion: "kustomize.config.k8s.io/v1beta1",
		Kind:       "Kustomization",
		Resources: []string{
			resourcestring,
		},
		Configurations: []string{
			"configurations.yaml",
		},
		Patches: []string{
			fileName,
		},
	}
	var tmpl *template.Template
	yamlData, err := yaml.Marshal(&kustomizeresult)
	if err != nil {
		return fmt.Errorf("Failed to marshal the kustomization file: %s", err)
	}
	tmpl, err = template.New("kust").Parse(string(yamlData))
	if err != nil {
		return fmt.Errorf("failed to create kustomization template: %s", err)
	}
	var file billy.File
	fs := wt.Filesystem
	file, err = fs.Create(path.Join(clusterPath, "kustomization.yaml"))
	if err != nil {
		return fmt.Errorf("failed to create kustomization.yaml: %s", err)
	}
	defer file.Close()
	err = tmpl.Execute(file, yamlData)
	if err != nil {
		return fmt.Errorf("failed to execute kustomization.yaml manifest")
	}
	if err != nil {
		return fmt.Errorf("failed to write to kustomization.yaml: %s", err)
	}
	dstPath := path.Join(clusterPath, fileName)
	dst, err := wt.Filesystem.Create(dstPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file %s: %s", dstPath, err)
	}
	defer dst.Close()
	_, err = io.Copy(dst, src)
	if err != nil {
		return fmt.Errorf("failed to copy embedded file: %s", err)
	}
	log.V(1).Info("copied embedded file", "destination", dstPath)
	return nil
}
