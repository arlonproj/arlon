package gitutils

import (
	"embed"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"text/template"

	"github.com/arlonproj/arlon/pkg/log"
	"github.com/go-git/go-billy"
	gogit "github.com/go-git/go-git/v5"
	"gopkg.in/yaml.v2"
)

// // -----------------------------------------------------------------------------

type kustomizeyaml struct {
	APIVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	Resources  []string `yaml:"resources"`
	Patches    []target `yaml:"patches"`
}
type target struct {
	Target info   `yaml:"target"`
	Path   string `yaml:"path"`
}

type info struct {
	Group   string `yaml:"group"`
	Version string `yaml:"version"`
	Kind    string `yaml:"kind"`
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

func CopyPatchManifests(wt *gogit.Worktree, filePath string, clusterPath string) error {
	log := log.GetLogger()
	files, err := ioutil.ReadDir(filePath)
	if err != nil {
		fmt.Println(err)
		return fmt.Errorf("Failed to read the directory %s", err)
	}
	var targetData []target
	for _, file := range files {
		fmt.Println(file.Name())
		newFilePath := path.Join(filePath, file.Name())
		fmt.Println(newFilePath)
		src, err := os.OpenFile(newFilePath, os.O_RDONLY, os.ModePerm)
		if err != nil {
			return fmt.Errorf("failed to open embedded file %s: %s", filePath, err)
		}

		kustomfile, err := ioutil.ReadFile(newFilePath)
		if err != nil {
			return fmt.Errorf("Failed to read the embedded file %s", err)
		}

		parsedData := make(map[interface{}]interface{})

		err2 := yaml.Unmarshal(kustomfile, &parsedData)
		if err2 != nil {
			fmt.Println(err2)
		}
		var targetcomp []string
		var kind string
		for k, v := range parsedData {
			if k == "apiVersion" {
				strv := fmt.Sprintf("%v", v)
				targetcomp = strings.Split(string(strv), "/")
			}
			if k == "kind" {
				kind = fmt.Sprintf("%v", v)
			}
			information := info{
				Group:   targetcomp[0],
				Version: targetcomp[1],
				Kind:    kind,
			}
			targetData = append(targetData, target{
				information, file.Name(),
			})
		}

		// remove manifests/ prefix
		components := strings.Split(newFilePath, "/")
		dstPath := path.Join(components[len(components)-1])
		dstPath = path.Join(clusterPath, dstPath)
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
	kustomizeresult := kustomizeyaml{
		APIVersion: "kustomize.config.k8s.io/v1beta1",
		Kind:       "Kustomization",
		Resources: []string{
			"../../bc1",
		},
		Patches: targetData,
	}
	var tmpl *template.Template
	yamlData, err := yaml.Marshal(&kustomizeresult)
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
	err = tmpl.Execute(file, yamlData)
	_ = file.Close()
	if err != nil {
		return fmt.Errorf("failed to write to kustomization.yaml: %s", err)
	}
	return nil
}
