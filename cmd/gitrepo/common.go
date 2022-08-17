package gitrepo

import (
	"errors"
	"fmt"
	"github.com/argoproj/argo-cd/v2/util/localconfig"
	"io"
	"k8s.io/apimachinery/pkg/util/json"
	"os"
	"path/filepath"
)

type RepoCtx struct {
	Url   string `json:"url,omitempty"`
	Alias string `json:"alias,omitempty"`
}

type RepoCtxCfg struct {
	Default RepoCtx   `json:"default,omitempty"`
	Repos   []RepoCtx `json:"repos,omitempty"`
}

const (
	repoCtxFile    = "repoctx"
	repoDefaultCtx = "default"
)

var (
	errNotFound     = errors.New("given alias doesn't exist")
	errLoadCfgFile  = errors.New("failed to load config file")
	errOverwriteCfg = errors.New("cannot overwrite config file")
)

func getRepoCfgPath() (string, error) {
	cfgDir, err := localconfig.DefaultConfigDir()
	if err != nil {
		err = fmt.Errorf("cannot open config file %s, error: %w", cfgDir, err)
		return "", err
	}
	return filepath.Join(cfgDir, repoCtxFile), nil
}

func loadRepoCfg(reader io.Reader) (RepoCtxCfg, error) {
	content, err := io.ReadAll(reader)
	if err != nil {
		return RepoCtxCfg{}, err
	}
	if len(content) == 0 {
		return RepoCtxCfg{}, nil
	}
	var cfg RepoCtxCfg
	if err := json.Unmarshal(content, &cfg); err != nil {
		return RepoCtxCfg{}, err
	}
	return cfg, nil
}

func aliasExists(repoList []RepoCtx, targetAlias string) bool {
	for _, repoCtx := range repoList {
		if repoCtx.Alias == targetAlias {
			return true
		}
	}
	return false
}

func truncateFile(file *os.File) error {
	if err := file.Truncate(0); err != nil {
		return err
	}
	if _, err := file.Seek(0, 0); err != nil {
		return err
	}
	return nil
}

func storeRepoCfg(writer io.Writer, data []byte) error {
	_, err := writer.Write(data)
	return err
}
