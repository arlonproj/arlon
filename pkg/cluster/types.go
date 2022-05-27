package cluster

type Cluster struct {
	Name string
	ClusterSpecName string // empty for unmanaged clusters
	ProfileName string
	IsExternal  bool
}

