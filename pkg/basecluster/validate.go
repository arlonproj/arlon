package basecluster

import (
	"fmt"
	"k8s.io/cli-runtime/pkg/resource"
)

// Validate verifies whether the resources in the specified file contain one and
// only one cluster, and that no resources have a namespace specified.
// If successful, the function returns the name of the cluster.
func Validate(fileName string) (clusterName string, err error) {
	bld := resource.NewLocalBuilder()
	opts := resource.FilenameOptions{
		Filenames: []string{fileName},
	}
	res := bld.Unstructured().FilenameParam(false, &opts).Do()
	infos, err := res.Infos()
	if err != nil {
		return "", fmt.Errorf("builder failed to run: %s", err)
	}
	for _, info := range infos {
		gvk := info.Object.GetObjectKind().GroupVersionKind()
		if info.Namespace != "" {
			return "",
				fmt.Errorf("resource %s of kind %s has a namespace defined",
					info.Name, gvk.Kind)
		}
		if gvk.Kind == "Cluster" {
			if clusterName != "" {
				return "", fmt.Errorf("there are 2 or more clusters")
			}
			clusterName = info.Name
		}
	}
	if clusterName == "" {
		return "", fmt.Errorf("failed to find cluster resource")
	}
	return
}
