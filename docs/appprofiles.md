# Application Profiles (new in v0.10.0)

The Application Profiles feature, also known as Gen2 Profiles, is an
addition to Arlon v0.10.0 that provides a new way to describe, group,
and deploy manifests to workload clusters. The new feature introduces
these concepts:
* Arlon Application (or just "App")
* Application Profile (a.k.a. AppProfile)

The feature provides an alternative to the Bundle and Profile concepts
("the old way") of earlier versions of Arlon. Specifically,
the Arlon Application can be viewed as a replacement for Bundles,
and AppProfile is a substitute for Profile.
In release v0.10.0, the "old way" continues to be supported, but is deprecated,
meaning it will likely be retired in an upcoming release.

## Arlon Application (a.k.a. "Arlon App")

An Arlon Application is similar to a Dynamic Bundle from earlier releases.
It specifies a source of one or more manifests stored git in any
"tool" format supported by ArgoCD (YAML, Helm, kustomize, etc ...)

Internally, Arlon represents an App as a specialized ArgoCD ApplicationSet resource.
This allows you to specify the manifest source in the Template section,
while Arlon takes care of targeting the deployment to the correct workload clusters
by automatically manipulating the Generators component.
ApplicationSets owned by Arlon to represent apps are distinguished from
other ApplicationSets via the `arlon-type=application` label.
The ApplicationSet's Generators list must contain a single generator of List
type. The Arlon AppProfile controller will modify this list in real-time
to deploy to application to the right workload clusters (or no cluster at all).

While it is possible for you create and edit an ApplicationSet resource manifest
satisfying the requirements to be an Arlon App from scratch, Arlon makes
this easier with the `arlon app create --output-yaml` command. You can then
save the output to a file and edit it to your liking before applying it to
the management cluster to actually create it. (Without the ``--output-yaml` flag,
the command will apply the resource for you).

## AppProfile

An AppProfile is simply a grouping (or set) of Arlon Apps.
Unlike an Arlon Application (which is represented by an ApplicationSet resource),
an AppProfile is represented by an Arlon-specific custom resource.

An AppProfile specifies the apps it is associated with via the `appNames` list.
It is legal for `appNames` to contain names of Arlon Apps that don't exist yet.
To indicate whether some app names are currently invalid, the AppProfile controller
will update the resource's `status` section as follows:
- If all specified app names refer to valid Arlon apps, `status.health`
  is set to `healthy`.
- If one or more specified app names refer to non-existent Arlon apps,
  then `status.health` is set to `degraded`, and `status.invalidAppNames`
  lists the invalid names.

