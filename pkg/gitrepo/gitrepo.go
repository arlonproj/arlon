package gitrepo

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	argocdio "github.com/argoproj/argo-cd/v2/util/io"
	"github.com/argoproj/argo-cd/v2/util/localconfig"
	"k8s.io/apimachinery/pkg/util/json"
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
	arlonCfgDir    = ".arlon"
	RepoDefaultCtx = "default"
)

var (
	ErrNotFound     = errors.New("given alias doesn't exist")
	ErrLoadCfgFile  = errors.New("failed to load config file")
	ErrOverwriteCfg = errors.New("cannot overwrite config file")
)

func getRepoCfgPath() (string, error) {
	argoDir, err := localconfig.DefaultConfigDir()
	if err != nil {
		return "", err
	}
	cfgBase := filepath.Dir(argoDir)
	return filepath.Join(cfgBase, arlonCfgDir, repoCtxFile), nil
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

func getAlias(repoAlias string, reader io.Reader) (*RepoCtx, error) {
	cfgData, err := LoadRepoCfg(reader)
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

func GetRepoUrl(repoAlias string) (string, error) {
	cfgFile, err := ReadDefaultConfig()
	if err != nil {
		return "", err
	}
	defer argocdio.Close(cfgFile)
	repoCtx, err := getAlias(repoAlias, cfgFile)
	if err != nil {
		if !errors.Is(err, ErrNotFound) {
			return "", fmt.Errorf("%v: %w", ErrNotFound, err)
		}
		return "", err
	}
	return repoCtx.Url, nil
}

func ReadDefaultConfig() (*os.File, error) {
	cfgFile, err := getRepoCfgPath()
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(cfgFile); err != nil {
		err := os.MkdirAll(filepath.Dir(cfgFile), os.ModePerm)
		if err != nil {
			return nil, err
		}
	}
	file, err := os.OpenFile(cfgFile, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		err = fmt.Errorf("%v: %w", ErrLoadCfgFile, err)
		return nil, err
	}
	return file, nil
}
