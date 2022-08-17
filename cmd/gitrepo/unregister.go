package gitrepo

import (
	"encoding/json"
	"fmt"
	"github.com/argoproj/argo-cd/v2/util/localconfig"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
)

func unregister() *cobra.Command {
	var (
		repoAlias string
	)
	command := &cobra.Command{
		Use:   "unregister",
		Args:  cobra.ExactArgs(1),
		Short: "unregister a previously registered configuration",
		Long:  "unregister a previously registered configuration",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			repoAlias = args[0]
			cfgDir, err := localconfig.DefaultConfigDir()
			if err != nil {
				return fmt.Errorf("%v: %w", errLoadCfgFile, err)
			}
			cfgFile := filepath.Join(cfgDir, repoCtxFile)
			file, err := os.OpenFile(cfgFile, os.O_RDWR|os.O_CREATE, 0666)
			if err != nil {
				return fmt.Errorf("%v: %w", errLoadCfgFile, err)
			}
			defer func(f *os.File) {
				err := f.Close()
				if err != nil {
					fmt.Printf("failed to close config file, error: %v\n", err)
				}
			}(file)
			repoCtxCfg, err := loadRepoCfg(file)
			if err != nil {
				return fmt.Errorf("%v: %w", errLoadCfgFile, err)
			}
			if len(repoCtxCfg.Repos) == 0 {
				fmt.Println("no repositories registered")
				return
			}
			for i, repo := range repoCtxCfg.Repos {
				if repo.Alias != repoAlias {
					continue
				}
				if repo.Alias == repoDefaultCtx {
					repoCtxCfg.Default = RepoCtx{}
				}
				repoCtxCfg.Repos = append(repoCtxCfg.Repos[:i], repoCtxCfg.Repos[i+1:]...)
				repoData, err := json.MarshalIndent(repoCtxCfg, "", "\t")
				if err != nil {
					return err
				}
				if err := truncateFile(file); err != nil {
					return fmt.Errorf("%v: %w", errOverwriteCfg, err)
				}
				if err := storeRepoCfg(file, repoData); err != nil {
					return fmt.Errorf("%v: %w", errOverwriteCfg, err)
				}
				fmt.Printf("Repository %s unregistered locally\n", repoAlias)
				return nil
			}
			return errNotFound
		},
	}
	return command
}
