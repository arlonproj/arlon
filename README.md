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

# Contents

* [Architecture](#architecture)
* [Concepts](#concepts)
* [Installation](#installation)
* [Tutorial](#tutorial)

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

## Cluster spec

A cluster spec contains desired settings when creating a new cluster.
They currently include:
- API Provider: the cluster orchestration technology. Supported values are `CAPI` (Cluster API) and `xplane` (Crossplane)
- Cloud Provider: the infrastructure cloud provider. The currently supported values is `aws`, with `gcp` and `azure` support coming later.
- Type: the cluster type. Some API providers support more than one type. On `aws` cloud, Cluster API supports `kubeadm` and `eks`, whereas Crossplane only supports `eks`.
- The (worker) node instance type
- The initial (worker) node count
- The Kubernetes version

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
A dynamic bunlde contains a reference to the manifest data stored in git.
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
in the Arlon database (specifically, as a configmap in the Management Cluster).

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
that Arlon creates and manages via a git directory structure store in
the workspace repository.

(Under construction)

# Installation

We plan to add a CLI command to simplify initial setup. Until then, please
follow these manual instructions.

## Management cluster

You can use any Kubernetes cluster that you have admin access to. Ensure:
- `kubectl` is in your path
- `KUBECONFIG` is pointing to the right file and the context set properly

## ArgoCD

- Follow steps 1-4 of the [ArgoCD installation guide](https://argo-cd.readthedocs.io/en/stable/getting_started/) to install ArgoCD onto your management cluster.
After this step, you should be logged in as `admin` and a config file was created at `${HOME}/.argocd/config`
- Create your workspace repository in your git provider if necessary, then register it.
  Example: `argocd repo add https://github.com/myname/arlon --username myname --password secret`.
  Note: type `argocd repo add --help` to see all available options.
- [Create a local user](https://argo-cd.readthedocs.io/en/stable/operator-manual/user-management/) named `arlon` with the `apiKey` capability.
  This involves editing the `argocd-cm` ConfigMap using `kubectl`.
- Adjust the RBAC settings to grant admin permissions to the `arlon` user.
  This involves editing the `argocd-rbac-cm` ConfigMap to add the entry
  `g, arlon, role:admin` under the `policy.csv` section. Example:
```
apiVersion: v1
data:
  policy.csv: |
    g, arlon, role:admin
kind: ConfigMap
[...]
```
- Generate an account token: `argocd account generate-token --account arlon`
- Make a temporary copy of the config file: `cp ${HOME}/.argocd/config /tmp` then
  edit it to replace the value of `auth-token` with the token from
  the previous step. Save changes. This file will be used to configure the Arlon
  controller's ArgoCD credentials during the next steps.

## Arlon controller
- Create the arlon namespace: `kubectl create ns arlon`
- Create the ArgoCD credentials secret from the temporary config file:
  `kubectl -n arlon create secret generic argocd-creds --from-file /tmp/config`
- Delete the temporary config file
- Clone the arlon git repo and cd to its top directory
- Create the `clusterregistrations` CRD: `kubectl apply -f config/crd/bases/arlon.io_clusterregistrations.yaml`
- Deploy the controller: `kubectl apply -f deploy/manifests/`
- Ensure the controller eventually enters the Running state: `watch kubectl -n arlon get pod`

## Arlon CLI
- From the top directory, run `make build`
- Optionally create a symlink from a directory
  (e.g. `/usr/local/bin`) included in your ${PATH} to the `bin/arlon` binary
  to make it easy to invoke the command.

## Cluster orchestration API providers

Arlon currently supports Cluster API on AWS cloud. It also has experimental
support for Crossplane on AWS.

### Cluster API
Using the [Cluster API Quickstart Guide](https://cluster-api.sigs.k8s.io/user/quick-start.html)
as reference, complete these steps:
- Install `clusterctl`
- Initialize the management cluster.
  In particular, follow instructions for your specific cloud provider (AWS in this example)
  Ensure `clusterctl init` completes successfully and produces the expected output.

### Crossplane (experimental)

Using the [Upbound AWS Reference Platform Quickstart Guide](https://github.com/upbound/platform-ref-aws#quick-start)
as reference, complete these steps:
- [Install UXP on your management cluster](https://github.com/upbound/platform-ref-aws#installing-uxp-on-a-kubernetes-cluster)
- [Install Crossplane kubectl extension](https://github.com/upbound/platform-ref-aws#install-the-crossplane-kubectl-extension-for-convenience)
- [Install the platform configuration](https://github.com/upbound/platform-ref-aws#install-the-platform-configuration)
- [Configure the cloud provider credentials](https://github.com/upbound/platform-ref-aws#configure-providers-in-your-platform)

You do not go any further, but you're welcome to try the Network Fabric example.

FYI: *we noticed the dozens/hundreds of CRDs that Crossplane installs in the management
cluster can noticeably slow down kubectl, and you may see a warning that looks like*:
```
I0222 17:31:14.112689   27922 request.go:668] Waited for 1.046146023s due to client-side throttling, not priority and fairness, request: GET:https://AA61XXXXXXXXXXX.gr7.us-west-2.eks.amazonaws.com/apis/servicediscovery.aws.crossplane.io/v1alpha1?timeout=32s
``` 

# Tutorial

This assumes that you plan to deploy workload clusters on AWS cloud, with
Cluster API ("CAPI") as the cluster orchestration API provider.

## Cluster specs

We first create a few cluster specs with different combinations of API providers
and cluster types (kubeadm vs EKS).
One of the cluster specs is for an unconfigured API provider (Crossplane);
this is for illustrative purposes, since we will not use it in this tutorial.

```
arlon clusterspec create capi-kubeadm-3node --api capi --cloud aws --type kubeadm --kubeversion v1.18.16 --nodecount 3 --nodetype t2.medium --tags devel,test --desc "3 node kubeadm for dev/test"
arlon clusterspec create capi-eks-2node --api capi --cloud aws --type eks --kubeversion v1.18.16 --nodecount 2 --nodetype t2.large --tags staging --desc "2 node eks for general purpose"
arlon clusterspec create xplane-eks-3node --api capi --cloud aws --type eks --kubeversion v1.18.16 --nodecount 4 --nodetype t2.small --tags experimental --desc "4 node eks managed by crossplane"
```
Ensure you can now list the cluster specs:
```
$ arlon clusterspec list
NAME                APIPROV  CLOUDPROV  TYPE     KUBEVERSION  NODETYPE   NODECOUNT  TAGS          DESCRIPTION
capi-eks-2node      capi     aws        eks      v1.18.16     t2.large   2          staging       2 node eks for general purpose
capi-kubeadm-3node  capi     aws        kubeadm  v1.18.16     t2.medium  3          devel,test    3 node kubeadm for dev/test
xplane-eks-3node    capi     aws        eks      v1.18.16     t2.small   4          experimental  4 node eks managed by crossplane
```

# Implementation details


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