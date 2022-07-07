# Next-gen Cluster Provisioning using Base Clusters

This proposal describes a new way of provisioning workload clusters from the *Base Cluster*
construct, which replaces the current ClusterSpec. To distinguish them from current generation
clusters, the ones deployed from a base cluster are called next-gen clusters.

## Goals

- Allow users to deploy arbitrarily complex clusters using the full Cluster API feature set.
- Fully declarative and gitops compatible: a cluster deployment should be composed of one or more
self-sufficient manifests that the user can choose to either apply directly (via kubectl) or store in
git for later-stage deployment by a gitops tool (mainly ArgoCD).
- Support Linked Mode update: an update to the the base cluster should
automatically propagate to all workload clusters deployed from it.

## Profile support

While profiles are also being re-architected, the first implementation of next-gen clusters
fully integrates with current-generation profiles, which are expressed as Profile custom resources
and compiled into a set of intermediate files in a git repository.

## Architecture diagram

This example shows a base cluster named capi-quickstart used to deploy two workload
clusters cluster-a and cluster-b. The cluster cluster-a is given profile xxx,
while cluster-b is given profile yyy.

![architecture](arlon_gen2.png)

## Base Cluster

A base cluster serves as a base for creating new workload clusters. The workload clusters
are all copies of the base cluster, meaning that they inherit all characteristics of the
base cluster except for resource names, which are renamed during the cluster creation process.

### Preparation

- To create a base cluster, a user first creates a single YAML file containing the desired Cluster API
cluster and all related resources (e.g. MachineDeployments, etc...), using whatever tool the user
chooses (e.g. `clusterctl generate cluster`). The user is responsible for the correctness of the file
and resources within. Arlon will not check for errors.
- The user then commits and pushes the manifest file to a dedicated directory in a git repository.
The name of the cluster resource does not matter, it will be used as a suffix during workload
cluster creation. The directory should be unique to the file, and not contain any other files.
- If not already registered, the git repository should also be registered in ArgoCD with
the proper credentials for read/write access.

To check whether the git directory is a compliant Arlon base cluster,
the user runs:
```
arlon basecluster validategit --repo-url <repoUrl> --repo-path <pathToDirectory> [--repo-revision revision]  
```
*Note: if --repo-revision is not specified, it defaults to main.*

The command produces an error the first time because the git directory has not yet been "prepped".
To "prep" the directory to become a compliant Arlon base cluster, the user runs:
```
arlon basecluster preparegit --repo-url <repoUrl> --repo-path <pathToDirectory> [--repo-revision revision]  
```

This pushes a commit to the repo with these changes:
- A `kustomization.yaml` file is added to the directory to make the manifest customizable by Kustomize.
- A `configurations.yaml` file is added to configure the `namereference` Kustomize plugin to rename resource names.
- All `namespace` properties in the cluster manifest are removed to allow Kustomize to override the
namespace of all resources.

If prep is successful, another invocation of `arlon basecluster validategit` should succeed as well.

## Workload clusters

### Creation

Use `arlon cluster create` to create a new workload cluster from the base cluster.
The command creates 2 (or 3, if a profile is specified) ArgoCD application resources that together
make up the cluster and its contents. The general usage is:
```
arlon cluster create --cluster-name <clusterName> --repo-url <repoUrl> --repo-path <pathToDirectory> [--output-yaml] [--profile <profileName>] 
```

The command supports two modes of operation:
- With `--output-yaml`: output yaml that you can inspect, save to a file, or pipe to `kubectl apply -f`
- Without `--output-yaml`: create the application resources directly in the management cluster currently referenced by your KUBECONFIG and context.  

The profile is optional; a cluster can be created with no profile.

## Composition

A workload cluster is composed of 2 to 3 ArgoCD application resources, which are named
based on the name of the base cluster and the workload cluster. For illustration purposes,
the following discussion assumes that the base cluster is named `capi-quickstart`, the
workload cluster is named `cluster-a`, and the optional profile is named `profile-xxx`.

### Cluster app

The `cluster-a` application is the "cluster app"; it is responsible for deploying the 
base cluster resources, meaning the Cluster API manifests.
It is named directly from the workload cluster name.
The ApplicationSource in its Spec points to the base cluster's git
directory. All resources are configured to:
- Reside in the `cluster-a` namespace, which is deployed by the "arlon app" (see below)
- Be named `cluster-a-capi-quickstart`, meaning the workload cluster name followed by the
base cluster name.



