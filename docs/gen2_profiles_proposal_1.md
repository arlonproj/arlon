# Gen2 Profiles - Proposal 1

In this design proposal, gen2 profiles are completely built on native
ArgoCD ApplicationSets and resource labels. There are no first-class
Arlon objects.

## Object model

* Arlon Application: a thin wrapper around an ApplicationSet.
  An ApplicationSet is an Arlon Application if it has the `managed-by=arlon` label.

* Profile name: any unique label value that appears in the `spec.generators[0].clusters.selector.matchExpressions.values[]`
  array of at least one Arlon application.

* Cluster: any cluster registered in ArgoCD. Not limited to clusters created by Arlon.

Observations:
- A profile can be associated with any number of applications. And an application can be associated with multiple profiles.
- A cluster is said to be associated with a profile if it is labeled with `arlon.io/profile=<profileName>`.
A cluster can be associated with at most one profile. A profile may be associated (attached to) any number of clusters.

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

## Discussion

Pros of the design:
* Lightweight, simple
* Fully declarative (no new resources introduced, relies entirely on existing ArgoCD resources)
* Does not require "workspace git repo" since a profile has no compiled component.

Cons:
* Profiles are not first class objects. A profile can cease to exist if it
  becomes unreferenced from any application. This can be confusing to users.
  For example, you can't create an empty profile and add apps to it later.
* A cluster can only have one gen2 profile attached to it. This is a result
  of the limited expressiveness of the `matchExpressions` logic.
  In contrast, any number of gen1 profiles can be attached to a cluster
  (the current implementation only allows one, but could be enhanced to allow many)
* It's impossible to specify per-cluster overrides for an application.
  That's because an ApplicationSet can be deployed to multiple clusters if
  they have a matching profile label.
* Any limitations of ApplicationSets (for e.g. lack of Sync Wave support?)
* The lightweight nature of this design may cause some to perceive Arlon's
  contribution to be very minimal (it's a thin wrapper around ArgoCD constructs).
* Relies on ApplicationSet, which is ArgoCD specific, making it harder to port Arlon
  to other gitops tools in the future, e.g. Flux. 

## Potential solutions to the profiles-are-not-firstclass-objects issue:

### The Null App

The Null App (NA) is an Arlon app (applicationset) that belongs
to (is associated with) all profiles.
Arlon ensures that the null app always exists and maintains the above invariant.
When deployed to a cluster, the NA does not change the cluster
state, so it's a no-OP. A possible implementation is to make the NA deploy the "default" namespace,
which already exists in all (most?) clusters.

Arlon CLI commands (and possibly APIs) will filter out the NA and automatically create and update it
as necessary, so the user doesn't see it in practice.

* The NA gets all profile labels, meaning all profiles "contain" the null app.
* A user can now create an empty profile. Internally, it is added to the NA's label list.
* When a profile is attached to any cluster, that cluster automatically "gets" the NA (since it's in all profiles), in addition to any other apps associated with the profile.
* When an app is "added" to a profile, meaning the profile is added to the app's labels list, the profile may not previously exist, therefore the profile is also added to the null app's label list. Therefore, when an app is added to a profile, two apps are modified.
* When an app is "removed" from a profile, meaning the profile is removed from the app's labels list, no change is made to the null app, therefore the profile remains in the null app's label list. (Actually, this behavior must change to support inconsistent states, see "declarative installation ..." section below.

### Lifecycle operations on profiles

* With the presence of the null app, profiles can appear to be first class objects with defined lifecycle operations.

* Creating an empty profile: a profile is "created" by adding its name to the null app's label list. If it already exists in the null app's list, the app is unmodified. If it already existed in another app's list, then that's fine too. That app is not modified either. At the end of this operation, which is idempotent, the profile is guaranteed to exist in at least one app.

* Deleting a profile: this deletes the profile from all apps in which it appears in the label list. The operation is idempotent. If the profile did not initially exist, a warning will be printed by no error occurs.


### Issue: declarative installation and inconsistent states

A user may want to provision profiles and applications in a declarative way, meaning with manifests and "kubectl apply -f". Those manifests contain applicationsets that satisfy the "arlon application" requirement. The user does not know about the null app. Therefore the user's declared applicationsets (with the arlon requirements) will solely completely define the arlon applications and profiles. We assume that the user has no interest in declaratively create empty profiles, only profiles that have at least one associated application.

Arlon must allow a partially inconsistent state, meaning, at any point in time, some profiles may not exist in the null app. This is fine, since the null app's only purpose is to maintain the existence of empty profiles. During an inconsistent state, profiles that exist in some apps but not in the null app are, by definition, existent, since they appear in at least one app. However, one enhancement is necessary on the "remove app from profile" operation: 
- In addition to removing the profile from the app's label list, the operation must ensure the existence of the profile in null app, meaning add it if it's not already there. This will ensure that at the end of the operation, the profile still exists in the null app. If it no longer exists anywhere else, then by definition it is empty.

## Full Custom Resource

(Under construction)

We could represent Gen2 profiles using a custom resource, either a new type, or by overloading
the existing Profile CR already used by Gen1. The downside is an increase
in implementation complexity, for e.g
* where is the source of truth for app-to-profile associations?
* what if an app refers to a profile label value not represented by any Profile CR?

A new controller would be most likely need to be developed.
