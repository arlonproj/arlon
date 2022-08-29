package basecluster

import (
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/go-git/go-billy/v5/osfs"
	"github.com/otiai10/copy"
	"gotest.tools/v3/assert"
)

type testCase struct {
	dirName         string
	expectedPrepErr error
}

var testCases = []testCase{
	{"requires_prep", nil},
	{"requires_prep_2", Err2orMoreClusters},
}

func TestPreparation(t *testing.T) {

	for _, tc := range testCases {
		testOneDir(t, tc)
	}
}

func testOneDir(t *testing.T, tc testCase) {
	srcDir := path.Join("testdata", tc.dirName)
	tmpDir, err := os.MkdirTemp("", "arlon-unittest-")
	assert.NilError(t, err)
	err = copy.Copy(srcDir, tmpDir)
	assert.NilError(t, err, "failed to copy directory %s: %s", tmpDir, err)
	fileInfos, err := readDir(tmpDir)
	assert.NilError(t, err, "failed to read directory %s: %s", tmpDir, err)
	_, err = validateDir(tmpDir, fileInfos)
	if err == nil {
		t.Fatalf("validation returned no error")
	}
	fmt.Println("validation returned expected error:", err)
	fs := osfs.New(tmpDir)
	manifestFileName, clusterName, err := prepareDir(fs, ".", tmpDir)
	if tc.expectedPrepErr == nil {
		assert.NilError(t, err, "expected nil error")
	} else {
		assert.ErrorContains(t, err, tc.expectedPrepErr.Error())
		return
	}
	assert.Equal(t, manifestFileName, "manifest.yaml")
	assert.Equal(t, clusterName, "capi-quickstart")

	// validate again
	fileInfos, err = readDir(tmpDir)
	assert.NilError(t, err)
	clusterName, err = validateDir(tmpDir, fileInfos)
	assert.NilError(t, err)
	assert.Equal(t, clusterName, "capi-quickstart")
}
