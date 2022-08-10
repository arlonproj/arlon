package basecluster

import (
	"fmt"
	"os"
	"path"
	"strings"
	"testing"
)

type testEntry struct {
	DirName     string
	ErrPattern  string
	ClusterName string
}

var testData = []testEntry{
	{
		"01_no_configurations",
		ErrNoConfigurationsYaml.Error(),
		"",
	},
	{
		"02_no_kustomization",
		ErrNoKustomizationYaml.Error(),
		"",
	},
	{
		"03_no_manifest",
		ErrNoManifest.Error(),
		"",
	},
	{
		"04_multiple_manifests",
		ErrMultipleManifests.Error(),
		"",
	},
	{
		"05_has_namespace",
		"has a namespace defined",
		"",
	},
	{
		"06_multiple_clusters",
		ErrMultipleClusters.Error(),
		"",
	},
	{
		"07_no_cluster",
		ErrNoClusterResource.Error(),
		"",
	},
	{
		"08_ok",
		"",
		"capi-quickstart",
	},
	{
		"09_invalid_manifest",
		"builder failed to run",
		"",
	},
}

func TestValidation(t *testing.T) {
	for _, testCase := range testData {
		dirPath := path.Join("testdata", testCase.DirName)
		fileInfos, err := readDir(dirPath)
		if err != nil {
			t.Fatalf("failed to read directory %s: %s", dirPath, err)
		}
		clustName, err := validateDir(dirPath, fileInfos)
		if err != nil {
			if testCase.ErrPattern == "" {
				t.Fatalf("unexpected error in %s: %s", testCase.DirName, err)
			}
			if !strings.Contains(err.Error(), testCase.ErrPattern) {
				t.Fatalf("unexpected error in %s: %s", testCase.DirName, err)
			}
		} else if testCase.ErrPattern != "" {
			t.Fatalf("did not find expected error in %s", testCase.DirName)
		} else if clustName != testCase.ClusterName {
			t.Fatalf("unexpected cluster name: %s", clustName)
		}
	}
}

func readDir(dirPath string) (fileInfos []os.FileInfo, err error) {
	dirEntries, err := os.ReadDir(dirPath)
	if err != nil {
		err = fmt.Errorf("failed to read directory: %s", err)
		return
	}
	for _, dirEntry := range dirEntries {
		fileInfo, err := dirEntry.Info()
		if err != nil {
			err = fmt.Errorf("failed to get fileinfo for %s",
				dirEntry.Name())
		}
		fileInfos = append(fileInfos, fileInfo)
	}
	return
}
