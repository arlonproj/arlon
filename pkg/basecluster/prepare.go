package basecluster

import (
	"fmt"
	logpkg "github.com/arlonproj/arlon/pkg/log"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func Prepare(fileName string) (clusterName string, err error) {
	bld := resource.NewLocalBuilder()
	opts := resource.FilenameOptions{
		Filenames: []string{fileName},
	}
	res := bld.Unstructured().FilenameParam(false, &opts).Do()
	infos, err := res.Infos()
	if err != nil {
		return "", fmt.Errorf("builder failed to run: %s", err)
	}
	for i, info := range infos {
		gvk := info.Object.GetObjectKind().GroupVersionKind()
		if i > 0 {
			fmt.Println("---")
		}
		if gvk.Kind == "Cluster" {
			if clusterName != "" {
				return "", fmt.Errorf("there are 2 or more clusters")
			}
			clusterName = info.Name
		}
		removeNsAndDumpObj(info.Object, info.Name)
	}
	if clusterName == "" {
		return "", fmt.Errorf("failed to find cluster resource")
	}
	return
}

func removeNsAndDumpObj(obj runtime.Object, name string) error {
	log := logpkg.GetLogger()
	unstr := &unstructured.Unstructured{}
	if err := scheme.Scheme.Convert(obj, unstr, nil); err != nil {
		return fmt.Errorf("failed to convert object: %s", err)
	}
	ns := unstr.GetNamespace()
	if ns != "" {
		log.V(1).Info("removing namespace",
			"resource", unstr.GetName(), "namespace", ns)
		unstr.SetNamespace("")
	}
	out, err := yaml.Marshal(unstr.Object)
	if err != nil {
		return fmt.Errorf("failed to marshall object: %s", err)
	}
	fmt.Println(string(out))
	return nil
}
