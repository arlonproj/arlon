
# Introduction

## ![logo](./images/logo_arlon.svg)

Arlon is a lifecycle management and configuration tool for Kubernetes clusters.
It allows an administrator to compose, deploy and configure a large number of
*workload clusters* in a structured, predictable way.
Arlon takes advantage of multiple declarative cluster management API providers for the
actual cluster orchestration: the first two supported API providers are
Cluster API and Crossplane.
Arlon uses ArgoCD as the underlying Kubernetes manifest deployment
and enforcement engine.
A workload cluster is composed of the following constructs:

- *Cluster spec*: a description of the infrastructure and external settings of a cluster,
e.g. Kubernetes version, cloud provider, cluster type, node instance type.
- *Profile*: a grouping of configuration bundles which will be installed into the cluster
- *Configuration bundle*: a unit of configuration which contains (or references) one or
more Kubernetes manifests. A bundle can encapsulate anything that can be deployed onto a cluster:
an RBAC ruleset, an add-on, an application, etc...

## Arlon Benefits

- Unifies infrastructure and application management
- Improves time to market by enabling better velocity for developers through infrastructure management that is more fluid and agile
- Reduces the risk of unexpected infrastructure & application downtime and outages - with consistent management of infrastructure and applications  
- Allows IT and Platform Ops admins to operate large scale of clusters & applications with significantly reduced team size & operational overhead

# Contents

- [Concepts](./concepts.md) 
- [Installation](./installation.md)
- [Tutorial (gen-1)](./tutorial.md)
- [Tutorial (gen-2)](./gen2_Tutorial.md)
- [Architecture](./architecture.md)
