# Arlon

Arlon is a lifecycle management and configuration tool for Kubernetes clusters.
It allows an administrator to compose, deploy and configure a large number of
*workload clusters* in a structured, predictable way.
Arlon takes advantage of multiple declarative cluster management API providers for the
actual cluster orchestration: the first two supported API providers are
Cluster API and Crossplane.
Arlon uses ArgoCD as the underlying Kubernetes manifest deployment
and enforcement engine.
A workload cluster is composed from the following constructs:
- *Cluster spec*: a description of the infrastructure and external settings of a cluster,
e.g. Kubernetes version, cloud provider, cluster type, node instance type.
- *Profile*: a grouping of configuration bundles which will be installed into the cluster
- *Configuration bundle*: a unit of configuration which contains (or references) one or
more Kubernetes manifests. A bundle can encapsulate anything that can be deployed onto a cluster:
an RBAC ruleset, an add-on, an application, etc... 

# Architecture

Arlon is composed of a controller, a library, and a CLI that exposes the library's
functions as commands. In the future, an API server may be built from
the library as well. 

## Management cluster
The management cluster is a Kubernetes cluster hosting all the components
needed by Arlon, including:
- The ArgoCD server
- The Arlon "database" (implemented as Kubernetes secrets and configmaps)
- The Arlon controller
- Cluster management API providers: Cluster API or Crossplane
- Custom resources (CRs) that drive the involved providers and controllers
- Custom resource definitions (CRDs) for all of the involved CRs

The user is responsible for supplying the management cluster, and to have
a access to a kubeconfig granting administrator permissions on the cluster.

## Controller

The Arlon controller observes and responds to changes in `clusterregistration`
custom resources. The Arlon library creates a `clusterregistration` at the
beginning of workload cluster creation,
causing the controller to wait for the cluster's kubeconfig
to become available, at which point it registers the cluster with ArgoCD to
enable manifests described by bundles to be deployed to the cluster.

## Library
The Arlon library is a Go module that contains the functions that communicate
with the Management Cluster to manipulate the Arlon state (bundles, profiles, clusterspecs)
and transforms them into git directory structures to drive ArgoCD's gitops engine. Initially, the
library is exposed via a CLI utility. In the future, it may also be embodied
into a server an exposed via a network API.

## Workspace repository
As mentioned earlier, Arlon creates and maintains directory structures in a git
repository to drive ArgoCD *sync* operations.
The user is responsible for supplying
this *workspace repository* (and base paths) hosting those structures.
Arlon relies on ArgoCD for repository registration, therefore the user should
register the workspace registry in ArgoCD before referencing it from Arlon data types.

# Concepts

## Configuration bundle

A configuration bundle (or just "bundle") is grouping of data files that
produce a set of Kubernetes manifests via a *tool*. This closely follows ArgoCD's
definition of *tool types*. Consequently, the list of supported bundle
types mirrors ArgoCD's supported set of manifest-producing tools.
Each bundle is defined using a Kubernetes ConfigMap resource in the arlon namespace.
A bundle can embed the manifest data itself ("static bundle"), or contain a reference
to the data stored in git ("dynamic bundle"). A dynamic bundle is distinguished
by having these fields set to non-empty values:
- git URL of the repo
- Directory path within the repo

The git URL must be registered in ArgoCD as a valid repository. The content of
the specified directory can contain manifests in any of the *tool* formats supported
by ArgoCD, including plain YAML, Helm and Kustomize.

When a dynamic bundle is updated in git, all clusters consuming that bundle
(through a profile specified at cluster creation time) will acquire the change.
In contrast, a cluster consuming a static bundle will always have a snapshot copy of
the bundle at the time the cluster was created, and is not affected by subsequent
changes to the bundle.

A bundle can also have a comma-separated list of tags, and a description.
Tags can be useful for classifying bundles, for e.g. by type
("addon", "cni", "rbac", "app").

## Cluster specification

A cluster specification contains desired settings when creating a new cluster.
They currently include:
- API Provider: the cluster orchestration technology. Supported values are `CAPI` (Cluster API) and `xplane` (Crossplane)
- Cloud Provider: the infrastructure cloud provider. The currently supported values is `aws`, with `gcp` and `azure` support coming later.
- Type: the cluster type. Some API providers support more than one type. On `aws` cloud, Cluster API supports `kubeadm` and `eks`, whereas Crossplane only supports `eks`.
- The (worker) node instance type
- The initial (worker) node count
- The Kubernetes version

## Profile

A profile expresses a desired configuration for a Kubernetes cluster.
It is just a set of references to bundles (static, dynamic, or a combination).

A profile can be static or dynamic. When a cluster consumes at static profile
at creation time, the set of bundles for the cluster is fixed at that time
and does not change over time even when the static bundle is updated.
(Note: the contents of some of those bundles referenced by the static
profile may however change over time if they are dynamic).
A static profile is stored as an item
in the Arlon database (specifically, as a configmap in the Management Cluster).

A dynamic profile, on the other hand, has two components: the specification
stored in the Arlon database, and a *compiled* component living in a git
repo+path specified by the user. The compiled component is essentially a
Helm chart of multiple ArgoCD app resources, each one pointing to a bundle.
When a user updates the composition of a dynamic profile, meaning redefines its
bundle set, the Arlon library updates the compiled component. Any cluster
consuming that dynamic profile will be affected by the change, meaning it may lose
or acquire new bundles in real time.

## Cluster chart

The cluster chart is a Helm chart that creates (and optionally applies) the
manifests necessary to create a cluster and deploy desired configurations
and applications to it. When a user uses Arlon to create and configure a cluster,
he or she specifies a profile. The profile's cluster specification, bundle
list and other settings are used to generate values for the chart, and the
chart is deployed as a Helm release into the *arlon* namespace in the
management cluster.

Here is a summary of the kinds of resources generated and deployed by the chart:
- A unique namespace with a name based on the cluster's name. All subsequent
  resources below are created inside of that namespace.
- The stack-specific resources to create the cluster (for e.g. Cluster API resources)
- A `clusterregistration` to automatically register the cluster with ArgoCD
- A GitRepoDir to automatically create a git repo and/or directory to host a copy
  of the expanded bundles. Every bundle referenced by the profile is
  copied/unpacked into its own subdirectory.
- One ArgoCD Application resource for each bundle.