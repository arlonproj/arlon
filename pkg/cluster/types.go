package cluster

type Cluster struct {
	Name            string
	ClusterSpecName string           // empty for external clusters
	BaseCluster     *BaseClusterInfo // gen2 only
	ProfileName     string
	IsExternal      bool
	SecretName      string // The corresponding argocd secret. Empty for non-external clusters.
}

type BaseClusterInfo struct {
	Name         string
	RepoUrl      string
	RepoRevision string
	RepoPath     string
}

const clusterTypeLabelKey = "arlon.io/cluster-type"
const externalClusterTypeLabel = "arlon.io/cluster-type=external"
const argoClusterSecretTypeLabel = "argocd.argoproj.io/secret-type=cluster"

const baseClusterNameAnnotation = "arlon.io/basecluster-name"
const baseClusterRepoUrlAnnotation = "arlon.io/basecluster-repo-url"
const baseClusterRepoRevisionAnnotation = "arlon.io/basecluster-repo-revision"
const baseClusterRepoPathAnnotation = "arlon.io/basecluster-repo-path"

const ArlonGen1ClusterLabelQueryOnArgoApps = "managed-by=arlon,arlon-type=cluster"
const ArlonGen2ClusterLabelQueryOnArgoApps = "managed-by=arlon,arlon-type=cluster-app"
