# Concepts

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
Each bundle is defined using a Kubernetes ConfigMap resource in the arlon namespace.

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

### Other properties

A bundle can also have a comma-separated list of tags, and a description.
Tags can be useful for classifying bundles, for e.g. by type
("addon", "cni", "rbac", "app").

## Profile

A profile expresses a desired configuration for a Kubernetes cluster.
It is just a set of references to bundles (static, dynamic, or a combination).
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

An Arlon cluster, also known as workload cluster, is a Kubernetes cluster
that Arlon creates and manages via a git directory structure stored in
the workspace repository.

(Under construction)

## Cluster spec

A cluster spec contains desired settings when creating a new cluster.
They currently include:

- API Provider: the cluster orchestration technology. Supported values are `CAPI` (Cluster API) and `xplane` (Crossplane)
- Cloud Provider: the infrastructure cloud provider. The currently supported values is `aws`, with `gcp` and `azure` support coming later.
- Type: the cluster type. Some API providers support more than one type. On `aws` cloud, Cluster API supports `kubeadm` and `eks`, whereas Crossplane only supports `eks`.
- The (worker) node instance type
- The initial (worker) node count
- The Kubernetes version

## Cluster Template

To know more about cluster template (Arlon gen2 clusters), read it [here](./clustertemplate.md)
