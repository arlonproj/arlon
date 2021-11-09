# Arlo
A companion to ArgoCD for managing both kubernetes cluster lifecycle and configuration in a declarative & gitops way.

# Architecture

Arlo introduces a few controllers and custom resources.

![architecture](./docs/architecture_diagram.png)

* A ClusterWatch resource causes another resource representing a target Kubernetes cluster provisioned by an external system like Cluster API or Crossplane to be observed.
* When the target cluster becomes ready, a ClusterRegistration containing the cluster's information and access credentials is created.
* The arlo controller uses the information in ClusterRegistration to add the cluster to ArgoCD, then sets the resource's state to **complete**.