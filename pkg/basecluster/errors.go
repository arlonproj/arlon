package basecluster

import (
	"errors"
)

var (
	ErrMultipleManifests    = errors.New("multiple manifests found")
	ErrNoManifest           = errors.New("failed to find cluster template manifest file")
	ErrNoKustomizationYaml  = errors.New("kustomization.yaml is missing")
	ErrNoConfigurationsYaml = errors.New("configurations.yaml is missing")
	ErrMultipleClusters     = errors.New("there are 2 or more clusters")
	ErrNoClusterResource    = errors.New("no cluster resource found")
	ErrBuilderFailedRun     = errors.New("builder failed to run")
	ErrResourceHasNamespace = errors.New("resource has a namespace defined")
	Err2orMoreClusters      = errors.New("there are 2 or more clusters")
)
