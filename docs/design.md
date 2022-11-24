# Arlon Design and Concepts

## Management cluster

This Kubernetes cluster hosts the following components:

- ArgoCD
- Arlon
- Cluster management stacks e.g. Cluster API and/or Crossplane

The Arlon state and controllers reside in the arlon namespace.

## Configuration bundle

A configuration bundle (or just "bundle") is grouping of data files that
produce a set of Kubernetes manifests via a *tool*. This closely follows ArgoCD's
definition of *tool types*. Consequently, the list of supported bundle
types mirrors ArgoCD's supported set of manifest-producing tools.
Each bundle is defined using a Kubernetes ConfigMap resource in the arlo namespace.
Additionally, a bundle can embed the data itself ("static bundle"), or contain a reference
to the data ("dynamic bundle"). A reference can be a URL, GitHub location, or Helm repo location.
The current list of supported bundle types is:

* manifest_inline: a single manifest yaml file embedded in the resource
* manifest_ref: a reference to a single manifest yaml file
* dir_inline: an embedded tarball that expands to a directory of YAML files
* helm_inline: an embedded Helm chart package
* helm_ref: an external reference to a Helm chart

### Bundle purpose

Bundles can specify an optional *purpose* to help classify and organize them.
In the future, Arlon may order bundle installation by purpose order (for e.g.
install bundles with purpose=*networking* before others) but that is not the
case today. The currently *suggested* purpose values are:

* networking
* add-on
* data-service
* application

## Profile

A profile expresses a desired configuration for a Kubernetes cluster.
It is composed of

- An optional clusterspec. If specified, it allows the profile
  to be used to create new clusters.
  If absent, the profile can only be applied to existing clusters.
- A list of bundles specifying the configuration to apply onto the cluster
  once it is operational
- An optional list of `values.yaml` settings for any Helm Chart type bundle
  in the bundle list

## Cluster
### Cluster Specification
A Cluster Specification contains desired settings when creating a new cluster. These settings are the values that define the shape and the configurations of the cluster.

Currently, there is a difference in the cluster specification for gen1 and gen2 clusters. The main difference in these cluster specifications is that gen2 Cluster Specification allow users to deploy arbitrarily complex clusters using the full Cluster API feature set.This is also closer to the gitops and declarative style of cluster creation and gives users more control over the cluster that they deploy.
#### gen1
A clusterspec contains desired settings when creating a new cluster. For gen1 clusters, this Cluster Specification is called [ClusterSpec](https://github.com/arlonproj/arlon/blob/main/docs/concepts.md#cluster-spec).

Clusterspec currently includes:

- Stack: the cluster provisioning stack, for e.g. *cluster-api* or *crossplane*
- Provider: the specific cluster management provider under that stack,
  if applicable. Example:
  for *cluster-api*, the possible values are *eks* and *kubeadm*
- Other settings that specify the "shape" of the cluster, such as the size of
  the control plane and the initial number of nodes of the data plane.
- The pod networking technology (under discussion: this may be moved to a
  bundle because most if not all CNI providers can be installed as manifests)  

#### gen2
for gen2 clusters, the Cluster Specification is called the base cluster, which is described in detail [here](https://github.com/arlonproj/arlon/blob/main/docs/baseclusters.md).
Base cluster manifest consists of : 

- A predefined list of Cluster API objects; Cluster, Machines, Machine Deployments, etc. to be deployed in the current namespace.
- The specific infrastructure provider to be used (e.g aws).ß
- Kubernetes verion
- Cluster templates/ flavors that need to be used for creating the cluster manifest (e.g eks, eks-managedmachinepool).

### Cluster Preparation
Once these cluster specifications are created successfully, the next step is to prepare the cluster for deployment.
#### gen1
Once the clusterspec is created for a gen-1 cluster, there is no need to prepare a workspace repository to create a new cluster.

#### gen2
Once the base cluster manifest is created, the next step is to preare the workspace repository directory in which this base cluster manifest is present. This is explained in detail [here](https://github.com/arlonproj/arlon/blob/main/docs/baseclusters.md#preparation)

### Cluster Creation
Now, all the prerequisites for creating a cluster are completed and the cluster can be created/deployed. 

#### Cluster Chart
The cluster chart is a Helm chart that creates (and optionally applies) the manifests necessary to create a cluster and deploy desired configurations and applications to it as a part of cluster creation, the following resources are created: The profile's Cluster Specification, bundle list and other settings are used to generate values for the cluster chart, and the chart is deployed as a Helm release into the *arlon* namespace in the management cluster.

Here is a summary of the kinds of resources generated and deployed by the chart:

- A unique namespace with a name based on the cluster's name. All subsequent
  resources below are created inside that namespace.
- The stack-specific resources to create the cluster (for e.g. Cluster API resources)
- A ClusterRegistration to automatically register the cluster with ArgoCD
- A GitRepoDir to automatically create a git repo and/or directory to host a copy
  of the expanded bundles. Every bundle referenced by the profile is
  copied/unpacked into its own subdirectory.
- One ArgoCD Application resource for each bundle.

#### gen1
Cluster deployment is explained [here](https://github.com/arlonproj/arlon/blob/main/docs/tutorial.md#clusters-gen1)

#### gen2
Base cluster creation is explained [here](https://github.com/arlonproj/arlon/blob/main/docs/baseclusters.md#creation)
