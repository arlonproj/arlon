package argocd

import (
	"context"
	"fmt"
	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"os"
	"strings"
)

type RepoCreds struct {
	Url      string
	Username string
	Password string
}

// -----------------------------------------------------------------------------

// CloneRepo clones a git repo registered with argocd into a local repository
func CloneRepo(
	creds *RepoCreds,
	repoUrl string,
	repoBranch string,
) (repo *gogit.Repository, tmpDir string, auth *http.BasicAuth, err error) {
	auth = &http.BasicAuth{
		Username: creds.Username,
		Password: creds.Password,
	}
	tmpDir, err = os.MkdirTemp("", "arlon-")
	branchRef := plumbing.NewBranchReferenceName(repoBranch)
	repo, err = gogit.PlainCloneContext(context.Background(), tmpDir, false, &gogit.CloneOptions{
		URL:           repoUrl,
		Auth:          auth,
		RemoteName:    gogit.DefaultRemoteName,
		ReferenceName: branchRef,
		SingleBranch:  true,
		NoCheckout:    false,
		Progress:      nil,
		Tags:          gogit.NoTags,
		CABundle:      nil,
	})
	if err != nil {
		return nil, "", nil, fmt.Errorf("failed to clone repository: %s", err)
	}
	return
}

// -----------------------------------------------------------------------------

func GetRepoCredsFromArgoCd(
	kubeClient *kubernetes.Clientset,
	argocdNs string,
	repoUrl string,
) (creds *RepoCreds, err error) {
	corev1 := kubeClient.CoreV1()
	secretsApi := corev1.Secrets(argocdNs)
	opts := metav1.ListOptions{
		LabelSelector: "argocd.argoproj.io/secret-type=repository",
	}
	secrets, err := secretsApi.List(context.Background(), opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list secrets: %s", err)
	}
	for _, repoSecret := range secrets.Items {
		if strings.Compare(repoUrl, string(repoSecret.Data["url"])) == 0 {
			creds = &RepoCreds{
				Url:      string(repoSecret.Data["url"]),
				Username: string(repoSecret.Data["username"]),
				Password: string(repoSecret.Data["password"]),
			}
			break
		}
	}
	if creds == nil {
		return nil, fmt.Errorf("did not find argocd repository matching %s (did you register it?)", repoUrl)
	}
	return creds, nil
}
