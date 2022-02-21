package clusterspec

import "fmt"

var (
	ValidApiProviders = map[string]bool{
		"capi":   true,
		"xplane": true,
	}
	ValidCloudProviders = map[string]bool{
		"aws":   true,
		"gcp":   true,
		"azure": true,
	}
	ValidClusterTypesByCloud = map[string]map[string]bool{
		"aws":   {"kubeadm": true, "eks": true},
		"gcp":   {"kubeadm": true, "gke": true},
		"azure": {"kubeadm": true, "aks": true},
	}
	KubeconfigSecretKeyNameByApiProvider = map[string]string{
		"capi":   "value",
		"xplane": "kubeconfig",
	}
)

func ValidValues(vals map[string]bool) string {
	var ret string
	var i int
	for val, _ := range vals {
		var sep string
		if i > 0 {
			sep = "|"
		}
		ret = ret + sep + val
		i = i + 1
	}
	return ret
}

func ValidApiProvider(apiProvider string) error {
	if !ValidApiProviders[apiProvider] {
		return fmt.Errorf("invalid api provider, the valid values are: %s",
			ValidValues(ValidApiProviders))
	}
	return nil
}

func ValidCloudProviderAndClusterType(cloudProvider string, clusterType string) error {
	if !ValidCloudProviders[cloudProvider] {
		return fmt.Errorf("invalid cloud provider, the valid values are: %s",
			ValidValues(ValidCloudProviders))
	}
	if !ValidClusterTypesByCloud[cloudProvider][clusterType] {
		return fmt.Errorf("invalid cluster type, the valid values are: %s",
			ValidValues(ValidClusterTypesByCloud[cloudProvider]))
	}
	return nil
}
