package cluster

type Cluster struct {
	Name string
	ClusterSpecName string // empty for external clusters
	ProfileName string
	IsExternal  bool
	SecretName string // The corresponding argocd secret. Empty for non-external clusters.
}

const clusterTypeLabelKey = "arlon.io/cluster-type"
const externalClusterTypeLabel = "arlon.io/cluster-type=external"
const argoClusterSecretTypeLabel = "argocd.argoproj.io/secret-type=cluster"
