# Arlon Design and Concepts

## Management cluster

This Kubernetes cluster hosts the following components:
- ArgoCD
- Arlo
- Cluster management stacks e.g. Cluster API and/or Crossplane

The Arlo state and controllers reside in the arlo namespace.

## Configuration bundle

A configuration bundle (or just "bundle") is grouping of data files that
produce a set of Kubernetes manifests via a *tool*. This closely follows ArgoCD's
definition of *tool types*. Consequently, the list of supported bundle
types mirrors ArgoCD's supported set of manifest-producing tools.
Each bundle is defined using a Kubernetes ConfigMap resource in the arlo namespace.
Additionally, a bundle can embed the data itself ("static bundle"), or contain a reference
to the data ("dynamic bundle"). A reference can be a URL, github location, or Helm repo location.
The current list of supported bundle types is:

* manifest_inline: a single manifest yaml file embedded in the resource
* manifest_ref: a reference to a single manifest yaml file
* dir_inline: an embedded tarball that expands to a directory of YAML files
* helm_inline: an embedded Helm chart package
* helm_ref: an external reference to a Helm chart

### Bundle purpose

Bundles can specify an optional *purpose* to help classify and organize them.
In the future, Arlo may order bundle installation by purpose order (for e.g.
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
and applications to it. When a user uses Arlo to create and configure a cluster,
he or she specifies a profile. The profile's cluster specification, bundle
list and other settings are used to generate values for the chart, and the
chart is deployed as a Helm release into the *arlo* namespace in the
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
