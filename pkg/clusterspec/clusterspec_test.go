package clusterspec

import (
	"errors"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strconv"
	"testing"
)

func TestFromConfigMap(t *testing.T) {
	testCases := []struct {
		Cm         *corev1.ConfigMap
		Desc       string
		ShouldFail bool
	}{
		{
			Cm: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: "name",
					Labels: map[string]string{
						"managed-by": "arlon",
						"arlon-type": "clusterspec",
					},
				},
				Data: map[string]string{
					ApiProviderKey:               "apiProvider",
					CloudProviderKey:             "cloudProvider",
					ClusterTypeKey:               "clusterType",
					KubernetesVersionKey:         "kubernetesVersion",
					NodeTypeKey:                  "nodeType",
					NodeCountKey:                 strconv.Itoa(3),
					MasterNodeCountKey:           strconv.Itoa(1),
					RegionKey:                    "region",
					PodCidrBlockKey:              "podCidrBlock",
					SshKeyNameKey:                "sshKeyName",
					ClusterAutoscalerEnabledKey:  strconv.FormatBool(true),
					ClusterAutoscalerMinNodesKey: strconv.Itoa(1),
					ClusterAutoscalerMaxNodesKey: strconv.Itoa(9),
					TagsKey:                      "tags",
					DescriptionKey:               "description",
				},
			},
			Desc:       "Valid config map",
			ShouldFail: false,
		},
		{
			Cm: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: "name",
					Labels: map[string]string{
						"managed-by": "arlon",
						"arlon-type": "clusterspec",
					},
				},
				Data: map[string]string{
					ApiProviderKey:               "apiProvider",
					CloudProviderKey:             "cloudProvider",
					ClusterTypeKey:               "clusterType",
					KubernetesVersionKey:         "kubernetesVersion",
					NodeTypeKey:                  "nodeType",
					NodeCountKey:                 "invalidNodeCount",
					MasterNodeCountKey:           strconv.Itoa(1),
					RegionKey:                    "region",
					PodCidrBlockKey:              "podCidrBlock",
					SshKeyNameKey:                "sshKeyName",
					ClusterAutoscalerEnabledKey:  strconv.FormatBool(true),
					ClusterAutoscalerMinNodesKey: strconv.Itoa(1),
					ClusterAutoscalerMaxNodesKey: strconv.Itoa(9),
					TagsKey:                      "tags",
					DescriptionKey:               "description",
				},
			},
			Desc:       "Invalid config map",
			ShouldFail: true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Desc, func(t *testing.T) {
			_, err := FromConfigMap(tc.Cm)
			if tc.ShouldFail {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

		})
	}

	defaultResetCase := struct {
		Cm   *corev1.ConfigMap
		Desc string
	}{
		Cm: &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: "name",
				Labels: map[string]string{
					"managed-by": "arlon",
					"arlon-type": "clusterspec",
				},
			},
			Data: map[string]string{
				ApiProviderKey:               "apiProvider",
				CloudProviderKey:             "cloudProvider",
				ClusterTypeKey:               "clusterType",
				KubernetesVersionKey:         "kubernetesVersion",
				NodeTypeKey:                  "nodeType",
				NodeCountKey:                 strconv.Itoa(3),
				MasterNodeCountKey:           "invalid",
				RegionKey:                    "region",
				PodCidrBlockKey:              "podCidrBlock",
				SshKeyNameKey:                "sshKeyName",
				ClusterAutoscalerEnabledKey:  "noIdea",
				ClusterAutoscalerMinNodesKey: "strconv.Itoa(1)",
				ClusterAutoscalerMaxNodesKey: "strconv.Itoa(9)",
				TagsKey:                      "tags",
				DescriptionKey:               "description",
			},
		},
		Desc: "Testing default values",
	}
	t.Run(defaultResetCase.Desc, func(t *testing.T) {
		cs, err := FromConfigMap(defaultResetCase.Cm)
		require.NoError(t, err)
		require.Equal(t, defaultMasterNodeCount, cs.MasterNodeCount)
		require.Equal(t, defaultClusterAutoscalerEnabled, cs.ClusterAutoscalerEnabled)
		require.Equal(t, defaultClusterAutoscalerMinNodes, cs.ClusterAutoscalerMinNodes)
		require.Equal(t, defaultClusterAutoscalerMaxNodes, cs.ClusterAutoscalerMaxNodes)
	})
}

func TestToConfigMap(t *testing.T) {
	testCases := []struct {
		name                      string
		apiProvider               string
		cloudProvider             string
		clusterType               string
		kubernetesVersion         string
		nodeType                  string
		nodeCount                 int
		masterNodeCount           int
		region                    string
		podCidrBlock              string
		sshKeyName                string
		clusterAutoscalerEnabled  bool
		clusterAutoscalerMinNodes int
		clusterAutoscalerMaxNodes int
		tags                      string
		description               string
	}{
		{
			name:                      "test1",
			apiProvider:               "capi",
			cloudProvider:             "aws",
			clusterType:               "eks",
			kubernetesVersion:         "1.22.33",
			nodeType:                  "noIdea",
			nodeCount:                 1,
			masterNodeCount:           1,
			region:                    "us-west-1",
			podCidrBlock:              "192.168.0.0/24",
			sshKeyName:                "ssh",
			clusterAutoscalerEnabled:  false,
			clusterAutoscalerMinNodes: 0,
			clusterAutoscalerMaxNodes: 0,
			tags:                      "unit-test",
			description:               "unit-test",
		},
		{
			name:                      "test2",
			apiProvider:               "not-capi",
			cloudProvider:             "not-aws",
			clusterType:               "eks",
			kubernetesVersion:         "1.22.33",
			nodeType:                  "noIdea",
			nodeCount:                 1,
			masterNodeCount:           1,
			region:                    "us-west-2",
			podCidrBlock:              "192.168.0.0/22",
			sshKeyName:                "ssh",
			clusterAutoscalerEnabled:  false,
			clusterAutoscalerMinNodes: 0,
			clusterAutoscalerMaxNodes: 0,
			tags:                      "unit-test",
			description:               "unit-test",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cm := ToConfigMap(tc.name, tc.apiProvider, tc.cloudProvider, tc.clusterType, tc.kubernetesVersion, tc.nodeType, tc.nodeCount, tc.masterNodeCount, tc.region, tc.podCidrBlock, tc.sshKeyName, tc.clusterAutoscalerEnabled, tc.clusterAutoscalerMinNodes, tc.clusterAutoscalerMaxNodes, tc.tags, tc.description)
			require.Equal(t, tc.name, cm.Name)
			require.Equal(t, tc.apiProvider, cm.Data[ApiProviderKey])
			require.Equal(t, tc.cloudProvider, cm.Data[CloudProviderKey])
			require.Equal(t, tc.clusterType, cm.Data[ClusterTypeKey])
			require.Equal(t, tc.kubernetesVersion, cm.Data[KubernetesVersionKey])
			require.Equal(t, tc.nodeType, cm.Data[NodeTypeKey])
			require.Equal(t, tc.region, cm.Data[RegionKey])
			require.Equal(t, tc.podCidrBlock, cm.Data[PodCidrBlockKey])
			require.Equal(t, tc.sshKeyName, cm.Data[SshKeyNameKey])
			require.Equal(t, tc.tags, cm.Data[TagsKey])
			require.Equal(t, tc.description, cm.Data[DescriptionKey])
			nodeCount, err := strconv.Atoi(cm.Data[NodeCountKey])
			require.NoError(t, err)
			require.Equal(t, tc.nodeCount, nodeCount)
			masterNodes, err := strconv.Atoi(cm.Data[MasterNodeCountKey])
			require.NoError(t, err)
			require.Equal(t, tc.masterNodeCount, masterNodes)
			casEnabled, err := strconv.ParseBool(cm.Data[ClusterAutoscalerEnabledKey])
			require.NoError(t, err)
			require.Equal(t, tc.clusterAutoscalerEnabled, casEnabled)
			casMin, err := strconv.Atoi(cm.Data[ClusterAutoscalerMinNodesKey])
			require.NoError(t, err)
			require.Equal(t, tc.clusterAutoscalerMinNodes, casMin)
			casMax, err := strconv.Atoi(cm.Data[ClusterAutoscalerMaxNodesKey])
			require.NoError(t, err)
			require.Equal(t, tc.clusterAutoscalerMaxNodes, casMax)
		})
	}
}

func TestSubchartNameFromClusterSpec(t *testing.T) {
	testCases := []struct {
		Desc     string
		Cspec    ClusterSpec
		Expected string
		Err      error
	}{
		{
			Desc: "ValidSpecCAPIAWSEKS",
			Cspec: ClusterSpec{
				ApiProvider:   capi,
				CloudProvider: aws,
				Type:          "eks",
			},
			Expected: "capi-aws-eks",
			Err:      nil,
		},
		{
			Desc: "ValidSpecCAPIAWSKubeadm",
			Cspec: ClusterSpec{
				ApiProvider:   capi,
				CloudProvider: aws,
				Type:          "kubeadm",
			},
			Expected: "capi-aws-kubeadm",
			Err:      nil,
		},
		{
			Desc: "ValidSpecCAPIGCPGKE",
			Cspec: ClusterSpec{
				ApiProvider:   capi,
				CloudProvider: gcp,
				Type:          "gke",
			},
			Expected: "capi-gcp-gke",
			Err:      nil,
		},
		{
			Desc: "ValidSpecCAPIGCPKubeadm",
			Cspec: ClusterSpec{
				ApiProvider:   capi,
				CloudProvider: gcp,
				Type:          "kubeadm",
			},
			Expected: "capi-gcp-kubeadm",
			Err:      nil,
		},
		{
			Desc: "ValidSpecCAPIAZAKS",
			Cspec: ClusterSpec{
				ApiProvider:   capi,
				CloudProvider: azure,
				Type:          "aks",
			},
			Expected: "capi-azure-aks",
			Err:      nil,
		},
		{
			Desc: "ValidSpecCAPIAZKubeadm",
			Cspec: ClusterSpec{
				ApiProvider:   capi,
				CloudProvider: azure,
				Type:          "kubeadm",
			},
			Expected: "capi-azure-kubeadm",
			Err:      nil,
		},

		{
			Desc: "ValidSpecXPAWSEKS",
			Cspec: ClusterSpec{
				ApiProvider:   xplane,
				CloudProvider: aws,
				Type:          "eks",
			},
			Expected: "xplane-aws-eks",
			Err:      nil,
		},
		{
			Desc: "ValidSpecXPAWSKubeadm",
			Cspec: ClusterSpec{
				ApiProvider:   xplane,
				CloudProvider: aws,
				Type:          "kubeadm",
			},
			Expected: "xplane-aws-kubeadm",
			Err:      nil,
		},
		{
			Desc: "ValidSpecXPGCPGKE",
			Cspec: ClusterSpec{
				ApiProvider:   xplane,
				CloudProvider: gcp,
				Type:          "gke",
			},
			Expected: "xplane-gcp-gke",
			Err:      nil,
		},
		{
			Desc: "ValidSpecXPGCPKubeadm",
			Cspec: ClusterSpec{
				ApiProvider:   xplane,
				CloudProvider: gcp,
				Type:          "kubeadm",
			},
			Expected: "xplane-gcp-kubeadm",
			Err:      nil,
		},
		{
			Desc: "ValidSpecXPAZAKS",
			Cspec: ClusterSpec{
				ApiProvider:   xplane,
				CloudProvider: azure,
				Type:          "aks",
			},
			Expected: "xplane-azure-aks",
			Err:      nil,
		},
		{
			Desc: "ValidSpecXPAZKubeadm",
			Cspec: ClusterSpec{
				ApiProvider:   xplane,
				CloudProvider: azure,
				Type:          "kubeadm",
			},
			Expected: "xplane-azure-kubeadm",
			Err:      nil,
		},
		{
			Desc: "InvalidSpecUnknownAZKubeadm",
			Cspec: ClusterSpec{
				ApiProvider:   "unknown",
				CloudProvider: azure,
				Type:          "kubeadm",
			},
			Expected: "",
			Err:      ErrInvalidAPIProvider,
		},
		{
			Desc: "InvalidSpecCAPIAZKubeadm",
			Cspec: ClusterSpec{
				ApiProvider:   capi,
				CloudProvider: "rain",
				Type:          "kubeadm",
			},
			Expected: "",
			Err:      ErrInvalidCloudProvider,
		},
		{
			Desc: "InvalidSpecCAPIAZKubeadm",
			Cspec: ClusterSpec{
				ApiProvider:   capi,
				CloudProvider: "rain",
				Type:          "kubeadm",
			},
			Expected: "",
			Err:      ErrInvalidCloudProvider,
		},
		{
			Desc: "InvalidSpecCAPIAZNotACluster",
			Cspec: ClusterSpec{
				ApiProvider:   capi,
				CloudProvider: azure,
				Type:          "notacluster",
			},
			Expected: "",
			Err:      errors.New("invalid cluster type, the valid values are: aks|kubeadm"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Desc, func(t *testing.T) {
			name, err := SubchartNameFromClusterSpec(&tc.Cspec)
			require.Equal(t, tc.Err, err)
			require.Equal(t, tc.Expected, name)
		})
	}
}

func TestSubchartName(t *testing.T) {
	testCases := []struct {
		Cm       *corev1.ConfigMap
		Desc     string
		Expected string
		Err      error
	}{
		{
			Cm: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: "name",
					Labels: map[string]string{
						"managed-by": "arlon",
						"arlon-type": "clusterspec",
					},
				},
				Data: map[string]string{
					ApiProviderKey:   capi,
					CloudProviderKey: azure,
					ClusterTypeKey:   "kubeadm",
					NodeCountKey:     "3",
				},
			},
			Desc:     "CAPIAzKubeadm",
			Expected: "capi-azure-kubeadm",
			Err:      nil,
		},
		{
			Cm: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: "name",
					Labels: map[string]string{
						"managed-by": "arlon",
						"arlon-type": "clusterspec",
					},
				},
				Data: map[string]string{
					ApiProviderKey:   capi,
					CloudProviderKey: azure,
					ClusterTypeKey:   "aks",
					NodeCountKey:     "3",
				},
			},
			Desc:     "CAPIAzAKS",
			Expected: "capi-azure-aks",
			Err:      nil,
		},
		{
			Cm: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: "name",
					Labels: map[string]string{
						"managed-by": "arlon",
						"arlon-type": "clusterspec",
					},
				},
				Data: map[string]string{
					ApiProviderKey:   "what",
					CloudProviderKey: azure,
					ClusterTypeKey:   "kubeadm",
					NodeCountKey:     "3",
				},
			},
			Desc:     "WhatAzKubeadm",
			Expected: "",
			Err:      ErrInvalidAPIProvider,
		},
		{
			Cm: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: "name",
					Labels: map[string]string{
						"managed-by": "arlon",
						"arlon-type": "clusterspec",
					},
				},
				Data: map[string]string{
					ApiProviderKey:   capi,
					CloudProviderKey: "water",
					ClusterTypeKey:   "kubeadm",
					NodeCountKey:     "3",
				},
			},
			Desc:     "CAPIWaterKubeadm",
			Expected: "",
			Err:      ErrInvalidCloudProvider,
		},
		{
			Cm: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: "name",
					Labels: map[string]string{
						"managed-by": "arlon",
						"arlon-type": "clusterspec",
					},
				},
				Data: map[string]string{
					ApiProviderKey:   capi,
					CloudProviderKey: azure,
					ClusterTypeKey:   "not-kubeadm",
					NodeCountKey:     "3",
				},
			},
			Desc:     "CAPIAzNotKubeadm",
			Expected: "",
			Err:      errors.New("invalid cluster type, the valid values are: aks|kubeadm"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Desc, func(t *testing.T) {
			name, err := SubchartName(tc.Cm)
			require.Equal(t, tc.Err, err)
			require.Equal(t, tc.Expected, name)
		})
	}
}

func TestClusterAutoscalerSubchartNameFromClusterSpec(t *testing.T) {
	testCases := []struct {
		Cs       ClusterSpec
		Desc     string
		Err      error
		Expected string
	}{
		{
			Cs: ClusterSpec{
				ApiProvider: xplane,
			},
			Desc:     "XplaneCspec",
			Err:      nil,
			Expected: "xplane-cluster-autoscaler",
		},
		{
			Cs: ClusterSpec{
				ApiProvider: capi,
			},
			Desc:     "CAPICspec",
			Err:      nil,
			Expected: "capi-cluster-autoscaler",
		},
		{
			Cs: ClusterSpec{
				ApiProvider: "unknown",
			},
			Desc:     "InvalidAPIProviderCspec",
			Err:      ErrInvalidAPIProvider,
			Expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Desc, func(t *testing.T) {
			name, err := ClusterAutoscalerSubchartNameFromClusterSpec(&tc.Cs)
			require.Equal(t, tc.Err, err)
			require.Equal(t, tc.Expected, name)
		})
	}
}
