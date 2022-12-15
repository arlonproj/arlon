# Concepts

## Understanding the Fundamentals

It is important to understand the fundamentals of the underlying technology that Arlon is built on, before you start making use of Arlon. 
We recommend a good understanding of core concepts around [Docker](https://www.docker.com/), containerization, [Kubernetes](https://kubernetes.io/), GitOps and Continuous Delivery to understand the fundamentals behind Arlon. 
We also recommend having a basic understanding of application templating technologies such as [Helm](https://helm.sh) and [Kustomize](https://kustomize.io) that are commonly used with Kubernetes. 

## Management cluster

Before you can use arlon, you need a Kubernetes cluster to host the following components:

- ArgoCD
- Arlon
- Cluster management stacks e.g. Cluster API and/or Crossplane

The Arlon state and controllers reside in the arlon namespace on this cluster. 

## Configuration bundle

A configuration bundle (or just "bundle") is grouping of data files that
produce a set of Kubernetes manifests via a *tool*. This closely follows ArgoCD's
definition of *tool types*. Consequently, the list of supported bundle
types mirrors ArgoCD's supported set of manifest-producing tools.
Each bundle is defined using a Kubernetes ConfigMap resource in the arlon namespace.
Additionally, a bundle can embed the data itself ("static bundle"), or contain a reference
to the data ("dynamic bundle"). A reference can be a URL, GitHub location, or Helm repo location.
The current list of supported bundle types is:

- manifest_inline: a single manifest yaml file embedded in the resource
- manifest_ref: a reference to a single manifest yaml file
- dir_inline: an embedded tarball that expands to a directory of YAML files
- helm_inline: an embedded Helm chart package
- helm_ref: an external reference to a Helm chart

### Static bundle

A static bundle embeds the manifest's YAML data itself ("static bundle").
A cluster consuming a static bundle will always have a snapshot copy of
the bundle at the time the cluster was created, and is not affected by subsequent
changes to the bundle's manifest data.

### Dynamic bundle

A dynamic bundle contains a reference to the manifest data stored in git.
A dynamic bundle is distinguished
by having these fields set to non-empty values:

- git URL of the repo
- Directory path within the repo

The git URL must be registered in ArgoCD as a valid repository. The content of
the specified directory can contain manifests in any of the *tool* formats supported
by ArgoCD, including plain YAML, Helm and Kustomize.

When the user updates a dynamic bundle in git, all clusters consuming that bundle
(through a profile specified at cluster creation time) will acquire the change.

### Bundle purpose

Bundles can specify an optional *purpose* to help classify and organize them.
In the future, Arlon may order bundle installation by purpose order (for e.g.
install bundles with purpose=*networking* before others) but that is not the
case today. The currently *suggested* purpose values are:

- networking
- add-on
- data-service
- application

### Other properties

A bundle can also have a comma-separated list of tags, and a description.
Tags can be useful for classifying bundles, for e.g. by type
("addon", "cni", "rbac", "app").

## Profile

A profile expresses a desired configuration for a Kubernetes cluster.
It is just a set of references to bundles (static, dynamic, or a combination).
A profile is composed of:

- An optional clusterspec. If specified, it allows the profile
  to be used to create new clusters.
  If absent, the profile can only be applied to existing clusters.
- A list of bundles specifying the configuration to apply onto the cluster
  once it is operational
- An optional list of `values.yaml` settings for any Helm Chart type bundle
  in the bundle list
A profile can be static or dynamic.

### Static profile

When a cluster consumes a static profile
at creation time, the set of bundles for the cluster is fixed at that time
and does not change over time even when the static bundle is updated.
(Note: the contents of some of those bundles referenced by the static
profile may however change over time if they are dynamic).
A static profile is stored as an item
in the Arlon database (specifically, as a CR in the Management Cluster).

### Dynamic profile

A dynamic profile, on the other hand, has two components: the specification
stored in the Arlon database, and a *compiled* component living in the workspace
repository at a path specified by the user.
(Note: this repository is usually the workspace repo, but it technically doesn't
have to be, as long as it's a valid repo registered in ArgoCD)
The compiled component is essentially a
Helm chart of multiple ArgoCD app resources, each one pointing to a bundle.
Arlon automatically creates and maintains the compiled component.
When a user updates the composition of a dynamic profile, meaning redefines its
bundle set, the Arlon library updates the compiled component to point to the
bundles specified in the new set. Any cluster
consuming that dynamic profile will be affected by the change, meaning it may lose
or acquire new bundles in real time.

## Cluster

An Arlon cluster, also called a 'workload cluster', is a Kubernetes cluster
that Arlon creates and manages via a git directory structure stored in
the workspace repository.

## Cluster Specification

A cluster spec contains desired settings when creating a new cluster.
They currently include:

- API Provider: the cluster orchestration technology. Supported values are `CAPI` (Cluster API) and `xplane` (Crossplane)
- Cloud Provider: the infrastructure cloud provider. The currently supported values is `aws`, with `gcp` and `azure` support coming later.
- Type: the cluster type. Some API providers support more than one type. On `aws` cloud, Cluster API supports `kubeadm` and `eks`, whereas Crossplane only supports `eks`.
- The (worker) node instance type
- The initial (worker) node count
- The Kubernetes version

## Base Cluster 

NOTE: The 'Base Cluster' is a new and evolved way of specifying cluster configuration. It will become the default mechanism for specifying cluster configuration starting version 0.10.0 of Arlon

The Base Cluster contains the desired settings when creating a new cluster.
They currently include:

- A predefined list of Cluster API objects: Cluster, Machines, Machine Deployments, etc. to be deployed in the current namespace
- The specific infrastructure provider to be used (e.g aws)
- Kubernetes version
- Cluster nodepool type that need to be used for creating the cluster manifest (e.g eks, eks-managedmachinepool)

To know more about 'Base Cluster', read about it [here](./baseclusters.md)

## Cluster Chart

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
