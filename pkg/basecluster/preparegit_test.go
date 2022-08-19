package basecluster

import (
	"errors"
	"github.com/arlonproj/arlon/pkg/argocd"
	"github.com/arlonproj/arlon/pkg/gitutils"
	gogit "github.com/go-git/go-git/v5"
	"github.com/otiai10/copy"
	"gotest.tools/assert"
	"os"
	"path"
	"testing"
)

func TestGitPreparation(t *testing.T) {
	subdirName := "requires_prep"
	repoRevision := "master"
	_, srcGitDir := createFileSystemBasedRepo(t, subdirName)
	t.Log("repo dir:", srcGitDir)
	creds := &argocd.RepoCreds{}
	repoUrl := "file://" + srcGitDir
	_, err := ValidateGitDir(creds, repoUrl, repoRevision, subdirName)
	assert.Assert(t, errors.Is(err, ErrNoKustomizationYaml), "unexpected validation error: %s", err)
	t.Log("got expected error:", err)
	clustName, changed, err := PrepareGitDir(creds, repoUrl, repoRevision, subdirName)
	assert.NilError(t, err, "failed to prepare git directory")
	assert.Assert(t, changed, "git dir preparation resulted in no changes")
	assert.Equal(t, clustName, "capi-quickstart", "unexpected cluster name: %s", clustName)
	_, err = ValidateGitDir(creds, repoUrl, repoRevision, subdirName)
	assert.NilError(t, err, "unexpected 2nd validation error: %s", err)
}

func createFileSystemBasedRepo(t *testing.T, subdirName string) (*gogit.Repository, string) {
	srcDir := path.Join("testdata", subdirName)
	tmpDir, err := os.MkdirTemp("", "arlon-unittest-")
	assert.NilError(t, err)
	repo, err := gogit.PlainInit(tmpDir, false)
	assert.NilError(t, err, "failed to init git dir")
	dstDir := path.Join(tmpDir, subdirName)
	err = copy.Copy(srcDir, dstDir)
	assert.NilError(t, err, "failed to copy directory")
	wt, err := repo.Worktree()
	assert.NilError(t, err, "failed to get worktree")
	changed, err := gitutils.CommitChanges(tmpDir, wt, "initial commit")
	assert.Assert(t, changed, "unexpected changed status from CommitChanges()")
	t.Cleanup(func() { os.RemoveAll(tmpDir) })
	return repo, tmpDir
}
