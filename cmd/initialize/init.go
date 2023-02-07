package initialize

import (
	"bytes"
	"context"
	e "errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/argoproj/argo-cd/v2/cmd/argocd/commands"
	"github.com/argoproj/argo-cd/v2/pkg/apiclient"
	"github.com/argoproj/argo-cd/v2/pkg/apiclient/account"
	"github.com/argoproj/argo-cd/v2/util/cli"
	argocdio "github.com/argoproj/argo-cd/v2/util/io"
	"github.com/argoproj/argo-cd/v2/util/localconfig"
	"github.com/arlonproj/arlon/cmd/basecluster"
	"github.com/arlonproj/arlon/cmd/gitrepo"
	"github.com/arlonproj/arlon/cmd/install"
	"github.com/arlonproj/arlon/config"
	"github.com/arlonproj/arlon/deploy"
	"github.com/arlonproj/arlon/pkg/argocd"
	gitrepo2 "github.com/arlonproj/arlon/pkg/gitrepo"
	"github.com/arlonproj/arlon/pkg/gitutils"
	gyaml "github.com/ghodss/yaml"
	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/spf13/cobra"
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
	clusterctl "sigs.k8s.io/cluster-api/cmd/clusterctl/client"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	argocdManifestURL                                = "https://raw.githubusercontent.com/argoproj/argo-cd/%s/manifests/install.yaml"
	defaultArgoNamespace                             = "argocd"
	defaultArlonNamespace                            = "arlon"
	defaultArlonArgoCDUser                           = "arlon"
	defaultArgoServerDeployment                      = "argocd-server"
	defaultArlonControllerDeployment                 = "arlon-controller"
	defaultArlonAppProfileControllerDeployment       = "arlon-appprof-ctrlr"
	reasonMinimumReplicasAvailable                   = "MinimumReplicasAvailable"
	argoInitialAdminSecret                           = "argocd-initial-admin-secret"
	argoServer                                       = "127.0.0.1:8080"
	exampleDir                                       = "examples"
	baseclusterDir                                   = "baseclusters"
	defaultCtrlPlaneCount                      int64 = 3
	defaultWorkerCount                         int64 = 3
	defaultK8sVersion                                = "v1.23.14"
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

var argocdGitTag string

type portForwardCallBack func(ctx context.Context, localPort uint16) error

func NewCommand() *cobra.Command {
	var (
		noConfirm   bool
		cliConfig   clientcmd.ClientConfig
		addExamples bool
		gitUser     string
		password    string
		repoUrl     string
	)
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
			canInstallArgo, err := canInstallArgocd(ctx, kubeClient, defaultArgoNamespace)
			if err != nil {
				return err
			}
			if canInstallArgo {
				fmt.Println("Cannot find argocd-server deployment. Argocd may not be installed")
				shouldInstallArgo := true
				if !noConfirm {
					shouldInstallArgo = cli.AskToProceed("ArgoCD not found, possibly not installed. Proceed to install? [y/n]")
				}
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
			canInstallArlonController, err := canInstallArlon(ctx, kubeClient, defaultArlonNamespace)
			if err != nil {
				return err
			}
			if canInstallArlonController {
				shouldInstallArlon := true
				if !noConfirm {
					shouldInstallArlon = cli.AskToProceed("arlon namespace not found. Install arlon controller?[y/n]")
				}
				if shouldInstallArlon {
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
					}, defaultArgoNamespace)
					if err != nil {
						return err
					}
				}
			}
			installCmd := install.NewCommand()
			_ = installCmd.Flag("capi-only").Value.Set("true")
			_ = installCmd.Flag("no-confirm").Value.Set(strconv.FormatBool(noConfirm))
			if err := installCmd.RunE(installCmd, []string{}); err != nil {
				return err
			}
			if !addExamples {
				return nil
			}
			err = portForward(&porfForwardCfg{
				ctx:        ctx,
				hostPort:   8080,
				remotePort: 8080,
				callback: func(ctx context.Context, localPort uint16) error {
					gitrepoCmd := gitrepo.NewCommand()
					var registerCmd *cobra.Command
					for _, c := range gitrepoCmd.Commands() {
						if c.Name() == "register" {
							registerCmd = c
							break
						}
					}
					fmt.Println("clearing arlon repoctx file")
					path, err := gitrepo2.GetRepoCfgPath()
					if err != nil {
						return err
					}
					if err := os.Remove(path); err != nil {
						if !os.IsNotExist(err) {
							return err
						}
						fmt.Println("repoctx file does not exist")
					}
					if len(repoUrl) == 0 {
						return e.New("repoUrl not set")
					}
					_ = registerCmd.Flag("user").Value.Set(gitUser)
					_ = registerCmd.Flag("password").Value.Set(password)
					err = registerCmd.RunE(cmd, []string{repoUrl})
					if err != nil {
						return err
					}
					return nil
				},
				restCfg:   cfg,
				kClient:   client,
				clientset: kubeClient,
			}, defaultArgoNamespace)
			if err != nil {
				return err
			}
			baseClusterArgs := []struct {
				repoPath         string
				name             string
				manifestFileName string
				provider         string
				flavor           string
			}{
				{
					repoPath:         filepath.Join(exampleDir, baseclusterDir, "docker"),
					name:             "docker-example",
					manifestFileName: "capd.yaml",
					provider:         "docker",
					flavor:           "development",
				},
				{
					repoPath:         filepath.Join(exampleDir, baseclusterDir, "aws"),
					name:             "aws-example",
					manifestFileName: "capa.yaml",
					provider:         "aws",
				},
				{
					repoPath:         filepath.Join(exampleDir, baseclusterDir, "aws-eks"),
					name:             "aws-eks-example",
					manifestFileName: "capa-eks.yaml",
					provider:         "aws",
					flavor:           "eks",
				},
			}
			baseClusterPaths := []string{}
			for _, b := range baseClusterArgs {
				baseClusterPaths = append(baseClusterPaths, b.repoPath)
			}
			exists, err := checkExamples(repoUrl, gitUser, password, baseClusterPaths, noConfirm)
			if err != nil {
				return err
			}
			if exists {
				fmt.Println("One or more example directories already exist and was not removed. Exiting...")
				return nil
			}
			for _, b := range baseClusterArgs {
				manifest, err := runGenerateClusterTemplate(b.name, defaultK8sVersion, b.provider, b.flavor)
				if err != nil {
					return err
				}
				if err := pushManifests(repoUrl, gitUser, password, b.repoPath, manifest, b.manifestFileName); err != nil {
					return err
				}
				bcl := basecluster.NewCommand()
				var prepCmd *cobra.Command
				for _, c := range bcl.Commands() {
					if c.Name() == "preparegit" {
						prepCmd = c
						break
					}
				}
				_ = prepCmd.Flag("repo-path").Value.Set(b.repoPath)
				if err := prepCmd.RunE(prepCmd, []string{}); err != nil {
					return err
				}
				fmt.Printf("to deploy a cluster on %s infrastructure run `arlon cluster create --cluster-name %s --repo-path %s`\n", b.provider, b.name, b.repoPath)
			}
			fmt.Printf("basecluster manifests pushed to %s\n", repoUrl)
			return nil
		},
	}
	cmd.Flags().BoolVarP(&noConfirm, "no-confirm", "y", false, "this flag disables the prompts for argocd and arlon installation on the management cluster")
	cmd.Flags().BoolVarP(&addExamples, "examples", "e", false, "this flag adds example cluster template manifests")
	cmd.Flags().StringVar(&gitUser, "username", "", "the git username for the workspace repository")
	cmd.Flags().StringVar(&password, "password", "", "the password for git user")
	cmd.Flags().StringVar(&repoUrl, "repoUrl", "", "URL for the workspace repository")
	cmd.MarkFlagsRequiredTogether("examples", "username", "password", "repoUrl")
	cliConfig = cli.AddKubectlFlagsToCmd(cmd)
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
		config.CRDAppProfile,
	}
	deplManifests := [][]byte{
		deploy.YAMLdeploy,
		deploy.YAMLrbacCHC,
		deploy.YAMLrbacClusterReg,
		deploy.YAMLrbacAppProf,
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
	err = wait.PollImmediate(time.Second*10, time.Minute*5, func() (bool, error) {
		fmt.Printf("waiting for arlon-controller\n")
		var deployment *apps.Deployment
		d, err := kubeClient.AppsV1().Deployments(arlonNs).Get(ctx, defaultArlonControllerDeployment, metav1.GetOptions{})
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
	err = wait.PollImmediate(time.Second*10, time.Minute*5, func() (bool, error) {
		fmt.Printf("waiting for arlon app profile controller\n")
		var depl *apps.Deployment
		d, err := kubeClient.AppsV1().Deployments(arlonNs).Get(ctx, defaultArlonAppProfileControllerDeployment, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		depl = d
		condition := getDeploymentCondition(depl.Status, apps.DeploymentAvailable)
		return condition != nil && condition.Reason == reasonMinimumReplicasAvailable, nil
	})
	if err != nil {
		return err
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
		fmt.Printf("waiting for argocd-server\n")
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

func canInstallArgocd(ctx context.Context, clientset *kubernetes.Clientset, argoNs string) (bool, error) {
	_, err := clientset.AppsV1().Deployments(argoNs).Get(ctx, defaultArgoServerDeployment, metav1.GetOptions{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "v1",
		},
	})
	if err != nil {
		if errors.IsNotFound(err) {
			return true, nil
		}
		return false, err
	}
	return false, nil
}

func canInstallArlon(ctx context.Context, kubeClient *kubernetes.Clientset, arlonNs string) (bool, error) {
	if _, err := kubeClient.CoreV1().Namespaces().Get(ctx, arlonNs, metav1.GetOptions{}); err != nil {
		if errors.IsNotFound(err) {
			return true, nil
		}
		return false, err
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

func portForward(args *porfForwardCfg, argoNs string) error {
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
	err = runPortForward(args, pods.Items[podIdx], argoNs)
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

func checkExamples(repoUrl, gitUser, password string, baseClusterPaths []string, noConfirm bool) (bool, error) {
	var exists bool
	repo, tmpDir, auth, err := argocd.CloneRepo(&argocd.RepoCreds{
		Url:      repoUrl,
		Username: gitUser,
		Password: password,
	}, repoUrl, "main")
	if err != nil {
		return false, err
	}
	defer func(path string) {
		_ = os.RemoveAll(path)
	}(tmpDir)
	wt, err := repo.Worktree()
	if err != nil {
		return false, err
	}
	fs := wt.Filesystem
	for _, path := range baseClusterPaths {
		_, err := fs.Stat(path)
		if os.IsNotExist(err) {
			continue
		}
		shouldDelete := noConfirm
		if !noConfirm {
			shouldDelete = cli.AskToProceed(fmt.Sprintf("Example directory %s already exists. Remove and proceed or Exit?[y/n]", path))
		}
		if shouldDelete {
			err := os.RemoveAll(filepath.Join(tmpDir, path))
			if err != nil {
				return false, err
			}
		} else {
			return true, nil
		}

	}
	status, err := wt.Status()
	if err != nil {
		return false, fmt.Errorf("failed to get worktree status: %w", err)
	}
	for file := range status {
		_, _ = wt.Add(file)
	}
	commitOpts := &gogit.CommitOptions{Author: &object.Signature{
		Name:  "arlon automation",
		Email: "arlon@arlon.io",
		When:  time.Now(),
	}}
	_, err = wt.Commit("clearing examples", commitOpts)
	if err != nil {
		return false, fmt.Errorf("failed to commit change: %w", err)
	}
	if err := repo.Push(&gogit.PushOptions{
		RemoteName: gogit.DefaultRemoteName,
		Auth:       auth,
		CABundle:   nil,
	}); err != nil {
		return false, err
	}

	return exists, nil
}

// from `clusterctl` source, had to copy it over because `clusterctl` only outputs to STDOUT :( Will add an issue on GH for this
func runGenerateClusterTemplate(clusterName, k8sVersion, infraProvider, flavor string) ([]byte, error) {
	c, err := clusterctl.New("")
	if err != nil {
		return nil, err
	}
	templateOpts := clusterctl.GetClusterTemplateOptions{
		ClusterName:              clusterName,
		KubernetesVersion:        k8sVersion,
		ControlPlaneMachineCount: to.Int64Ptr(defaultCtrlPlaneCount),
		WorkerMachineCount:       to.Int64Ptr(defaultWorkerCount),
		ProviderRepositorySource: &clusterctl.ProviderRepositorySourceOptions{
			InfrastructureProvider: infraProvider,
			Flavor:                 flavor,
		},
	}
	templ, err := c.GetClusterTemplate(templateOpts)
	if err != nil {
		return nil, err
	}
	printer := clusterctl.YamlPrinter(templ)
	out, err := printer.Yaml()
	if err != nil {
		return nil, err
	}
	out = append(out, '\n')
	return out, nil
}

func pushManifests(repoUrl string, user string, password string, repoPath string, manifestData []byte, manifestName string) error {
	repo, tmpDir, auth, err := argocd.CloneRepo(&argocd.RepoCreds{
		Url:      repoUrl,
		Username: user,
		Password: password,
	}, repoUrl, "main")
	if err != nil {
		return err
	}
	defer func(path string) {
		_ = os.RemoveAll(path)
	}(tmpDir)
	wt, err := repo.Worktree()
	if err != nil {
		return err
	}
	fs := wt.Filesystem
	err = os.MkdirAll(filepath.Join(tmpDir, repoPath), 0755)
	if err != nil {
		return err
	}
	f, err := fs.OpenFile(filepath.Join(repoPath, manifestName), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer argocdio.Close(f)
	_, err = f.Write(manifestData)
	if err != nil {
		return err
	}
	changed, err := gitutils.CommitChanges(tmpDir, wt, fmt.Sprintf("add %s", filepath.Join(repoPath, manifestName)))
	if err != nil {
		return err
	}
	if !changed {
		return nil
	}
	if err := repo.Push(&gogit.PushOptions{
		RemoteName: gogit.DefaultRemoteName,
		Auth:       auth,
		CABundle:   nil,
	}); err != nil {
		return err
	}
	return nil
}
