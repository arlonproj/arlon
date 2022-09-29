package verify

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/argoproj/pkg/kube/cli"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	ErrKubectlInstall    = errors.New("kubectl is not installed or missing in your $PATH")
	ErrKubeConfigMissing = errors.New("KUBECONFIG must be set")
	ErrKubectlSet        = errors.New("error setting kubeconfig")
	ErrKcPermission      = errors.New("set the kubeconfig or kubeconfig does not have required access")
	ErrArgoCD            = errors.New("argocd is not installed or missing in your $PATH")
	ErrArgoCDAuthToken   = errors.New("argocd auth token has expired, login to argocd again")
	ErrGit               = errors.New("git is not installed or missing in your $PATH")
	ErrArlonNs           = errors.New("arlon is not installed or missing in your $PATH")
	ErrCapi              = errors.New("capi services are not installed or missing in your $PATH")
	ErrNs                = errors.New("failed to get the namespace")
	Yellow               = color.New(color.FgHiYellow).SprintFunc()
	Green                = color.New(color.FgGreen).SprintFunc()
	Red                  = color.New(color.FgRed).SprintFunc()
)

func NewCommand() *cobra.Command {
	var clientConfig clientcmd.ClientConfig
	cmd := &cobra.Command{
		Use:               "verify",
		Short:             "Verify if arlon cli can run",
		Long:              "Verify if required kubectl,argocd,git access is present before profiles and bundles are created",
		DisableAutoGenTag: true,
		Example:           "arlonctl verify",
		RunE: func(c *cobra.Command, args []string) error {
			err := verify(clientConfig)
			if err != nil {
				return err
			}
			return nil
		},
	}
	clientConfig = cli.AddKubectlFlagsToCmd(cmd)
	return cmd
}

func verify(clientConfig clientcmd.ClientConfig) error {
	var err error
	config, err := clientConfig.ClientConfig()
	if err != nil {
		return fmt.Errorf("failed to get k8s client config")
	}

	kubeconfigPath := os.Getenv("KUBECONFIG")

	// Verify kubectl status
	kubectlStatus, err := verifyKubectl(kubeconfigPath)
	if err != nil {
		fmt.Println(Red("x ")+"Error while verifying kubectl status: ", err)
	} else {
		fmt.Println("Successfully verified kubectl status")
	}

	// Verify argocd status
	argoStatus, err := verifyArgoCD()
	if err != nil {
		fmt.Println(Red("x ")+"Error while verifying argocd status: ", err)
	} else {
		fmt.Println("Successfully verified argocd status")
	}

	// Verify git status
	gitStatus, err := verifyGit()
	if err != nil {
		fmt.Println(Red("x ")+"Error while verifying git status: ", err)
	} else {
		fmt.Println("Successfully verified git status")
	}

	// Verify capi status
	capiStatus, err := verifyCapi(config)
	if err == ErrCapi {
		fmt.Println(Red("x ")+"Error while verifying capi services status ", err)
	} else {
		fmt.Println("Successfully verified capi status")
	}

	// Verify arlon status
	arlonStatus, err := verifyArlon(config)
	if err != nil {
		fmt.Println(Red("x ")+"Error while verifying arlon status: ", err)
	} else {
		fmt.Println("Successfully verified arlon namespace is present")
	}

	fmt.Println()
	if kubectlStatus && argoStatus && gitStatus && arlonStatus && capiStatus {
		fmt.Println(Green("✓") + " All requirements are installed")
	} else {
		fmt.Println("The check for Arlon prerequisites failed. Please install the missing tool(s).")
	}
	return nil
}

// Check if kubectl is installed and the kubeconfig is pointing to the correct kubeconfig
func verifyKubectl(kubeconfigPath string) (bool, error) {
	_, err := exec.LookPath("kubectl")
	if err != nil {
		return false, ErrKubectlInstall
	}

	if len(kubeconfigPath) == 0 {
		return false, ErrKubeConfigMissing
	}

	//Check if kubeconfig is correct and kubectl commands are functional
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return false, ErrKubectlSet
	}
	errKcPermission := checkNamespace(config, "kube-system")
	if errKcPermission != nil {
		return false, ErrKcPermission
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
	if err == ErrNs {
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

	return true, nil

}

// Function to check for a particular namespace
func checkNamespace(config *restclient.Config, namespace string) error {
	kubeClient := kubernetes.NewForConfigOrDie(config)
	_, err := kubeClient.CoreV1().Namespaces().Get(context.TODO(), namespace, metav1.GetOptions{})
	if err != nil {
		return ErrNs
	}
	return nil
}
