package deploy

import _ "embed"

var (
	//go:embed manifests/deploy.yaml
	YAMLdeploy []byte
	//go:embed manifests/rbac_callhomeconfig.yaml
	YAMLrbacCHC []byte
	//go:embed manifests/rbac_clusterregistration.yaml
	YAMLrbacClusterReg []byte
	//go:embed manifests/rbac_appprofile.yaml
	YAMLrbacAppProf []byte
)
