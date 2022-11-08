# Gen2 Profiles - Proposal 2

This is an update to the previous [Proposal 1 for Gen2 Profiles](gen2_profiles_proposal_1.md) design.
The main change is the introduction of the new **AppProfile** custom resource,
which elevates profiles to first-class objects. The rest of the design
remains mostly unchanged, meaning Arlon apps are still based on ApplicationSets,
and a cluster is associated with an AppProfile by labeling it, except the
labeling is handled slightly differently (see [Labeling Algorithm](#Labeling-Algorithm)).
AppProfiles are now the source of truth for profile-to-app mappings.
A new controller was introduced to reconcile not only AppProfiles,
but clusters and ApplicationSets as well since they are all inter linked.
  
## Object model

* *Arlon Application* (or *App* for short): a thin wrapper around an ApplicationSet.
  An ApplicationSet is an Arlon Application if it has the `managed-by=arlon` label.

* *App Profile*: a uniquely named set of Arlon Applications. It is backed by a new custom resource and CRD.
  (The resource is named **AppProfile** to distinguish it from the gen1 **Profile** resource. Even though
  gen1 profiles will deprecated and eventually retired, the naming scheme avoids conflicts during the transition).

- *Arlon Cluster*: a gen2 cluster created by Arlon.
  - As a reminder, it is represented by between 2 and 3 ArgoCD Application resources:
    - The cluster application (named with the workload cluster name)
    - The arlon application (named by appending the -arlon suffix to the cluster application's name)
    - The optional profile application (named by appending -profile suffix to the cluster application's name)
  - The first application (the cluster application) is treated as the anchor for the entire set. When an Arlon Cluster
    is associated with an AppProfile, the cluster application will be labeled with the AppProfile's name.
  - An Arlon Cluster that was successfully deployed always has an associated ArgoCD cluster (thanks to the ClusterRegistration mechanism).
 
* *ArgoCD Cluster*: the set of ArgoCD clusters is a superset of Arlon clusters.

* *External Cluster*: any ArgoCD cluster that was not created by Arlon.
  So essentially `External Clusters Set = ArgoCD Clusters Set - Arlon Clusters Set`.
  A user may want to associate an external cluster with an app profile.

Observations:
- An app profile can be associated with (or "contain") any number of applications
- And an app can be associated with multiple profiles.
- A cluster is said to be associated with a profile if it is labeled with `arlon.io/profile=<profileName>`.
- A cluster can be associated with **at most one** profile. A profile may be associated (attached to) any number of clusters.

This table summarizes the actual resources backing the objects:
| Object Type        | Actual Resource        |
|--------------------|------------------------|
| Arlon Application  | ArgoCD ApplicationSet  |
| AppProfile         | AppProfile             |
| Arlon Cluster      | ArgoCD Application     |
| ArgoCD Cluster     | Kubernetes Secret      |

## Labeling Algorithm

Just like in proposal 1, associating a cluster with an app profile is done by labeling the cluster
with the profile's name, and ensuring that that name is included in the corresponding ApplicationSet's
`matchExpressions` values list. But there are some differences:
- For an Arlon cluster, which is anchored by an ArgoCD Application resource,
  the user should label the Application resource, not the corresponding ArgoCD cluster.
  The new AppProfile controller will propagate the label to the corresponding ArgoCD cluster.
  This allows the user to deploy an Arlon cluster, create and populate a profile, and associate
  the cluster to the profile all in one declarative "apply" operation. (A user can't label
  an ArgoCD cluster that doesn't exist yet)
- For non-Arlon clusters, generally referred as "external", the design allows those existing
  ArgoCD clusters to be labeled directly, but this will be managed outside of the AppProfiles controller
  and essentially the user's responsibility, and has limitations.

## Controller

A new controller was developed to not only reconcile AppProfiles, but also clusters and ApplicationSets
(those representing Arlon Applications) since they are now all inter linked through profiles.
- The main controller logic resides in `pkg/appprofile/reconcile.go` and `controllers/appprofile_controller.go`.
- Additionally, logic was added to reconcile ArgoCD applications (representing Arlon clusters) and
  ArgoCD ApplicationSets (representing Arlon apps) with the relationships defined by AppProfiles:
  - `controllers/application_controller.go`
  - `controllers/applicationset_controller.go`

The reconciliation algorithm is complex due to the number of interdependent resources.
See [Appendix A: Reconciliation Algorithm](#Appendix-A-Reconciliation-Algorithm) for details.

## Usage

### Managing applications

`arlon app list` shows the current list of Arlon applications.
It is initially empty.

The prototype does not currently support direct Arlon application creation.
(This is easy to add later as a new command)
An Arlon app has to be created manually by one of these methods:
- Create a new ApplicationSet from a YAML file with
  - The `managed-by=arlon` label
  - The spec as follows:
```
spec:
  generators:
  - clusters:
      selector:
        matchExpressions:
        - key: arlon.io/profile
          operator: In
          values: []
```
- Modify an existing ApplicationSet with the above requirements

### Managing profiles

Profiles are not first class objects. They only exist as labels referenced
by applications and placed on clusters. If a particular profile label value is not referenced from
any application, it does not exist.

`arlon ngprofile list` shows the current list of profiles, the applications
associated with each profile, and the clusters currently using each profile.
The list is constructed
by scanning all applications and determining the unique set of labels
referenced in the `matchExpressions.values[]` array of each.

To create a profile that doesn't exist yet, it needs to be added to at least
one application's label set. This is conceptually achieved by "adding the app to the profile":

`arlon app addtoprofile <appName> <profileName>`

Conversely, a profile label can be removed from an application by
"removing the app from the profile":

`arlon app removefromprofile <appname> <profileName>`

Caution: this can cause the profile to cease to exist if that was the last app referencing it.

### Associating profiles with clusters

A cluster can have at most one profile attached to it.
To attach a profile to a cluster:

`arlon nprofile attach <profilename> <clustername>`

Similarly, to detach:

`arlon nprofile detach <profilename> <clustername>`

Internally, an attach operation simply labels the cluster (via ArgoCD API)
with the `arlon.io/profile=<profileName>` key value pair.

## Appendix A: Reconciliation Algorithm

The pseudocode looks something like:

![image](arlon-gen2-profiles-reconc-algo.png)

