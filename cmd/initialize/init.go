package initialize

import (
	"bytes"
	"context"
	_ "embed"
	e "errors"
	"fmt"
	"github.com/argoproj/argo-cd/v2/cmd/argocd/commands"
	"github.com/arlonproj/arlon/pkg/argocd"

	"github.com/argoproj/argo-cd/v2/pkg/apiclient"
	"github.com/argoproj/argo-cd/v2/pkg/apiclient/account"
	"github.com/argoproj/argo-cd/v2/util/cli"
	argocdio "github.com/argoproj/argo-cd/v2/util/io"
	"github.com/argoproj/argo-cd/v2/util/localconfig"
	"github.com/arlonproj/arlon/config"
	"github.com/arlonproj/arlon/deploy"
	gyaml "github.com/ghodss/yaml"
	"github.com/spf13/cobra"
	"io"
	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
	"net/http"
	"net/url"
	"os"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
	"time"
)

const (
	argocdManifestURL              = "https://raw.githubusercontent.com/argoproj/argo-cd/%s/manifests/install.yaml"
	defaultArgoNamespace           = "argocd"
	defaultArlonNamespace          = "arlon"
	defaultArlonArgoCDUser         = "arlon"
	defaultArgoServerDeployment    = "argocd-server"
	reasonMinimumReplicasAvailable = "MinimumReplicasAvailable"
	argoInitialAdminSecret         = "argocd-initial-admin-secret"
)

type porfForwardCfg struct {
	ctx        context.Context
	hostPort   uint16
	remotePort uint16
	callback   portForwardCallBack
	restCfg    *rest.Config
	kClient    k8sclient.Client
	clientset  *kubernetes.Clientset
}

var argocdGitTag string = "release-2.4"

type portForwardCallBack func(ctx context.Context, localPort uint16) error

func NewCommand() *cobra.Command {
	var argoCfgPath string
	var cliConfig clientcmd.ClientConfig
	argoServer := "127.0.0.1:8080"
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
			//canInstallArgo, err := canInstallArgocd()
			if err != nil {
				return err
			}
			if true {
				fmt.Println("Cannot initialize argocd client. Argocd may not be installed")
				shouldInstallArgo := cli.AskToProceed("argo-cd not found, possibly not installed. Proceed to install? [y/n]")
				if shouldInstallArgo {
					if err := beginArgoCDInstall(ctx, client, kubeClient); err != nil {
						return err
					}
				}
			}
			argoCfg, err := localconfig.DefaultLocalConfigPath()
			if err != nil {
				return err
			}
			err = portForward(&porfForwardCfg{
				ctx:        ctx,
				hostPort:   8080,
				remotePort: 8080,
				callback: func() portForwardCallBack {
					return func(ctx context.Context, localPort uint16) error {
						c := commands.NewLoginCommand(&apiclient.ClientOptions{
							ConfigPath: argoCfg, // we do this because argocd needs to write the local config.
							Insecure:   true,
						})
						password, err := getArgoAdminPassword(ctx, kubeClient, defaultArgoNamespace)
						if err != nil {
							return err
						}
						_ = c.Flag("password").Value.Set(password)
						_ = c.Flag("username").Value.Set("admin")
						c.Run(cmd, []string{argoServer})
						argoClient := argocd.NewArgocdClientOrDie("")
						if err := beginArlonInstall(ctx, client, kubeClient, argoClient, defaultArlonNamespace, defaultArgoNamespace); err != nil {
							return err
						}
						return nil
					}
				}(),
				restCfg:   cfg,
				kClient:   client,
				clientset: kubeClient,
			}, defaultArgoNamespace, defaultArlonNamespace)
			if err != nil {
				return err
			}
			return nil
		},
	}
	cliConfig = cli.AddKubectlFlagsToCmd(cmd)
	cmd.Flags().StringVar(&argoCfgPath, "argo-cfg", "", "Path to argocd configuration file")
	return cmd
}

func getArgoAdminPassword(ctx context.Context, clientset *kubernetes.Clientset, argoNs string) (string, error) {
	secret, err := clientset.CoreV1().Secrets(argoNs).Get(ctx, argoInitialAdminSecret, metav1.GetOptions{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
	})
	if err != nil {
		return "", err
	}
	pass64 := secret.Data["password"]
	return string(pass64), nil
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
	argoCm, err := kubeClient.CoreV1().ConfigMaps(argoNs).Get(ctx, "argocd-cm", metav1.GetOptions{})
	if err != nil {
		return err
	}
	argoCm.Data = map[string]string{
		"accounts.arlon": "apiKey, login",
	}
	argoRbacCm, err := kubeClient.CoreV1().ConfigMaps(argoNs).Get(ctx, "argocd-rbac-cm", metav1.GetOptions{})
	if err != nil {
		return err
	}
	argoRbacCm.Data = map[string]string{
		"policy.csv": "g, arlon, role:admin",
	}
	cm, err := kubeClient.CoreV1().ConfigMaps(argoNs).Update(ctx, argoCm, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	fmt.Printf("ConfigMap %s updated\n", cm.GetName())
	rbacCm, err := kubeClient.CoreV1().ConfigMaps(argoNs).Update(ctx, argoRbacCm, metav1.UpdateOptions{})
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
	fmt.Printf("successfully created %s\n", objDesc)
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
		CurrentContext: defaultInClusterUser,
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
		"config": out,
	}
	created, err := clientset.CoreV1().Secrets(arlonNamespace).Create(ctx, &secret, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	return created, nil
}

func portForward(args *porfForwardCfg, argoNs, arlonNs string) error {
	// use this command to get the argocd pod ➜  ~ kubectl get pods -l app.kubernetes.io/name=argocd-server -o yaml
	pods, err := args.clientset.CoreV1().Pods(argoNs).List(args.ctx, metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/name=argocd-server",
	})
	if err != nil {
		return err
	}
	if len(pods.Items) == 0 {
		return errors.NewNotFound(schema.GroupResource{
			Group:    "v1",
			Resource: "pod",
		}, defaultArgoServerDeployment)
	}
	var podIdx int
	for idx, pod := range pods.Items {
		if strings.Contains(pod.Name, defaultArgoServerDeployment) {
			if pod.Status.Phase == v1.PodRunning {
				podIdx = idx
				break
			}
		}
	}
	runPortForward(args, pods.Items[podIdx], argoNs)
	if err != nil {
		return err
	}
	return nil
}

func runPortForward(args *porfForwardCfg, pod v1.Pod, argoNs string) error {
	reqUrl, err := url.Parse(fmt.Sprintf("%s/api/v1/namespaces/%s/pods/%s/portforward", args.restCfg.Host, argoNs, pod.GetName()))
	if err != nil {
		return err
	}
	transport, upgrader, err := spdy.RoundTripperFor(args.restCfg)
	if err != nil {
		return err
	}
	stop := make(chan struct{}, 1)
	ready := make(chan struct{})
	dialer := spdy.NewDialer(upgrader, &http.Client{
		Transport: transport,
		Timeout:   time.Minute * 3,
	}, http.MethodPost, reqUrl)
	fw, err := portforward.NewOnAddresses(dialer, []string{"127.0.0.1"}, []string{fmt.Sprintf("%d:%d", args.hostPort, args.remotePort)}, stop, ready, os.Stdout, os.Stderr)
	if err != nil {
		return err
	}
	defer func() {
		stop <- struct{}{}
	}()
	go func() {
		_ = fw.ForwardPorts()
	}()
	<-ready
	ports, err := fw.GetPorts()
	if err != nil {
		return err
	}
	if len(ports) != 1 {
		return e.New("failed to get ports")
	}
	if err := args.callback(args.ctx, ports[0].Local); err != nil {
		return err
	}
	return nil
}
