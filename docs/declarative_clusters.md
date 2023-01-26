# Declarative Clusters and Overrides

The new Cluster resource type aims to solve these issues with release v0.10.0:
- Cluster overrides are not declarative (issue [#416](https://github.com/arlonproj/arlon/issues/416))
- Even though Arlon clusters are currently declarative, there are awkwardly so,
  since a cluster is composed of two ArgoCD Application resources
  (the "cluster app" and the "arlon app"). The `arlon create` helps create
  the required manifest, but the overall design may be confusing for some users.
- Cluster teardown initiated by deleting the two top-level resources does
  not work well, due to CAPI and CAPA bugs not dealing with race conditions
  very well, resulting in forever-stuck resources. One major contributor to
  this is the fact that the namespace containing all child resources is itself
  being deleted, and CAPI / CAPA controllers don't handle unexpected errors
  well during the cleanup phase.

