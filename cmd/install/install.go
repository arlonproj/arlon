package install

import (
	"errors"
	"fmt"
	"os/exec"
	"runtime"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	ErrKubectlInstall  = errors.New("kubectl is not installed")
	ErrKubeconfigNs    = errors.New("set the kubeconfig or kubeconfig does not have required permissions")
	ErrArgoCD          = errors.New("argocd is not installed")
	ErrArgoCDAuthToken = errors.New("argocd auth token has expired, login to argocd again")
	ErrGit             = errors.New("git is not installed")
	ErrKubectlPresent  = errors.New("kubectl is already installed")
	ErrGitPresent      = errors.New("git is already installed")
	ErrArgoCDPresent   = errors.New("argocd is already present")
	ErrKubectlFail     = errors.New("error installing kubectl")
	ErrArgoCDFail      = errors.New("error installing argocd")
	kubectlPath        = "/usr/local/bin/kubectl"
	argocdPath         = "/usr/local/bin/argocd"
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
		Example:           "arlonctl install",
		RunE: func(c *cobra.Command, args []string) error {
			var err error
			// Install kubectl and point it to the kubeconfig
			_, err = installKubectl()
			if err == ErrKubectlPresent {
				fmt.Println(Yellow("! ") + "kubectl is already present in the path")
			} else if err != nil {
				fmt.Println(Red("x ")+"Error while installing kubectl ", err)
			} else {
				fmt.Println(Green("✓") + "Successfully installed kubectl")
			}

			fmt.Println()
			_, err = verifyGit()
			if err == ErrGitPresent {
				fmt.Println("git is already present in the path")
			} else {
				fmt.Println(Yellow("! ") + "Install git cli")
			}

			fmt.Println()
			_, err = installArgoCD()
			if err == ErrArgoCDPresent {
				fmt.Println(Yellow("! ") + "argocd is already present in the path")
			} else if err != nil {
				fmt.Println(Red("x ")+"Error while installing argocd ", err)
			} else {
				fmt.Println(Green("✓") + "Successfully installed argocd")
			}

			return nil
		},
	}
	return command
}

// Check if kubectl is installed and if not then install kubectl
func installKubectl() (bool, error) {
	_, err := exec.LookPath("kubectl")
	if err != nil {
		fmt.Println("kubectl is not installed")
		fmt.Println(" → Proceeding to install kubectl")
		errInstallKubectl := installKubectlPlatform()
		if errInstallKubectl != nil {
			fmt.Println(" → Error installing kubectl")
			return false, ErrKubectlFail
		} else {
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
	_, err := exec.LookPath("argocd")
	if err != nil {
		fmt.Println("argocd cli is not installed")
		fmt.Println(" → Proceeding to install argocd")
		errInstallArgocd := installArgoCDPlatform()
		if errInstallArgocd != nil {
			fmt.Println(" → Error installing argocd")
			return false, ErrArgoCDFail
		} else {
			return true, nil
		}

	}
	return true, ErrArgoCDPresent
}

// Check the platform and on the basis of that install kubectl
func installKubectlPlatform() error {
	osPlatform := runtime.GOOS
	fmt.Println(" → Installing kubectl")
	switch osPlatform {
	case "darwin":
		err1 := downloadKubectlLatest(osPlatform)
		if err1 != nil {
			fmt.Println(" → Error installing the latest kubectl version")
		}
		_, err2 := exec.Command("chmod", "+x", kubectlPath).Output()
		if err2 != nil {
			fmt.Println(" → Error giving access to kubectl")
			return err2
		}

	case "windows":
		err1 := downloadKubectlLatest(osPlatform)
		if err1 != nil {
			fmt.Println(" → Error installing the latest kubectl version")
		}
		fmt.Println(" → Add kubectl binary to your windows path")

	case "linux":
		err1 := downloadKubectlLatest(osPlatform)
		if err1 != nil {
			fmt.Println(" → Error installing the latest kubectl version")
		}
		_, err2 := exec.Command("chmod", "+x", kubectlPath).Output()
		if err2 != nil {
			fmt.Println(" → Error giving access to kubectl")
			return err2
		}
	}
	return nil
}

// Downloads the latest version of kubectl
func downloadKubectlLatest(osPlatform string) error {
	latestVersion := "https://storage.googleapis.com/kubernetes-release/release/stable.txt"
	ver, err := exec.Command("curl", "-sL", latestVersion).Output()
	if err != nil {
		fmt.Println(" → Error fetching latest kubectl version")
		return err
	}
	var downloadKubectl string
	if osPlatform == "windows" {
		downloadKubectl = "https://storage.googleapis.com/kubernetes-release/release/" + string(ver) + "/bin/" + osPlatform + "/amd64/kubectl.exe"
	} else {
		downloadKubectl = "https://storage.googleapis.com/kubernetes-release/release/" + string(ver) + "/bin/" + osPlatform + "/amd64/kubectl"
	}
	_, err1 := exec.Command("curl", "-o", kubectlPath, "-LO", downloadKubectl).Output()
	if err1 != nil {
		fmt.Println(" → Error downloading latest kubectl version")
		return err1
	}
	return nil
}

// Check the platform and on the basis of that install argocd
func installArgoCDPlatform() error {
	osPlatform := runtime.GOOS
	fmt.Println(" → Installing argocd")
	switch osPlatform {
	case "darwin":
		err1 := downloadArgoCDLatest(osPlatform)
		if err1 != nil {
			fmt.Println(" → Error installing the latest argocd version")
		}
		_, err2 := exec.Command("chmod", "+x", argocdPath).Output()
		if err2 != nil {
			fmt.Println(" → Error giving access to argocd")
			return err2
		}

	case "windows":
		err1 := downloadArgoCDLatest(osPlatform)
		if err1 != nil {
			fmt.Println(" → Error installing the latest argocd version")
		}
		fmt.Println(" → Add argocd binary to your windows path")

	case "linux":
		err1 := downloadArgoCDLatest(osPlatform)
		if err1 != nil {
			fmt.Println(" → Error installing the latest argocd version")
		}
		_, err2 := exec.Command("chmod", "+x", argocdPath).Output()
		if err2 != nil {
			fmt.Println(" → Error giving access to argocd")
			return err2
		}
	}
	return nil
}

// Downloads the latest version of argocd
func downloadArgoCDLatest(osPlatform string) error {
	var downloadArgoCD string
	if osPlatform == "windows" {
		downloadArgoCD = "https://github.com/argoproj/argo-cd/releases/download/v2.2.12/argocd-" + osPlatform + "-amd64.exe"
	} else {
		downloadArgoCD = "https://github.com/argoproj/argo-cd/releases/download/v2.2.12/argocd-" + osPlatform + "-amd64"
	}
	_, err1 := exec.Command("curl", "-o", argocdPath, "-LO", downloadArgoCD).Output()
	if err1 != nil {
		fmt.Println(" → Error downloading latest kubectl version")
		return err1
	}
	return nil
}
