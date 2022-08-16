package gitrepo

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	cmdutil "github.com/argoproj/argo-cd/v2/cmd/util"
	"github.com/argoproj/argo-cd/v2/pkg/apiclient/repository"
	"github.com/argoproj/argo-cd/v2/util/cli"
	"github.com/argoproj/argo-cd/v2/util/errors"
	argocdio "github.com/argoproj/argo-cd/v2/util/io"
	"github.com/argoproj/argo-cd/v2/util/localconfig"
	"github.com/arlonproj/arlon/pkg/argocd"
	"github.com/spf13/cobra"
)

type RepoCtx struct {
	Url   string `json:"url,omitempty"`
	Alias string `json:"alias,omitempty"`
}

type RepoCtxCfg struct {
	Current RepoCtx   `json:"current,omitempty"`
	Repos   []RepoCtx `json:"repos,omitempty"`
}

func register() *cobra.Command {
	var (
		repoUrl     string
		userName    string
		password    string
		alias       string
		repoCtxFile = "repoctx"
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
			repoCtx := RepoCtx{
				Url:   createdRepo.Repo,
				Alias: alias,
			}
			cfgDir, err := localconfig.DefaultConfigDir()
			if err != nil {
				err = fmt.Errorf("cannot open config file %s, error: %w", cfgDir, err)
				return
			}
			cfgFile := filepath.Join(cfgDir, repoCtxFile)
			file, err := os.OpenFile(cfgFile, os.O_RDWR|os.O_CREATE, 0666)
			if err != nil {
				err = fmt.Errorf("cannot open config file, error: %w", err)
				return
			}
			defer func(f *os.File) {
				err := f.Close()
				if err != nil {
					fmt.Printf("failed to close config file, error: %v\n", err)
				}
			}(file)
			content, err := io.ReadAll(file)
			if err != nil {
				err = fmt.Errorf("cannot read config file, error: %w", err)
				return
			}
			var repoCtxCfg RepoCtxCfg
			if len(content) > 0 {
				if err = json.Unmarshal(content, &repoCtxCfg); err != nil {
					err = fmt.Errorf("cannot read config file, error: %w", err)
					return
				}
			}
			repoCtxCfg.Repos = append(repoCtxCfg.Repos, repoCtx)
			if repoCtxCfg.Current.Url == "" {
				repoCtxCfg.Current = repoCtx
			}
			repoData, err := json.MarshalIndent(repoCtxCfg, "", "\t")
			if err != nil {
				return fmt.Errorf("cannot serialize repo context, error: %w", err)
			}
			if err := file.Truncate(0); err != nil {
				return fmt.Errorf("cannot overwrite config file, error: %w", err)
			}
			if _, err := file.Seek(0, 0); err != nil {
				return fmt.Errorf("cannot overwrite config file, error: %w", err)
			}
			_, err = file.Write(repoData)
			if err != nil {
				return err
			}
			fmt.Printf("Repository %s added\n", createdRepo.Repo)
			return nil
		},
	}
	command.Flags().StringVar(&userName, "user", "", "username for the repository configuration")
	command.Flags().StringVar(&password, "password", "", "password of the user")
	command.Flags().StringVar(&alias, "alias", "default", "alias for the git repository")
	command.MarkFlagRequired("user")
	return command
}
