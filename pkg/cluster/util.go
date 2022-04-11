package cluster

import (
	"fmt"
	"os"
	"path"
	"strings"
)

func clusterPathFromBasePath(basePath string, clusterName string) string {
	return path.Join(basePath, clusterName)
}

func mgmtPathFromClusterPath(clusterPath string) string {
	return path.Join(clusterPath, "mgmt")
}

func workloadPathFromClusterPath(clusterPath string) string {
	return path.Join(clusterPath, "workload")
}

func mgmtPathFromBasePath(basePath string, clusterName string) string {
	return path.Join(clusterPathFromBasePath(basePath, clusterName), "mgmt")
}

func decomposePath(mgmtPath string) (basePath string, clusterName string, err error) {
	comps := strings.Split(mgmtPath, string(os.PathSeparator))
	l := len(comps)
	if l < 3 {
		return "", "", fmt.Errorf("malformed repo path")
	}
	if comps[l-1] != "mgmt" {
		return "", "", fmt.Errorf("malformed repo path: unexpected last component (%s)", comps[l-1])
	}
	clusterName = comps[l-2]
	basePath = strings.Join(comps[0:l-2], string(os.PathSeparator))
	return
}
