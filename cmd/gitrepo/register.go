package gitrepo

import (
	"context"
	"encoding/json"
	"fmt"
	cmdutil "github.com/argoproj/argo-cd/v2/cmd/util"
	"github.com/argoproj/argo-cd/v2/pkg/apiclient/repository"
	"github.com/argoproj/argo-cd/v2/util/cli"
	"github.com/argoproj/argo-cd/v2/util/errors"
	argocdio "github.com/argoproj/argo-cd/v2/util/io"
	"github.com/arlonproj/arlon/pkg/argocd"
	"github.com/arlonproj/arlon/pkg/gitrepo"
	"github.com/spf13/cobra"
)

func register() *cobra.Command {
	var (
		repoUrl  string
		userName string
		password string
		alias    string
	)
	command := &cobra.Command{
		Use:   "register",
		Short: "register a git repository configuration",
		Long:  "register a git repository configuration",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			var repoOpts cmdutil.RepoOptions
			repoUrl = args[0]
			repoOpts.Repo.Repo = repoUrl
			repoOpts.Repo.Password = password
			repoOpts.Repo.Username = userName

			// Taken from argo-cd/cmd/argocd/commands/repo.go
			// Set repository connection properties only when creating repository, not
			// when creating repository credentials.
			// InsecureIgnoreHostKey is deprecated and only here for backwards compat
			repoOpts.Repo.InsecureIgnoreHostKey = repoOpts.InsecureIgnoreHostKey
			repoOpts.Repo.Insecure = repoOpts.InsecureSkipServerVerification
			repoOpts.Repo.EnableLFS = repoOpts.EnableLfs
			repoOpts.Repo.EnableOCI = repoOpts.EnableOci
			repoOpts.Repo.GithubAppId = repoOpts.GithubAppId
			repoOpts.Repo.GithubAppInstallationId = repoOpts.GithubAppInstallationId
			repoOpts.Repo.GitHubAppEnterpriseBaseURL = repoOpts.GitHubAppEnterpriseBaseURL
			repoOpts.Repo.Proxy = repoOpts.Proxy

			conn, client := argocd.NewArgocdClientOrDie("").NewRepoClientOrDie()
			defer argocdio.Close(conn)
			if repoOpts.Repo.Username != "" && repoOpts.Repo.Password == "" {
				repoOpts.Repo.Password = cli.PromptPassword(repoOpts.Repo.Password)
			}
			file, err := gitrepo.ReadDefaultConfig()
			if err != nil {
				return err
			}
			defer argocdio.Close(file)
			repoCtxCfg, err := gitrepo.LoadRepoCfg(file)
			if err != nil {
				return fmt.Errorf("%v: %w", gitrepo.ErrLoadCfgFile, err)
			}
			if gitrepo.AliasExists(repoCtxCfg.Repos, alias) {
				fmt.Printf("alias already exists")
				return
			}
			if repoCtxCfg.Default.Url != "" && alias == gitrepo.RepoDefaultCtx {
				err = fmt.Errorf("default alias already exists")
				return
			}
			repoAccessReq := repository.RepoAccessQuery{
				Repo:     repoOpts.Repo.Repo,
				Username: repoOpts.Repo.Username,
				Password: repoOpts.Repo.Password,
			}
			ctx := context.Background()
			_, err = client.ValidateAccess(ctx, &repoAccessReq)
			errors.CheckError(err)
			repoCreateReq := repository.RepoCreateRequest{
				Repo:   &repoOpts.Repo,
				Upsert: repoOpts.Upsert,
			}
			createdRepo, err := client.CreateRepository(ctx, &repoCreateReq)
			errors.CheckError(err)
			repoCtx := gitrepo.RepoCtx{
				Url:   createdRepo.Repo,
				Alias: alias,
			}
			repoCtxCfg.Repos = append(repoCtxCfg.Repos, repoCtx)
			if repoCtx.Alias == gitrepo.RepoDefaultCtx {
				repoCtxCfg.Default = repoCtx
			}
			repoData, err := json.MarshalIndent(repoCtxCfg, "", "\t")
			if err != nil {
				return err
			}
			err = gitrepo.TruncateFile(file)
			if err != nil {
				return fmt.Errorf("%v: %w", gitrepo.ErrOverwriteCfg, err)
			}
			err = gitrepo.StoreRepoCfg(file, repoData)
			if err != nil {
				return fmt.Errorf("%v: %w", gitrepo.ErrOverwriteCfg, err)
			}
			fmt.Printf("Repository %s added\n", createdRepo.Repo)
			return nil
		},
	}
	command.Flags().StringVar(&userName, "user", "", "username for the repository configuration")
	command.Flags().StringVar(&password, "password", "", "password of the user")
	command.Flags().StringVar(&alias, "alias", gitrepo.RepoDefaultCtx, "alias for the git repository")
	_ = command.MarkFlagRequired("user")
	return command
}
