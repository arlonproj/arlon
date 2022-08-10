package gitutils

import (
	"fmt"
	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"os"
	"path/filepath"
	"time"
)

func CommitChanges(tmpDir string, wt *gogit.Worktree, commitMsg string) (changed bool, err error) {
	status, err := wt.Status()
	if err != nil {
		return false, fmt.Errorf("failed to get worktree status: %s", err)
	}

	// The following was copied from flux2/internal/bootstrap/git/gogit/gogit.go:
	//
	// go-git has [a bug](https://github.com/go-git/go-git/issues/253)
	// whereby it thinks broken symlinks to absolute paths are
	// modified. There's no circumstance in which we want to commit a
	// change to a broken symlink: so, detect and skip those.
	for file, sts := range status {
		if sts.Staging == gogit.Deleted {
			continue
		}
		abspath := filepath.Join(tmpDir, file)
		info, err := os.Lstat(abspath)
		if err != nil {
			return false, fmt.Errorf("failed to check if %s is a symlink: %w", file, err)
		}
		if info.Mode()&os.ModeSymlink > 0 {
			// symlinks are OK; broken symlinks are probably a result
			// of the bug mentioned above, but not of interest in any
			// case.
			if _, err := os.Stat(abspath); os.IsNotExist(err) {
				continue
			}
		}
		_, _ = wt.Add(file)
		changed = true
	}

	if !changed {
		return false, nil
	}
	commitOpts := &gogit.CommitOptions{
		Author: &object.Signature{
			Name:  "arlon automation",
			Email: "arlon@arlon.io",
			When:  time.Now(),
		},
	}
	_, err = wt.Commit(commitMsg, commitOpts)
	if err != nil {
		return changed, fmt.Errorf("failed to commit change: %s", err)
	}
	return
}
