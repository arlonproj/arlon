package config

import _ "embed"

var (
	//go:embed crd/bases/core.arlon.io_callhomeconfigs.yaml
	CRDCallHomeConfig []byte
	//go:embed crd/bases/core.arlon.io_clusterregistrations.yaml
	CRDClusterReg []byte
	//go:embed crd/bases/core.arlon.io_profiles.yaml
	CRDProfile []byte
	//go:embed crd/bases/core.arlon.io_appprofiles.yaml
	CRDAppProfile []byte
)
