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
	RepoDefaultCtx = "default"
)

var (
	ErrNotFound     = errors.New("given alias doesn't exist")
	ErrLoadCfgFile  = errors.New("failed to load config file")
	ErrOverwriteCfg = errors.New("cannot overwrite config file")
)

func GetRepoCfgPath() (string, error) {
	cfgDir, err := localconfig.DefaultConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cfgDir, repoCtxFile), nil
}

func LoadRepoCfg(reader io.Reader) (RepoCtxCfg, error) {
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

func AliasExists(repoList []RepoCtx, targetAlias string) bool {
	for _, repoCtx := range repoList {
		if repoCtx.Alias == targetAlias {
			return true
		}
	}
	return false
}

func TruncateFile(file *os.File) error {
	if err := file.Truncate(0); err != nil {
		return err
	}
	if _, err := file.Seek(0, 0); err != nil {
		return err
	}
	return nil
}

func StoreRepoCfg(writer io.Writer, data []byte) error {
	_, err := writer.Write(data)
	return err
}

func GetAlias(repoAlias string) (*RepoCtx, error) {
	cfgPath, err := GetRepoCfgPath()
	if err != nil {
		return nil, err
	}
	cfgFile, err := os.OpenFile(cfgPath, os.O_RDONLY, 0666)
	if err != nil {
		return nil, err
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			fmt.Printf("failed to close config file, error: %v\n", err)
		}
	}(cfgFile)
	cfgData, err := LoadRepoCfg(cfgFile)
	if err != nil {
		return nil, err
	}
	if repoAlias == RepoDefaultCtx {
		defaultCfg := cfgData.Default
		return &defaultCfg, nil
	}
	for _, repo := range cfgData.Repos {
		if repo.Alias != repoAlias {
			continue
		}
		return &repo, nil
	}
	return nil, ErrNotFound
}
