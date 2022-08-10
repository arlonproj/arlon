package basecluster

import (
	"fmt"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/otiai10/copy"
	"os"
	"path"
	"testing"
)

func TestPreparation(t *testing.T) {
	srcDir := path.Join("testdata", "requires_prep")
	tmpDir, err := os.MkdirTemp("", "arlon-unittest-")
	if err != nil {
		t.Fatalf("failed to create temp directory: %s", err)
	}
	err = copy.Copy(srcDir, tmpDir)
	if err != nil {
		t.Fatalf("failed to copy directory %s: %s", tmpDir, err)
	}
	fileInfos, err := readDir(tmpDir)
	if err != nil {
		t.Fatalf("failed to read directory %s: %s", tmpDir, err)
	}
	_, err = validateDir(tmpDir, fileInfos)
	if err == nil {
		t.Fatalf("validation returned no error")
	}
	fmt.Println("validation returned expected error:", err)
	fs := osfs.New(tmpDir)
	manifestFileName, clusterName, err := prepareDir(fs, ".", tmpDir)
	if err != nil {
		t.Fatalf("preparation failed: %s", err)
	}
	if manifestFileName != "manifest.yaml" {
		t.Fatalf("unexpected manifest file name: %s", manifestFileName)
	}
	if clusterName != "capi-quickstart" {
		t.Fatalf("unexpected cluster name: %s", clusterName)
	}
	// validate again
	fileInfos, err = readDir(tmpDir)
	if err != nil {
		t.Fatalf("failed to read directory %s: %s", tmpDir, err)
	}
	clusterName, err = validateDir(tmpDir, fileInfos)
	if err != nil {
		t.Fatalf("unexpected validation error: %s", err)
	}
	if clusterName != "capi-quickstart" {
		t.Fatalf("unexpected cluster name: %s", clusterName)
	}
}
