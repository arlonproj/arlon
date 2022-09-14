package install

import (
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/client"
	"time"

	"github.com/arlonproj/arlon/pkg/log"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	ErrKubectlPresent  = errors.New("kubectl is already present at default(/usr/local/bin/kubectl) location or user specifed location")
	ErrGitPresent      = errors.New("git is already installed")
	ErrArgoCDPresent   = errors.New("argocd is already present at default(/usr/local/bin/argocd) location or user specifed location")
	ErrKubectlFail     = errors.New("error installing kubectl")
	ErrArgoCDFail      = errors.New("error installing argocd")
	ErrCurlMissing     = errors.New("please install curl and set it in path")
	kubectlPath        string
	argocdPath         string
	defaultKubectlPath = "/usr/local/bin/kubectl"
	defaultArgocdPath  = "/usr/local/bin/argocd"
	kubeconfigPath     string
	capiCoreProvider   = "cluster-api:1.1.5"
	Yellow             = color.New(color.FgHiYellow).SprintFunc()
	Green              = color.New(color.FgGreen).SprintFunc()
	Red                = color.New(color.FgRed).SprintFunc()
)

func NewCommand() *cobra.Command {
	command := &cobra.Command{
		Use:               "install",
		Short:             "Install required tools for Arlon",
		Long:              "Install kubectl, Argocd cli, check git cli",
		DisableAutoGenTag: true,
		Example:           "arlon install --kubectlPath <string> --argocdPath <string>",
		RunE: func(c *cobra.Command, args []string) error {
			fmt.Println("Note: SUDO access is required to install the required tools for Arlon")
			fmt.Println()
			var err error
			// Install kubectl and point it to the kubeconfig
			_, err = installKubectl()
			if err == ErrKubectlPresent {
				fmt.Println(Green("✓") + " kubectl is already present at default(" + Red(defaultKubectlPath) + ")location or user specifed location")
			} else if err != nil {
				fmt.Println(Red("x ")+"Error while installing kubectl ", err)
			} else {
				fmt.Println(Green("✓") + " Successfully installed kubectl")
			}

			fmt.Println()
			_, err = verifyGit()
			if err == ErrGitPresent {
				fmt.Println(Green("✓") + " git is already present in the path")
			} else {
				fmt.Println(Yellow("! ") + "Install git cli")
			}

			fmt.Println()
			_, err = installArgoCD()
			if err == ErrArgoCDPresent {
				fmt.Println(Green("✓") + " argocd is already present at default(" + Red(defaultArgocdPath) + ")location or user specifed location")
			} else if err != nil {
				fmt.Println(Red("x ")+"Error while installing argocd ", err)
			} else {
				fmt.Println(Green("✓") + " Successfully installed argocd")
			}

			if err := installCAPI(capiCoreProvider); err != nil {
				return err
			}
			return nil
		},
	}
	command.Flags().StringVar(&kubectlPath, "kubectlPath", defaultKubectlPath, "kubectl download location")
	command.Flags().StringVar(&argocdPath, "argocdPath", defaultArgocdPath, "argocd download location")
	command.Flags().StringVar(&kubeconfigPath, "kubeconfig", "", "kubeconfig path for the management cluster")
	return command
}

// Check if kubectl is installed and if not then install kubectl
func installKubectl() (bool, error) {
	var err error
	l := log.GetLogger()
	_, err = exec.LookPath(defaultKubectlPath)
	if err == nil {
		return true, ErrKubectlPresent
	}

	_, err = exec.LookPath(kubectlPath)
	if err != nil {
		fmt.Println(Yellow("! ") + "kubectl is not installed")
		errInstallKubectl := installKubectlPlatform()
		if errInstallKubectl != nil {
			return false, ErrKubectlFail
		} else {
			l.V(1).Info("Successfully installed kubectl at ", kubectlPath)
			return true, nil
		}
	}
	return true, ErrKubectlPresent
}

// Check if git is installed and if not, then prompt user
func verifyGit() (bool, error) {
	_, err := exec.LookPath("git")
	if err != nil {
		return false, err
	}
	return true, ErrGitPresent
}

// Check if argocd is installed and if not, then install argocd
func installArgoCD() (bool, error) {
	var err error
	l := log.GetLogger()
	_, err = exec.LookPath(defaultArgocdPath)
	if err == nil {
		return true, ErrArgoCDPresent
	}

	_, err = exec.LookPath(argocdPath)
	if err != nil {
		fmt.Println(Yellow("! ") + "argocd cli is not installed")
		errInstallArgocd := installArgoCDPlatform()
		if errInstallArgocd != nil {
			fmt.Println(" → Error installing argocd")
			return false, ErrArgoCDFail
		} else {
			l.V(1).Info("Successfully installed argocd at ", argocdPath)
			return true, nil
		}

	}
	return true, ErrArgoCDPresent
}

// Check the platform and on the basis of that install kubectl
func installKubectlPlatform() error {
	var err error
	osPlatform := runtime.GOOS
	fmt.Println(" → Installing kubectl")
	switch osPlatform {
	case "windows":
		_, err := exec.LookPath("curl")
		if err != nil {
			return ErrCurlMissing
		}
		err = downloadKubectlLatest(osPlatform)
		if err != nil {
			fmt.Println(" → Error installing the latest kubectl version")
			return err
		}
		fmt.Println(" → Add kubectl binary to your windows path")
	default:
		err = downloadKubectlLatest(osPlatform)
		if err != nil {
			fmt.Println(" → Error installing the latest kubectl version")
			return err
		}
		_, err = exec.Command("chmod", "+x", kubectlPath).Output()
		if err != nil {
			fmt.Println(" → Error giving execute permission to kubectl")
			return err
		}
	}
	return nil
}

// Downloads the latest version of kubectl
func downloadKubectlLatest(osPlatform string) error {
	latestVersion := "https://storage.googleapis.com/kubernetes-release/release/stable.txt"
	var err error
	ver, err := exec.Command("curl", "-sL", latestVersion).Output()
	if err != nil {
		return err
	}
	var downloadKubectl string
	if osPlatform == "windows" {
		downloadKubectl = "https://storage.googleapis.com/kubernetes-release/release/" + string(ver) + "/bin/" + osPlatform + "/amd64/kubectl.exe"
	} else {
		downloadKubectl = "https://storage.googleapis.com/kubernetes-release/release/" + string(ver) + "/bin/" + osPlatform + "/amd64/kubectl"
	}
	_, err = exec.Command("curl", "-o", kubectlPath, "-LO", downloadKubectl).Output()
	if err != nil {
		return err
	}
	return nil
}

// Check the platform and on the basis of that install argocd
func installArgoCDPlatform() error {
	var err error
	osPlatform := runtime.GOOS
	fmt.Println(" → Installing argocd")
	switch osPlatform {
	case "windows":
		_, err := exec.LookPath("curl")
		if err != nil {
			return ErrCurlMissing
		}
		err = downloadArgoCD(osPlatform)
		if err != nil {
			fmt.Println(" → Error installing the latest argocd version")
		}
		fmt.Println(" → Add argocd binary to your windows path")
	default:
		err = downloadArgoCD(osPlatform)
		if err != nil {
			fmt.Println(" → Error installing the latest argocd version")
		}
		_, err = exec.Command("chmod", "+x", argocdPath).Output()
		if err != nil {
			fmt.Println(" → Error giving execute permission to argocd (ArgoCD CLI)")
			return err
		}
	}
	return nil
}

// Downloads the latest version of argocd
func downloadArgoCD(osPlatform string) error {
	var downloadArgoCD string
	argocdVersion := "v2.4.11"
	if osPlatform == "windows" {
		downloadArgoCD = "https://github.com/argoproj/argo-cd/releases/download/" + argocdVersion + "/argocd-" + osPlatform + "-amd64.exe"
	} else {
		downloadArgoCD = "https://github.com/argoproj/argo-cd/releases/download/" + argocdVersion + "/argocd-" + osPlatform + "-amd64"
	}
	_, err := exec.Command("curl", "-o", argocdPath, "-LO", downloadArgoCD).Output()
	if err != nil {
		return err
	}
	return nil
}

func installCAPI(ver string) error {
	c, err := client.New("")
	if err != nil {
		return err
	}
	options := client.InitOptions{
		Kubeconfig:              client.Kubeconfig{Path: kubeconfigPath},
		CoreProvider:            ver,
		BootstrapProviders:      nil,
		InfrastructureProviders: nil,
		ControlPlaneProviders:   nil,
		TargetNamespace:         "",
		LogUsageInstructions:    false,
		WaitProviders:           false,
		WaitProviderTimeout:     time.Second * 5 * 60, // this is the default for clusterctl
	}
	if _, err := c.Init(options); err != nil {
		return err
	}
	return nil
}
