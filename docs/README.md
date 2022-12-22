
# Introduction

# ![logo](./images/logo_arlon.svg)

## What Is Arlon?

Arlon is a declarative, gitops based fleet management tool for Kubernetes clusters.
It allows administrators to:

- Deploy and upgrade a large number of *workload clusters*
- Secure clusters by installing and configuring policies
- Install a set of applications / add-ons on the clusters

all in a structured, predictable manner. Arlon makes Kubernetes cluster fleet management secure, version controlled, auditable and easy to perform at scale.

Arlon takes advantage of multiple declarative cluster management API providers for the
actual cluster orchestration. The first two supported API providers are Cluster API and Crossplane.
Arlon uses ArgoCD as the underlying Kubernetes manifest deployment and enforcement engine.

A workload cluster is composed of the following constructs:

- *Cluster spec*: a description of the infrastructure and external settings of a cluster,
e.g. Kubernetes version, cloud provider, cluster type, node instance type.
- *Profile*: a grouping of configuration bundles which will be installed into the cluster
- *Configuration bundle*: a unit of configuration which contains (or references) one or
more Kubernetes manifests. A bundle can encapsulate anything that can be deployed onto a cluster:
an RBAC ruleset, an add-on, an application, etc...

## Arlon Benefits

- Improves time to market by enabling better velocity for developers through infrastructure management that is more fluid and agile. Define, store, change and enforce your cluster infrastructure & application add-ons at scale.  
- Reduces the risk of unexpected infrastructure downtime and outages, or unexpected security misconfiguration, with consistent management of infrastructure and security policies.
- Allows IT and Platform Ops admins to operate large scale of clusters, infrastructure & add-ons with significantly reduced team size & operational overhead, using GitOps.
