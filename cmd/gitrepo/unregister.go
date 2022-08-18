package gitrepo

import (
	"encoding/json"
	"fmt"
	"github.com/arlonproj/arlon/pkg/gitrepo"
	"github.com/spf13/cobra"
	"os"
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
			cfgFile, err := gitrepo.GetRepoCfgPath()
			if err != nil {
				return fmt.Errorf("%v: %w", gitrepo.ErrLoadCfgFile, err)
			}
			file, err := os.OpenFile(cfgFile, os.O_RDWR|os.O_CREATE, 0666)
			if err != nil {
				return fmt.Errorf("%v: %w", gitrepo.ErrLoadCfgFile, err)
			}
			defer func(f *os.File) {
				err := f.Close()
				if err != nil {
					fmt.Printf("failed to close config file, error: %v\n", err)
				}
			}(file)
			repoCtxCfg, err := gitrepo.LoadRepoCfg(file)
			if err != nil {
				return fmt.Errorf("%v: %w", gitrepo.ErrLoadCfgFile, err)
			}
			if len(repoCtxCfg.Repos) == 0 {
				fmt.Println("no repositories registered")
				return
			}
			for i, repo := range repoCtxCfg.Repos {
				if repo.Alias != repoAlias {
					continue
				}
				if repo.Alias == gitrepo.RepoDefaultCtx {
					repoCtxCfg.Default = gitrepo.RepoCtx{}
				}
				repoCtxCfg.Repos = append(repoCtxCfg.Repos[:i], repoCtxCfg.Repos[i+1:]...)
				repoData, err := json.MarshalIndent(repoCtxCfg, "", "\t")
				if err != nil {
					return err
				}
				if err := gitrepo.TruncateFile(file); err != nil {
					return fmt.Errorf("%v: %w", gitrepo.ErrOverwriteCfg, err)
				}
				if err := gitrepo.StoreRepoCfg(file, repoData); err != nil {
					return fmt.Errorf("%v: %w", gitrepo.ErrOverwriteCfg, err)
				}
				fmt.Printf("Repository %s unregistered locally\n", repoAlias)
				return nil
			}
			return gitrepo.ErrNotFound
		},
	}
	return command
}
