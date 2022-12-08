package gitutils

import (
	"embed"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/arlonproj/arlon/pkg/log"
	gogit "github.com/go-git/go-git/v5"
	"gopkg.in/yaml.v2"
)

// -----------------------------------------------------------------------------

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

func CopyPatchManifests(wt *gogit.Worktree, fs embed.FS, filePath string, mgmtPath string) error {
	log := log.GetLogger()
	files, err := ioutil.ReadDir(filePath)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	for _, file := range files {
		fmt.Println(file.Name())
		newFilePath := path.Join(filePath, file.Name())
		fmt.Println(newFilePath)
		src, err := os.OpenFile(newFilePath, os.O_RDONLY, os.ModePerm)
		if err != nil {
			return fmt.Errorf("failed to open embedded file %s: %s", filePath, err)
		}

		file, err := ioutil.ReadFile(newFilePath)
		if err != nil {
			fmt.Println(err)
		}

		parsedData := make(map[interface{}]interface{})

		err2 := yaml.Unmarshal(file, &parsedData)
		if err2 != nil {
			fmt.Println(err2)
		}

		for k, v := range parsedData {
			fmt.Printf("%s -> %d\n", k, v)
		}

		// remove manifests/ prefix
		components := strings.Split(newFilePath, "/")
		dstPath := path.Join(components[len(components)-1])
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
	return nil
}
