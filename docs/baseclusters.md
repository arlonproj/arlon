# Next-gen Cluster Provisioning using Base Clusters

This proposal describes a new way of provisioning workload clusters from the *Base Cluster*
construct, which replaces the older ClusterSpec.

## Goals

- Allow users to deploy arbitrarily complex clusters using the full Cluster API feature set.
- Fully declarative and gitops compatible: a cluster deployment should be composed of one or more
self-sufficient manifests that the user can choose to either apply directly (via kubectl) or store in
git for later-stage deployment by a gitops tool (mainly ArgoCD).
- Support Linked Mode update: an update to the the base cluster should
automatically propagate to all workload clusters deployed from it.

