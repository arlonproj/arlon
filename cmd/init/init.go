package init

import (
	"fmt"
	"github.com/argoproj/argo-cd/v2/util/cli"
	"github.com/arlonproj/arlon/pkg/argocd"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var argocdGitTag string

const argocdManifestURL = "https://raw.githubusercontent.com/argoproj/argo-cd/%s/manifests/install.yaml"

func NewCommand() *cobra.Command {
	var argoCfgPath string
	var cliConfig clientcmd.ClientConfig
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Run the init command",
		Long:  "Run the init command",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := cliConfig.ClientConfig()
			if err != nil {
				return err
			}
			client := kubernetes.NewForConfigOrDie(cfg)
			_, err = argocd.NewArgocdClient(argoCfgPath)
			if err != nil {
				fmt.Println("Cannot initialize argocd client. Argocd may not be installed")
				// prompt for a message and proceed
				canInstallArgo := cli.AskToProceed("argo-cd not found, possibly not installed. Proceed to install? [y/n]")
				if canInstallArgo {
					downloadLink := fmt.Sprintf(argocdManifestURL, argocdGitTag)

				}
			}
		},
	}
	cmd.Flags().StringVar(&argoCfgPath, "argo-cfg", "", "Path to argocd configuration file")
	return cmd
}
