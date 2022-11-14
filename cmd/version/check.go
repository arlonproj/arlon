package version

import (
	"fmt"
	"github.com/blang/semver"
	"github.com/spf13/cobra"
	"io"
	"net/http"
	"strings"
	"time"
)

func check() *cobra.Command {
	return &cobra.Command{
		Use:   "check",
		Short: "Check for a patch for the CLI",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := http.Client{Timeout: time.Minute * 3}
			reqUrl := fmt.Sprintf("https://raw.githubusercontent.com/arlonproj/arlon/v%s/version", strings.Join(strings.Split(cliVersion, ".")[:2], "."))
			req, err := http.NewRequest(http.MethodGet, reqUrl, nil)
			if err != nil {
				return err
			}
			res, err := client.Do(req)
			if err != nil {
				return err
			}
			defer func(Body io.ReadCloser) {
				_ = Body.Close()
			}(res.Body)
			version, err := io.ReadAll(res.Body)
			if err != nil {
				return err
			}
			latestVersion := strings.TrimSpace(string(version))
			ver, err := semver.Parse(latestVersion)
			if err != nil {
				return err
			}
			currentVersion, err := semver.Parse(cliVersion)
			if err != nil {
				return err
			}
			if ver.Compare(currentVersion) == 1 {
				fmt.Printf("CLI version %s is outdated. New patch %s available\n", cliVersion, ver.String())
				return nil
			}
			fmt.Printf("CLI version %s Upto date\n", cliVersion)
			return nil
		},
	}
}
