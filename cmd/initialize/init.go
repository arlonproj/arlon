package initialize

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/base64"
	"fmt"
	"github.com/argoproj/argo-cd/v2/pkg/apiclient"
	"github.com/argoproj/argo-cd/v2/pkg/apiclient/account"
	"github.com/argoproj/argo-cd/v2/util/cli"
	argocdio "github.com/argoproj/argo-cd/v2/util/io"
	"github.com/argoproj/argo-cd/v2/util/localconfig"
	"github.com/arlonproj/arlon/config"
	"github.com/arlonproj/arlon/deploy"
	"github.com/arlonproj/arlon/pkg/argocd"
	gyaml "github.com/ghodss/yaml"
	"github.com/spf13/cobra"
	"io"
	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"net/http"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

const (
	argocdManifestURL              = "https://raw.githubusercontent.com/argoproj/argo-cd/%s/manifests/install.yaml"
	defaultArgoNamespace           = "argocd"
	defaultArlonNamespace          = "arlon"
	defaultArlonArgoCDUser         = "arlon"
	defaultArgoServerDeployment    = "argocd-server"
	reasonMinimumReplicasAvailable = "MinimumReplicasAvailable"
)

// this is how argocd-creds looks
//apiVersion: v1
//data:
//  config: Y29udGV4dHM6Ci0gbmFtZTogMTI3LjAuMC4xOjgwODAKICBzZXJ2ZXI6IDEyNy4wLjAuMTo4MDgwCiAgdXNlcjogMTI3LjAuMC4xOjgwODAKY3VycmVudC1jb250ZXh0OiAxMjcuMC4wLjE6ODA4MApzZXJ2ZXJzOgotIGdycGMtd2ViLXJvb3QtcGF0aDogIiIKICBpbnNlY3VyZTogdHJ1ZQogIHNlcnZlcjogMTI3LjAuMC4xOjgwODAKdXNlcnM6Ci0gYXV0aC10b2tlbjogZXlKaGJHY2lPaUpJVXpJMU5pSXNJblI1Y0NJNklrcFhWQ0o5LmV5SnBjM01pT2lKaGNtZHZZMlFpTENKemRXSWlPaUpoWkcxcGJqcHNiMmRwYmlJc0ltVjRjQ0k2TVRZMk56WXpNVFUxTkN3aWJtSm1Jam94TmpZM05UUTFNVFUwTENKcFlYUWlPakUyTmpjMU5EVXhOVFFzSW1wMGFTSTZJakkxWmpreFlqUmhMVGMwWW1JdE5HSTVZUzA0TkRRekxUSmpabUU0T0RNME5EYzRNaUo5LjJ6NjJSSGF2T0JTemVFSkR4QUcweWxWZzdKbUcxLXJqYzhNRnJUSk51cjAKICBuYW1lOiAxMjcuMC4wLjE6ODA4MAo=
//kind: Secret
//metadata:
//  creationTimestamp: "2022-11-04T07:01:08Z"
//  name: argocd-creds
//  namespace: arlon
//  resourceVersion: "27314"
//  uid: 2111435a-b679-4158-b88c-42c181b3f057
//type: Opaque

var argocdGitTag string = "release-2.4"

func NewCommand() *cobra.Command {
	var argoCfgPath string
	var cliConfig clientcmd.ClientConfig
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Run the init command",
		Long:  "Run the init command",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := cliConfig.ClientConfig()
			if err != nil {
				return err
			}
			client, err := k8sclient.New(cfg, k8sclient.Options{})
			if err != nil {
				return err
			}
			kubeClient := kubernetes.NewForConfigOrDie(cfg)

			canInstallArgo, err := canInstallArgocd()
			if err != nil {
				return err
			}
			if canInstallArgo {
				fmt.Println("Cannot initialize argocd client. Argocd may not be installed")
				shouldInstallArgo := cli.AskToProceed("argo-cd not found, possibly not installed. Proceed to install? [y/n]")
				if shouldInstallArgo {
					if err := beginArgoCDInstall(ctx, client, kubeClient); err != nil {
						return err
					}
				}
			}

			argoClient := argocd.NewArgocdClientOrDie("")
			canInstall, err := canInstallArlon(ctx, kubeClient)
			if err != nil {
				return err
			}
			if canInstall {
				fmt.Println("arlon namespace not found. Arlon controller might not be installed")
				shouldInstallArlon := cli.AskToProceed("Install arlon controller? [y/n]")
				if shouldInstallArlon {
					if err := beginArlonInstall(ctx, client, kubeClient, argoClient, defaultArlonNamespace, defaultArgoNamespace); err != nil {
						return err
					}
				}
			}
			return nil
		},
	}
	cliConfig = cli.AddKubectlFlagsToCmd(cmd)
	cmd.Flags().StringVar(&argoCfgPath, "argo-cfg", "", "Path to argocd configuration file")
	return cmd
}

func beginArlonInstall(ctx context.Context, client k8sclient.Client, kubeClient *kubernetes.Clientset, argoClient apiclient.Client, arlonNs, argoNs string) error {
	ns, err := kubeClient.CoreV1().Namespaces().Create(ctx, &v1.Namespace{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Namespace",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      arlonNs,
			Namespace: arlonNs,
		},
	}, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}
	if errors.IsAlreadyExists(err) {
		fmt.Printf("namespace %s already exists\n", arlonNs)
	} else {
		fmt.Printf("namespage %s created\n", ns.GetName())
	}
	cm, err := kubeClient.CoreV1().ConfigMaps(argoNs).Update(ctx, &v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "argocd-cm",
		},
		Immutable: nil,
		Data: map[string]string{
			"accounts.arlon": "apiKey, login",
		},
		BinaryData: nil,
	}, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	fmt.Printf("ConfigMap %s updated\n", cm.GetName())
	rbacCm, err := kubeClient.CoreV1().ConfigMaps(argoNs).Update(ctx, &v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "argocd-rbac-cm",
			Namespace: argoNs,
		},
		Data: map[string]string{
			"policy.csv": "g, arlon, role:admin",
		},
	}, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	fmt.Printf("ConfigMap %s updated\n", rbacCm.GetName())
	sec, err := createArgoCreds(ctx, kubeClient, argoClient, arlonNs, argoNs)
	if err != nil {
		return err
	}
	fmt.Printf("Secret %s created", sec.GetName())
	crds := [][]byte{
		config.CRDProfile,
		config.CRDClusterReg,
		config.CRDCallHomeConfig,
	}
	deplManifests := [][]byte{
		deploy.YAMLdeploy,
		deploy.YAMLrbacCHC,
		deploy.YAMLrbacClusterReg,
		deploy.YAMLwebhook,
	}
	decodedCrds := [][]*unstructured.Unstructured{}
	for _, crd := range crds {
		decoded, err := decodeResources(crd)
		if err != nil {
			return err
		}
		decodedCrds = append(decodedCrds, decoded)
	}
	decodedDeplManifests := [][]*unstructured.Unstructured{}
	for _, manifest := range deplManifests {
		decoded, err := decodeResources(manifest)
		if err != nil {
			return err
		}
		decodedDeplManifests = append(decodedDeplManifests, decoded)
	}

	for i := 0; i < len(decodedCrds); i++ {
		for j := 0; j < len(decodedCrds[i]); j++ {
			if err := applyObject(ctx, client, decodedCrds[i][j], arlonNs); err != nil {
				return err
			}
		}
	}

	for i := 0; i < len(decodedDeplManifests); i++ {
		for j := 0; j < len(decodedDeplManifests[i]); j++ {
			if err := applyObject(ctx, client, decodedDeplManifests[i][j], arlonNs); err != nil {
				return err
			}
		}
	}
	return nil
}

func beginArgoCDInstall(ctx context.Context, client k8sclient.Client, kubeClient *kubernetes.Clientset) error {
	downloadLink := fmt.Sprintf(argocdManifestURL, argocdGitTag)
	err := client.Create(ctx, &v1.Namespace{
		TypeMeta: metav1.TypeMeta{
			Kind: "Namespace",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: defaultArgoNamespace,
		},
	})
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}
	if err := installArgo(downloadLink, client); err != nil {
		return err
	}
	err = wait.PollImmediate(time.Second*10, time.Minute*5, func() (bool, error) {
		fmt.Printf("waiting for argocd-server")
		var deployment *apps.Deployment
		d, err := kubeClient.AppsV1().Deployments(defaultArgoNamespace).Get(ctx, defaultArgoServerDeployment, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		deployment = d
		condition := getDeploymentCondition(deployment.Status, apps.DeploymentAvailable)
		return condition != nil && condition.Reason == reasonMinimumReplicasAvailable, nil
	})
	if err != nil {
		return err
	}
	return nil
}

func canInstallArgocd() (bool, error) {
	return true, nil
}

func canInstallArlon(ctx context.Context, kubeClient *kubernetes.Clientset) (bool, error) {
	if _, err := kubeClient.CoreV1().Namespaces().Get(ctx, defaultArlonNamespace, metav1.GetOptions{}); err != nil {
		if errors.IsNotFound(err) {
			return true, nil
		}
	}
	return false, nil
}

func installArgo(downloadLink string, client k8sclient.Client) error {
	manifest, err := downloadManifest(downloadLink)
	if err != nil {
		return err
	}
	resources, err := decodeResources(manifest)
	if err != nil {
		return err
	}
	for _, obj := range resources {
		err := applyObject(context.Background(), client, obj, defaultArgoNamespace)
		if err != nil {
			return err
		}
	}
	return nil
}

func decodeResources(manifest []byte) ([]*unstructured.Unstructured, error) {
	var resources []*unstructured.Unstructured
	decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(manifest), 4096)
	for {
		resource := unstructured.Unstructured{}
		err := decoder.Decode(&resource)
		if err == nil {
			resources = append(resources, &resource)
		} else if err == io.EOF {
			break
		} else {
			return nil, err
		}
	}
	return resources, nil
}

func applyObject(ctx context.Context, client k8sclient.Client, object *unstructured.Unstructured, namespace string) error {
	name := object.GetName()
	object.SetNamespace(namespace)
	if name == "" {
		return fmt.Errorf("object %s has no name", object.GroupVersionKind().String())
	}
	groupVersionKind := object.GroupVersionKind()
	objDesc := fmt.Sprintf("(%s) %s/%s", groupVersionKind.String(), namespace, name)
	err := client.Create(ctx, object)
	if err != nil {
		if errors.IsAlreadyExists(err) {
			fmt.Printf("%s already exists\n", objDesc)
			return nil
		}
		return fmt.Errorf("could not create %s. Error: %v", objDesc, err.Error())
	}
	fmt.Printf("successfully created %s", objDesc)
	return nil
}

func downloadManifest(link string) ([]byte, error) {
	client := http.Client{
		Timeout: 30 * time.Second,
	}
	res, err := client.Get(link)
	if err != nil {
		return nil, err
	}
	defer argocdio.Close(res.Body)
	respBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	return respBytes, nil
}

func getDeploymentCondition(status apps.DeploymentStatus, condType apps.DeploymentConditionType) *apps.DeploymentCondition {
	for i := range status.Conditions {
		c := status.Conditions[i]
		if c.Type == condType {
			return &c
		}
	}
	return nil
}

func createArgoCreds(ctx context.Context, clientset *kubernetes.Clientset, argoClient apiclient.Client, arlonNamespace string, argoNamespace string) (*v1.Secret, error) {
	conn, accountClient := argoClient.NewAccountClientOrDie()
	defer argocdio.Close(conn)
	res, err := accountClient.CreateToken(ctx, &account.CreateTokenRequest{
		Name:      defaultArlonArgoCDUser,
		ExpiresIn: 0,
	})
	if err != nil {
		return nil, err
	}
	defaultInClusterUser := fmt.Sprintf("%s.%s.svc.cluster.local", defaultArgoServerDeployment, argoNamespace)
	argoCfg := localconfig.LocalConfig{
		CurrentContext: "",
		Contexts: []localconfig.ContextRef{
			{
				Name:   defaultInClusterUser,
				Server: defaultInClusterUser,
				User:   defaultInClusterUser,
			},
		},
		Servers: []localconfig.Server{
			{
				Server:          defaultInClusterUser,
				Insecure:        true,
				GRPCWebRootPath: "",
			},
		},
		Users: []localconfig.User{
			{
				Name:      defaultInClusterUser,
				AuthToken: res.GetToken(),
			},
		},
	}
	out, err := gyaml.Marshal(argoCfg)
	if err != nil {
		return nil, err
	}
	b64ArgoSecret := make([]byte, base64.StdEncoding.EncodedLen(len(out)))
	base64.StdEncoding.Encode(b64ArgoSecret, out)
	secret := v1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "argocd-creds",
			Namespace: arlonNamespace,
		},
		Data: nil,
		Type: v1.SecretTypeOpaque,
	}
	secret.Data = map[string][]byte{
		"config": b64ArgoSecret,
	}
	created, err := clientset.CoreV1().Secrets(arlonNamespace).Create(ctx, &secret, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	return created, nil
}
