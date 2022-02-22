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
The management server is a Kubernetes cluster hosting all the components
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
library is exposed via a CLI utility. In the future, it may also be encapsulated
into the 

## Workspace repository
As mentioned earlier, Arlon creates and maintains directory structures in a git
repository to drive ArgoCD. The user is responsible for supplying the git repository
(and base paths) hosting those structures.

# Concepts

## Configuration bundle

A configuration bundle (or just "bundle") is grouping of data files that
produce a set of Kubernetes manifests via a *tool*. This closely follows ArgoCD's
definition of *tool types*. Consequently, the list of supported bundle
types mirrors ArgoCD's supported set of manifest-producing tools.
Each bundle is defined using a Kubernetes ConfigMap resource in the arlon namespace.
Additionally, a bundle can embed the data itself ("static bundle"), or contain a reference
to the data ("dynamic bundle"). A reference can be a URL, github location, or Helm repo location.
The current list of supported bundle types is:

* manifest_static: a single manifest yaml file embedded in the resource
* manifest_dynamic: a reference to a single manifest yaml file
* dir_static: an embedded tarball that expands to a directory of YAML files
* helm_static: an embedded Helm chart package
* helm_dynamic: an external reference to a Helm chart

### Bundle purpose

Bundles can specify an optional *purpose* to help classify and organize them.
In the future, Arlon may order bundle installation by purpose order (for e.g.
install bundles with purpose=*networking* before others) but that is not the
case today. The currenty *suggested* purpose values are:
- networking
- add-on
- data-service
- application


## Cluster specification

A cluster specification contains desired settings when creating a new cluster.
They currently include:
- Stack: the cluster provisioning stack, for e.g. *cluster-api* or *crossplane*
- Provider: the specific cluster management provider under that stack,
  if applicable. Example:
  for *cluster-api*, the possible values are *eks* and *kubeadm*
- Other settings that specify the "shape" of the cluster, such as the size of
  the control plane and the initial number of nodes of the data plane.
- The pod networking technology (under discussion: this may be moved to a
  bundle because most if not all CNI providers can be installed as manifests)

## Profile

A profile expresses a desired configuration for a Kubernetes cluster.
It is composed of
- An optional Cluster Specification. If specified, it allows the profile
  to be used to create new clusters.
  If absent, the profile can only be applied to existing clusters.
- A list of bundles specifying the configuration to apply onto the cluster
  once it is operational
- An optional list of `value.yaml` settings for any Helm Chart type bundle
  in the bundle list

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
- A ClusterRegistration to automatically register the cluster with ArgoCD
- A GitRepoDir to automatically create a git repo and/or directory to host a copy
  of the expanded bundles. Every bundle referenced by the profile is
  copied/unpacked into its own subdirectory.
- One ArgoCD Application resource for each bundle.