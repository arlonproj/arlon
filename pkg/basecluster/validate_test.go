package basecluster

import (
	"errors"
	"fmt"
	"os"
	"path"
	"testing"

	"gotest.tools/v3/assert"
)

type testEntry struct {
	DirName     string
	ErrPattern  error
	ClusterName string
}

var testData = []testEntry{
	{
		"01_no_configurations",
		ErrNoConfigurationsYaml,
		"",
	},
	{
		"02_no_kustomization",
		ErrNoKustomizationYaml,
		"",
	},
	{
		"03_no_manifest",
		ErrNoManifest,
		"",
	},
	{
		"04_multiple_manifests",
		ErrMultipleManifests,
		"",
	},
	{
		"05_has_namespace",
		ErrResourceHasNamespace,
		"",
	},
	{
		"06_multiple_clusters",
		ErrMultipleClusters,
		"",
	},
	{
		"07_no_cluster",
		ErrNoClusterResource,
		"",
	},
	{
		"08_ok",
		nil,
		"capi-quickstart",
	},
	{
		"09_invalid_manifest",
		ErrBuilderFailedRun,
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
		if !errors.Is(err, testCase.ErrPattern) {
			t.Fatalf("unexpected error in %s, expected: %v, got: %v", testCase.DirName, testCase.ErrPattern, err)
		}
		assert.Equal(t, clustName, testCase.ClusterName)
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
			return nil, err
		}
		fileInfos = append(fileInfos, fileInfo)
	}
	return
}
