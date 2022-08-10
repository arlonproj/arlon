package basecluster

import "errors"

var (
	ErrMultipleManifests    = errors.New("multiple manifests found")
	ErrNoManifest           = errors.New("failed to find base cluster manifest file")
	ErrNoKustomizationYaml  = errors.New("kustomization.yaml is missing")
	ErrNoConfigurationsYaml = errors.New("configurations.yaml is missing")
	ErrMultipleClusters     = errors.New("there are 2 or more clusters")
	ErrNoClusterResource    = errors.New("no cluster resource found")
)
