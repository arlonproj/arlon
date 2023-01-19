package clusterspec

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"sort"
)

const (
	aws    = "aws"
	gcp    = "gcp"
	azure  = "azure"
	capi   = "capi"
	xplane = "xplane"
)

var (
	ValidApiProviders = map[string]bool{
		capi:   true,
		xplane: true,
	}
	ValidCloudProviders = map[string]bool{
		aws:   true,
		gcp:   true,
		azure: true,
	}
	ValidClusterTypesByCloud = map[string]map[string]bool{
		aws:   {"kubeadm": true, "eks": true},
		gcp:   {"kubeadm": true, "gke": true},
		azure: {"kubeadm": true, "aks": true},
	}
	KubeconfigSecretKeyNameByApiProvider = map[string]string{
		"capi":   "value",
		"xplane": "kubeconfig",
	}
	// validRegionsAWS enumerates all the valid regions. This is obtained from the endpoint package of the AWS SDK
	// https://pkg.go.dev/github.com/aws/aws-sdk-go/aws/endpoints#pkg-constants
	validRegionsAWS = []string{
		endpoints.AfSouth1RegionID,
		endpoints.ApEast1RegionID,
		endpoints.ApNortheast1RegionID,
		endpoints.ApNortheast2RegionID,
		endpoints.ApNortheast3RegionID,
		endpoints.ApSouth1RegionID,
		endpoints.ApSoutheast1RegionID,
		endpoints.ApSoutheast2RegionID,
		endpoints.ApSoutheast3RegionID,
		endpoints.CaCentral1RegionID,
		endpoints.EuCentral1RegionID,
		endpoints.EuNorth1RegionID,
		endpoints.EuSouth1RegionID,
		endpoints.EuWest1RegionID,
		endpoints.EuWest2RegionID,
		endpoints.EuWest3RegionID,
		endpoints.MeSouth1RegionID,
		endpoints.SaEast1RegionID,
		endpoints.UsEast1RegionID,
		endpoints.UsEast2RegionID,
		endpoints.UsWest1RegionID,
		endpoints.UsWest2RegionID,
		endpoints.CnNorth1RegionID,
		endpoints.CnNorthwest1RegionID,
		endpoints.UsGovEast1RegionID,
		endpoints.UsGovWest1RegionID,
		endpoints.UsIsoEast1RegionID,
		endpoints.UsIsoWest1RegionID,
		endpoints.UsIsobEast1RegionID,
	}
	validRegionsByProvider = map[string][]string{
		aws: validRegionsAWS,
	}
)

var (
	ErrInvalidAPIProvider = fmt.Errorf("invalid api provider, the valid values are: %s",
		ValidValues(ValidApiProviders))
	ErrInvalidCloudProvider = fmt.Errorf("invalid cloud provider, the valid values are: %s",
		ValidValues(ValidCloudProviders))
)

func ValidValues(vals map[string]bool) string {
	var allKeys []string
	for key := range vals {
		allKeys = append(allKeys, key)
	}
	sort.Strings(allKeys)
	var ret string
	var i int
	for _, val := range allKeys {
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
		return ErrInvalidAPIProvider
	}
	return nil
}

func ValidCloudProviderAndClusterType(cloudProvider string, clusterType string) error {
	if !ValidCloudProviders[cloudProvider] {
		return ErrInvalidCloudProvider
	}
	if !ValidClusterTypesByCloud[cloudProvider][clusterType] {
		return fmt.Errorf("invalid cluster type, the valid values are: %s",
			ValidValues(ValidClusterTypesByCloud[cloudProvider]))
	}
	return nil
}

// isOneOf takes in a string value and a slice of all possible valid values for the string.
// It returns true if the provided val is in possibleValues, false otherwise.
func isOneOf(val string, possibleValues []string) bool {
	for _, value := range possibleValues {
		if val == value {
			return true
		}
	}
	return false
}

// ValidateRegionByProvider checks if a supplied region string is valid or not for the given provider.
func ValidateRegionByProvider(provider string, region string) error {
	validRegions := validRegionsByProvider[provider]
	if !isOneOf(region, validRegions) {
		return fmt.Errorf("invalid region %s for %s, valid values are: %v", region, provider, validRegions)
	}
	return nil
}
