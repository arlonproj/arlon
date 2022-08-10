package basecluster

import (
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
}

func TestValidation(t *testing.T) {
	for _, testCase := range testData {
		dirPath := path.Join("testdata", testCase.DirName)
		dirEntries, err := os.ReadDir(dirPath)
		if err != nil {
			t.Fatalf("failed to read directory: %s", err)
		}
		var fileInfos []os.FileInfo
		for _, dirEntry := range dirEntries {
			fileInfo, err := dirEntry.Info()
			if err != nil {
				t.Fatalf("failed to get fileinfo in %s for %s",
					testCase.DirName, dirEntry.Name())
			}
			fileInfos = append(fileInfos, fileInfo)
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
