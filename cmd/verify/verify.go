package verify

import (
	"context"
	"errors"
	"fmt"
	"os/exec"

	"github.com/argoproj/pkg/kube/cli"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	ErrKubectlInstall    = errors.New("kubectl is not installed or missing in your $PATH")
	ErrKubectlSet        = errors.New("error setting kubeconfig")
	ErrClusterInfo       = errors.New("set the kubeconfig or kubeconfig does not have required access")
	ErrArgoCD            = errors.New("argocd is not installed or missing in your $PATH")
	ErrArgoCDAuthToken   = errors.New("argocd auth token has expired, login to argocd again")
	ErrGit               = errors.New("git is not installed or missing in your $PATH")
	ErrArlonNs           = errors.New("arlon is not installed or missing in your $PATH")
	ErrCapi              = errors.New("capi services are not installed or missing in your $PATH")
	ErrCapiCP            = errors.New("error fetching the capi cloudproviders")
	ErrNs                = errors.New("failed to get the namespace")
	ErrListNs            = errors.New("error listing namespaces")
	ErrKubeconfigMissing = errors.New("enter kubeconfig using arlon verify --kubeconfigPath")
)

func NewCommand() *cobra.Command {
	var clientConfig clientcmd.ClientConfig
	var kubeconfigPath string
	command := &cobra.Command{
		Use:               "verify",
		Short:             "Verify if arlon cli can run",
		Long:              "Verify if required kubectl,argocd,git access is present before profiles and bundles are created",
		DisableAutoGenTag: true,
		Example:           "arlonctl verify --kubeconfigPath",
		RunE: func(c *cobra.Command, args []string) error {
			config, err := clientConfig.ClientConfig()
			if err != nil {
				return fmt.Errorf("failed to get k8s client config")
			}

			// Verify kubectl status
			kubectlStatus, err := verifyKubectl(kubeconfigPath)
			if err != nil {
				fmt.Println("Error while verifying kubectl status: ", err)
			} else {
				fmt.Println("Successfully verified kubectl status")
			}

			// Verify argocd status
			argoStatus, err := verifyArgoCD()
			if err != nil {
				fmt.Println("Error while verifying argocd status: ", err)
			} else {
				fmt.Println("Successfully verified argocd status")
			}

			// Verify git status
			gitStatus, err := verifyGit()
			if err != nil {
				fmt.Println("Error while verifying git status: ", err)
			} else {
				fmt.Println("Successfully verified git status")
			}

			// Verify capi status
			capiStatus, err := verifyCapi(config)
			if err == ErrCapiCP {
				fmt.Println("Error while verifying capi cloudprovider status: ", err)
			} else if err == ErrCapi {
				fmt.Println("Error while verifying capi services status ", err)
			} else {
				fmt.Println("Successfully verified capi status")
			}

			// Verify arlon status
			arlonStatus, err := verifyArlon(config)
			if err != nil {
				fmt.Println("Error while verifying arlon status: ", err)
			} else {
				fmt.Println("Successfully verified arlon status")
			}

			fmt.Println()
			if kubectlStatus && argoStatus && gitStatus && arlonStatus && capiStatus {
				fmt.Println("All requirements are installed")
			} else {
				fmt.Println("The check for Arlon prerequisites failed. Please install the missing tool(s).")
			}
			return nil
		},
	}
	clientConfig = cli.AddKubectlFlagsToCmd(command)
	command.Flags().StringVar(&kubeconfigPath, "kubeconfigPath", "kubeconfig", "kubeconfig file location")
	return command
}

// Check if kubectl is installed and the kubeconfig is pointing to the correct kubeconfig
func verifyKubectl(kubeconfigPath string) (bool, error) {
	_, err := exec.LookPath("kubectl")
	if err != nil {
		return false, ErrKubectlInstall
	}

	//Check if kubeconfig is correct and kubectl commands are functional
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return false, ErrKubectlSet
	}
	errClusterInfo := checkNamespace(config, "kube-system")
	if errClusterInfo != nil {
		return false, ErrClusterInfo
	}
	return true, nil
}

// Check if argocd cli is installed and the account has admin access
func verifyArgoCD() (bool, error) {
	// Check if argocd is installed
	_, err := exec.LookPath("argocd")
	if err != nil {
		return false, ErrArgoCD
	}

	//Check if argocd has access and auth-token is not expired
	_, errAcc := exec.Command("argocd", "account", "list").Output()
	if errAcc != nil {
		return false, ErrArgoCDAuthToken
	}
	return true, nil
}

// Check if git cli is installed
func verifyGit() (bool, error) {
	_, err := exec.LookPath("git")
	if err != nil {
		return false, ErrGit
	}
	return true, nil
}

// Verify if the arlon services are running
func verifyArlon(config *restclient.Config) (bool, error) {
	err := checkNamespace(config, "arlon")
	if err == ErrListNs {
		return false, ErrListNs
	} else if err == ErrNs {
		return false, ErrArlonNs
	}
	return true, nil
}

// Verify if capi services are running
func verifyCapi(config *restclient.Config) (bool, error) {
	errCapi := checkNamespace(config, "capi-system")
	if errCapi != nil {
		return false, ErrCapi
	}

	// Check for capa-system namespace
	errCapaCP := checkCapaCloudProvider(config)
	if errCapaCP == ErrNs {
		//Check for capz-system incase capa-system is not present
		errCapaZ := checkCapzCloudProvider(config)
		if errCapaZ == ErrNs {
			return false, ErrCapiCP
		} else if errCapaZ == ErrListNs {
			return false, ErrListNs
		}
	}

	return true, nil

}

// Function to check capa-system namespace
func checkCapaCloudProvider(config *restclient.Config) error {
	errCapa := checkNamespace(config, "capa-system")
	if errCapa != nil {
		return errCapa
	}
	return nil
}

// Function to check capz-system namespace
func checkCapzCloudProvider(config *restclient.Config) error {
	errCapz := checkNamespace(config, "capz-system")
	if errCapz != nil {
		return errCapz
	}
	return nil
}

// Function to check for a particular namespace
func checkNamespace(config *restclient.Config, namespace string) error {
	kubeClient := kubernetes.NewForConfigOrDie(config)
	corev1 := kubeClient.CoreV1()
	nsList, err := corev1.Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return ErrListNs
	}
	for _, n := range nsList.Items {
		if n.Name == namespace {
			return nil
		}
	}
	return ErrNs
}
