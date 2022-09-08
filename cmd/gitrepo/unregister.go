package gitrepo

import (
	"encoding/json"
	"fmt"

	argocdio "github.com/argoproj/argo-cd/v2/util/io"
	"github.com/arlonproj/arlon/pkg/gitrepo"
	"github.com/spf13/cobra"
)

func unregister() *cobra.Command {
	var (
		repoAlias string
	)
	command := &cobra.Command{
		Use:     "unregister",
		Args:    cobra.ExactArgs(1),
		Short:   "unregister a previously registered configuration",
		Long:    "unregister a previously registered configuration",
		Example: "arlon git unregister <repoAlias>",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			repoAlias = args[0]
			file, err := gitrepo.ReadDefaultConfig()
			if err != nil {
				return err
			}
			defer argocdio.Close(file)
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
