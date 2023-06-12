package install

import (
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/argoproj/argo-cd/v2/util/cli"
	"github.com/arlonproj/arlon/pkg/install"
	"github.com/arlonproj/arlon/pkg/log"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	clusterctlv1 "sigs.k8s.io/cluster-api/cmd/clusterctl/api/v1alpha3"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/client"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/client/cluster"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/client/config"
)

const (
	defaultKubectlPath = "/usr/local/bin/kubectl"
	defaultArgocdPath  = "/usr/local/bin/argocd"
)

// by default, we install aws and docker. We also have --infrastructure flag to take user input
var (
	ErrKubectlPresent     = errors.New("kubectl is already present in PATH")
	ErrInvalidKubectlPath = errors.New("directory of specified kubectl path not in PATH")
	ErrGitPresent         = errors.New("git is already installed")
	ErrArgoCDPresent      = errors.New("argocd is already present in PATH")
	ErrInvalidArgoCDPath  = errors.New("directory of specified argocd path not in PATH")
	ErrKubectlFail        = errors.New("error installing kubectl")
	ErrArgoCDFail         = errors.New("error installing argocd")
	ErrCurlMissing        = errors.New("please install curl and set it in path")
	kubectlPath           string
	argocdPath            string
	kubeconfigPath        string
	Yellow                = color.New(color.FgHiYellow).SprintFunc()
	Green                 = color.New(color.FgGreen).SprintFunc()
	Red                   = color.New(color.FgRed).SprintFunc()
	capiCoreProvider      string
	infraProviders        []string
	bootstrapProviders    []string
)

func NewCommand() *cobra.Command {
	var (
		toolsOnly bool
		capiOnly  bool
		noConfirm bool
	)
	command := &cobra.Command{
		Use:               "install",
		Short:             "Install required tools for Arlon",
		Long:              "Install kubectl, Argocd cli, check git cli, and install compatible CAPI version on the management cluster",
		DisableAutoGenTag: true,
		Example: `arlon install --kubectlPath <string> --argocdPath <string> --kubeconfigPath /path/to/kubeconfig # install CAPI core provider and CLI tools, with the given argocd path and kubeconfig path
				  arlon install # installs CLI tools and CAPI
				  arlon install --tools-only # installs only the CLI tools
				  arlon install --capi-only # installs only CAPI
				  arlon install --capi-only --infrastructure aws # installs CAPA provider on the management cluster
		`,
		RunE: func(c *cobra.Command, args []string) error {
			// Install kubectl and point it to the kubeconfig
			isCapiOnly, _ := strconv.ParseBool(c.Flag("capi-only").Value.String())
			isToolsOnly, _ := strconv.ParseBool(c.Flag("tools-only").Value.String())
			if !isToolsOnly && !isCapiOnly { // none of these flags were set, so set to true and install both
				isToolsOnly = true
				isCapiOnly = true
			}
			if isToolsOnly {
				installCLITools()
			}
			fmt.Println()
			if isCapiOnly {
				fmt.Printf("Attempting to install %s with infrastructure providers %v and bootstrap providers %v\n", capiCoreProvider, infraProviders, bootstrapProviders)
				if err := installCAPI(capiCoreProvider, infraProviders, bootstrapProviders, noConfirm); err != nil {
					return err
				}
				fmt.Printf("%s CAPI is installed...\n", Green("✓"))
			}
			return nil
		},
	}
	command.Flags().BoolVarP(&noConfirm, "no-confirm", "y", false, "this flag disables prompts, all prompts are assumed to be answered as \"yes\"")
	command.Flags().StringVar(&kubectlPath, "kubectlPath", defaultKubectlPath, "kubectl download location")
	command.Flags().StringVar(&argocdPath, "argocdPath", defaultArgocdPath, "argocd download location")
	command.Flags().StringVar(&kubeconfigPath, "kubeconfigPath", "", "kubeconfig path for the management cluster")
	command.Flags().StringSliceVar(&infraProviders, "infrastructure", []string{"aws", "docker"}, "comma separated list of infrastructure provider components to install alongside CAPI")
	command.Flags().StringSliceVar(&bootstrapProviders, "bootstrap", nil, "bootstrap provider components to add to the management cluster")
	command.Flags().BoolVarP(&toolsOnly, "tools-only", "t", false, "set this flag to install only CLI tools")
	command.Flags().BoolVarP(&capiOnly, "capi-only", "c", false, "set this flag to install only CAPI on the management cluster")
	command.MarkFlagsMutuallyExclusive("tools-only", "capi-only")
	return command
}

func installCLITools() {
	err := installKubectl()
	if err == ErrKubectlPresent {
		fmt.Println(Green("✓") + " kubectl is already present in PATH")
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
	err = installArgoCD()
	if err == ErrArgoCDPresent {
		fmt.Println(Green("✓") + " argocd is already present in PATH")
	} else if err != nil {
		fmt.Println(Red("x ")+"Error while installing argocd ", err)
	} else {
		fmt.Println(Green("✓") + " Successfully installed argocd")
	}
}

// Check if kubectl is installed and if not then install kubectl
func installKubectl() error {
	var err error
	logger := log.GetLogger()
	_, err = exec.LookPath("kubectl")
	if err == nil {
		return ErrKubectlPresent
	}

	_, err = exec.LookPath(kubectlPath)
	if err != nil {
		fmt.Println(Yellow("! ") + "kubectl is not installed")
		errInstallKubectl := installKubectlPlatform()
		if errInstallKubectl != nil {
			return ErrKubectlFail
		} else {
			logger.V(1).Info("Successfully installed kubectl at ", kubectlPath)
		}
	}
	// Ensure existing or downloaded program in PATH
	_, err = exec.LookPath("kubectl")
	if err != nil {
		return ErrInvalidKubectlPath
	}
	return ErrKubectlPresent
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
func installArgoCD() error {
	logger := log.GetLogger()
	_, err := exec.LookPath("argocd")
	if err == nil {
		return ErrArgoCDPresent
	}

	_, err = exec.LookPath(argocdPath)
	if err != nil {
		fmt.Println(Yellow("! ") + "argocd cli is not installed")
		errInstallArgocd := installArgoCDPlatform()
		if errInstallArgocd != nil {
			fmt.Println(" → Error installing argocd")
			return ErrArgoCDFail
		} else {
			logger.V(1).Info("Successfully installed argocd at ", argocdPath)
		}
	}
	// Make sure the downloaded (or already present) program is in PATH
	_, err = exec.LookPath("argocd")
	if err == nil {
		return ErrInvalidArgoCDPath
	}
	return ErrArgoCDPresent
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
		_, err = exec.Command("sudo", "chmod", "+x", kubectlPath).Output()
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
	_, err = exec.Command("sudo", "curl", "-o", kubectlPath, "-LO", downloadKubectl).Output()
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
		_, err = exec.Command("sudo", "chmod", "+x", argocdPath).Output()
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
	argocdVersion := "v2.4.18"
	if osPlatform == "windows" {
		downloadArgoCD = "https://github.com/argoproj/argo-cd/releases/download/" + argocdVersion + "/argocd-" + osPlatform + "-amd64.exe"
	} else {
		downloadArgoCD = "https://github.com/argoproj/argo-cd/releases/download/" + argocdVersion + "/argocd-" + osPlatform + "-amd64"
	}
	_, err := exec.Command("sudo", "curl", "-o", argocdPath, "-LO", downloadArgoCD).Output()
	if err != nil {
		return err
	}
	return nil
}

func installCAPI(coreProviderVersion string, infrastructureProviders, bootstrapProviders []string, noConfirm bool) error {
	var providerNames []string
	for _, provider := range infrastructureProviders {
		providerName := strings.Split(provider, ":")
		providerNames = append(providerNames, providerName[0])
	}
	for _, name := range providerNames {
		installer, err := install.NewInstallerService(name, noConfirm)
		if err != nil {
			return err
		}
		if err := installer.EnsureRequisites(); err != nil {
			var errBootstrap *install.ErrBootstrap
			if errors.As(err, &errBootstrap) {
				if errBootstrap.HardFail {
					return errBootstrap
				}
				if !errBootstrap.HardFail {
					if !cli.AskToProceed(fmt.Sprintf("%s failed to perform bootstrap steps for %s\n. Continue(CAPI provider may not succeed)?[y/n]", Yellow("Warning"), name)) {
						return errBootstrap
					}
				}
			} else {
				return err
			}
		}
		if err := installer.Bootstrap(); err != nil {
			var errBootstrap *install.ErrBootstrap
			if errors.As(err, &errBootstrap) {
				if errBootstrap.HardFail {
					return errBootstrap
				}
				if !errBootstrap.HardFail {
					if !cli.AskToProceed(fmt.Sprintf("%s failed to perform bootstrap steps for %s\n. Continue(CAPI provider may not succeed)?[y/n]", Yellow("Warning"), name)) {
						return errBootstrap
					}
					continue
				}
			}
			return err
		}
	}
	c, err := client.New("")
	if err != nil {
		return err
	}
	options := client.InitOptions{
		Kubeconfig:              client.Kubeconfig{Path: kubeconfigPath},
		BootstrapProviders:      bootstrapProviders,
		InfrastructureProviders: infrastructureProviders,
		LogUsageInstructions:    true,
		WaitProviders:           true,                 // this is set to false for clusterctl
		WaitProviderTimeout:     time.Second * 5 * 60, // this is the default for clusterctl
	}
	clientCfg, err := config.New("")
	if err != nil {
		return err
	}
	clusterClient := cluster.New(cluster.Kubeconfig(options.Kubeconfig), clientCfg)
	if isFirstRun(clusterClient) {
		options.CoreProvider = coreProviderVersion
	}
	if _, err := c.Init(options); err != nil {
		return err
	}
	return nil
}

func isFirstRun(client cluster.Client) bool {
	// From `clusterctl` source:
	// Check if there is already a core provider installed in the cluster
	// Nb. we are ignoring the error so this operation can support listing images even if there is no an existing management cluster;
	// in case there is no an existing management cluster, we assume there are no core providers installed in the cluster.
	currentCoreProvider, _ := client.ProviderInventory().GetDefaultProviderName(clusterctlv1.CoreProviderType)
	return currentCoreProvider == ""
}
