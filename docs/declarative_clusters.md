# Declarative Clusters and Overrides

The new Cluster resource type aims to solve these issues with release v0.10.0:
- Cluster overrides are not declarative (issue [#416](https://github.com/arlonproj/arlon/issues/416)),
  making them incompatible with `kubectl apply/delete -f` and impossible to integrate with
  the `examples/declarative` demo.
- Even though Arlon clusters are currently declarative, there are awkwardly so,
  since a cluster is composed of two ArgoCD Application resources
  (the "cluster app" and the "arlon app"). The `arlon create` helps create
  the required manifest, but the overall design may be confusing for some users.
- Cluster teardown initiated by simultaneously deleting (for e.g. via `kubectl delete -f`)
  the two top-level resources does
  not work well, due to CAPI and CAPA bugs not dealing with race conditions
  very well, resulting in forever-stuck resources. One major contributor to
  this is the fact that the namespace containing all child resources is itself
  being deleted, and CAPI / CAPA controllers don't handle unexpected errors
  well during the cleanup phase, and keep retrying.

## The new Cluster resource

The Cluster custom resource is the new Arlon representation of a cluster.
Here is an example:

```
apiVersion: core.arlon.io/v1
kind: Cluster
metadata:
  annotations:
    arlon.io/profiles: engineering,marketing
  name: k3
  namespace: arlon
spec:
  clusterTemplate:
    path: baseclusters/mykubeadm
    revision: main
    url: https://github.com/bcle/fleet-infra.git
  override:
    patch: |
      apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
      kind: AWSCluster
      metadata:
        name: mykubeadm
      spec:
        region: us-west-2
        sshKeyName: leb
    repo:
      path: patches/k3
      revision: main
      url: https://github.com/bcle/fleet-infra.git
status:
  innerClusterName: mykubeadm
  message: cluster app creation successful
  overrideSuccessful: true
  state: created
```

The spec's `clusterTemplate` section is self-explanatory. The `override` section is optional. If present, then `override.patch` contains the raw patch string, and `override.repo` specifies the git location where the Kustomization directory containing the patch file will be created.

### Creation sequence

A new controller invoked via `arlon clustercontroller` will reconcile the resource. It follows this general sequence:
1. Validate the cluster template and write `status.innerClusterName` if successful
1. If `override` is present, then create the Kustomization directory in git using the patch content, and set `status.overrideSuccessful` if that succeeds.
1. Create the cluster's arlon application resource if not present
1. Create the cluster's cluster application resource if not present
1. Set `status.state` to `created`

If errors are encountered along the way, `status.state` is set to `retrying` and `status.message` contains a description of the message.

### Teardown

During teardown, the controller deletes the Kustomization directory in git if an override was used, then deletes the cluster application resource first and waits for it to disappear completely. It then deletes the arlon application resource. This solves most of the CAPI/CAPA race conditions causing stuck resources.

### AppProfiles integration

The AppProfile controller monitors the `arlon.io/profiles` annotation on a cluster's **cluster ArgoCD Application resource** and has no knowledge of the new Cluster resource. To allow attaching AppProfiles to the new style clusters, the Cluster controller syncs the `arlon.io/profiles` annotation from the Cluster resource to the ArgoCD Application resource. This is one way only, so if a user sets the annotation on the ArgoCD Application resource directly, the Cluster controller will be unaware, and a future modification of the Cluster's annotation will overwrite the one in the application resource. This is ok for now, since a user of the new Cluster resource is expected (and instructed) to annotate that resource, instead of the application resource.
