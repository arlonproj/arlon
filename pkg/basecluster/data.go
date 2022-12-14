package basecluster

const kustomizationYamlTemplate = `resources:
- {{.ManifestFileName}}

configurations:
- configurations.yaml
`

// -----------------------------------------------------------------------------

const ConfigurationsYaml = `# Source: https://blog.scottlowe.org/2021/10/11/kustomize-transformer-configurations-for-cluster-api-v1beta1/
nameReference:
- kind: Cluster
  group: cluster.x-k8s.io
  version: v1beta1
  fieldSpecs:
  - path: spec/clusterName
    kind: MachineDeployment
  - path: spec/template/spec/clusterName
    kind: MachineDeployment
  - path: spec/clusterName
    kind: MachinePool
  - path: spec/template/spec/clusterName
    kind: MachinePool
- kind: AWSCluster
  group: infrastructure.cluster.x-k8s.io
  version: v1beta1
  fieldSpecs:
  - path: spec/infrastructureRef/name
    kind: Cluster
- kind: KubeadmControlPlane
  group: controlplane.cluster.x-k8s.io
  version: v1beta1
  fieldSpecs:
  - path: spec/controlPlaneRef/name
    kind: Cluster
- kind: AWSManagedControlPlane
  group: controlplane.cluster.x-k8s.io
  version: v1beta1
  fieldSpecs:
  - path: spec/controlPlaneRef/name
    kind: Cluster
  - path: spec/infrastructureRef/name
    kind: Cluster
- kind: AWSManagedControlPlane
  group: controlplane.cluster.x-k8s.io
  version: v1beta2
  fieldSpecs:
  - path: spec/controlPlaneRef/name
    kind: Cluster
- kind: AWSManagedCluster
  group: infrastructure.cluster.x-k8s.io
  version: v1beta2
  fieldSpecs:
  - path: spec/infrastructureRef/name
    kind: Cluster
- kind: AWSMachine
  group: infrastructure.cluster.x-k8s.io
  version: v1beta1
  fieldSpecs:
  - path: spec/infrastructureRef/name
    kind: Machine
- kind: KubeadmConfig
  group: bootstrap.cluster.x-k8s.io
  version: v1beta1
  fieldSpecs:
  - path: spec/bootstrap/configRef/name
    kind: Machine
  - path: spec/template/spec/bootstrap/configRef/name
    kind: MachinePool
- kind: AWSMachineTemplate
  group: infrastructure.cluster.x-k8s.io
  version: v1beta2
  fieldSpecs:
  - path: spec/template/spec/infrastructureRef/name
    kind: MachineDeployment
  - path: spec/machineTemplate/infrastructureRef/name
    kind: KubeadmControlPlane
- kind: AWSMachineTemplate
  group: infrastructure.cluster.x-k8s.io
  version: v1beta2
  fieldSpecs:
  - path: spec/template/spec/infrastructureRef/name
    kind: MachineDeployment
  - path: spec/machineTemplate/infrastructureRef/name
    kind: KubeadmControlPlane
- kind: AWSMachineTemplate
  group: infrastructure.cluster.x-k8s.io
  version: v1beta1
  fieldSpecs:
  - path: spec/template/spec/infrastructureRef/name
    kind: MachineDeployment
  - path: spec/machineTemplate/infrastructureRef/name
    kind: KubeadmControlPlane
- kind: KubeadmConfigTemplate
  group: bootstrap.cluster.x-k8s.io
  version: v1beta1
  fieldSpecs:
  - path: spec/template/spec/bootstrap/configRef/name
    kind: MachineDeployment
- kind: EKSConfigTemplate
  group: bootstrap.cluster.x-k8s.io
  version: v1beta2
  fieldSpecs:
  - path: spec/template/spec/bootstrap/configRef/name
    kind: MachineDeployment
- kind: EKSConfigTemplate
  group: bootstrap.cluster.x-k8s.io
  version: v1beta1
  fieldSpecs:
  - path: spec/template/spec/bootstrap/configRef/name
    kind: MachineDeployment
- kind: DockerCluster
  group: infrastructure.cluster.x-k8s.io
  version: v1beta1
  fieldSpecs:
  - path: spec/infrastructureRef/name
    kind: Cluster
- kind: DockerMachine
  group: infrastructure.cluster.x-k8s.io
  version: v1beta1
  fieldSpecs:
  - path: spec/infrastructureRef/name
    kind: Machine
- kind: DockerMachineTemplate
  group: infrastructure.cluster.x-k8s.io
  version: v1beta1
  fieldSpecs:
  - path: spec/template/spec/infrastructureRef/name
    kind: MachineDeployment
  - path: spec/machineTemplate/infrastructureRef/name
    kind: KubeadmControlPlane
- kind: AWSManagedMachinePool
  group: infrastructure.cluster.x-k8s.io
  version: v1beta2
  fieldSpecs:
  - path: spec/template/spec/infrastructureRef/name
    kind: MachinePool
- kind: AWSManagedMachinePool
  group: infrastructure.cluster.x-k8s.io
  version: v1beta1
  fieldSpecs:
  - path: spec/template/spec/infrastructureRef/name
    kind: MachinePool
- kind: AWSMachinePool
  group: infrastructure.cluster.x-k8s.io
  version: v1beta2
  fieldSpecs:
  - path: spec/template/spec/infrastructureRef/name
    kind: MachinePool
- kind: AWSMachinePool
  group: infrastructure.cluster.x-k8s.io
  version: v1beta1
  fieldSpecs:
  - path: spec/template/spec/infrastructureRef/name
    kind: MachinePool
- kind: EKSConfig
  group: bootstrap.cluster.x-k8s.io
  version: v1beta2
  fieldSpecs:
  - path: spec/template/spec/bootstrap/configRef/name
    kind: MachinePool
`
