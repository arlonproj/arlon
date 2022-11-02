package initialize

import (
	"bytes"
	"context"
	"fmt"
	"github.com/argoproj/argo-cd/v2/util/cli"
	argocdio "github.com/argoproj/argo-cd/v2/util/io"
	"github.com/arlonproj/arlon/pkg/argocd"
	"github.com/spf13/cobra"
	"io"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/tools/clientcmd"
	"net/http"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

const (
	argocdManifestURL = "https://raw.githubusercontent.com/argoproj/argo-cd/%s/manifests/install.yaml"
)

var argocdGitTag = "release-2.4"

func NewCommand() *cobra.Command {
	var argoCfgPath string
	var cliConfig clientcmd.ClientConfig
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Run the init command",
		Long:  "Run the init command",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := argocd.NewArgocdClient(argoCfgPath)
			if err != nil {
				fmt.Println("Cannot initialize argocd client. Argocd may not be installed")
				// prompt for a message and proceed
				//canInstallArgo := cli.AskToProceed("argo-cd not found, possibly not installed. Proceed to install? [y/n]")
				if true {
					cfg, err := cliConfig.ClientConfig()
					if err != nil {
						return err
					}
					client, err := k8sclient.New(cfg, k8sclient.Options{})
					if err != nil {
						return err
					}
					downloadLink := fmt.Sprintf(argocdManifestURL, argocdGitTag)
					if err := client.Create(cmd.Context(), &v1.Namespace{
						TypeMeta: metav1.TypeMeta{
							Kind: "Namespace",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name: "argocd",
						},
					}); err != nil {
						return err
					}
					if err := installArgo(downloadLink, client); err != nil {
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

func installArgo(downloadLink string, client k8sclient.Client) error {
	manifest, err := downloadManifest(downloadLink)
	if err != nil {
		return err
	}
	resources := []*unstructured.Unstructured{}
	decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(manifest), 4096)
	for {
		resource := unstructured.Unstructured{}
		err := decoder.Decode(&resource)
		if err == nil {
			resources = append(resources, &resource)
		} else if err == io.EOF {
			break
		} else {
			return err
		}
	}
	for _, obj := range resources {
		err := applyObject(context.Background(), client, obj, "argocd")
		if err != nil {
			return err
		}
	}
	return nil
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
