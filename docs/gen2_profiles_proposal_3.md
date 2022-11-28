# Gen2 Profiles - Proposal 3

This is an update to [Proposal 2 for Gen2 Profiles](gen2_profiles_proposal_2.md) design
to add the ability to associate (attach) multiple AppProfiles to a cluster.

## Summary of changes

* The **arlon.io/profiles** annotation replaces the **arlon.io/profile** label
  in an Arlon cluster (represented by an ArgoCD Application resource with label
  `arlon-type=cluster-app`). The annotation stores a comma separated list of
  AppProfile names. An annotation was chosen instead of a label because
  Kubernetes label values cannot contain the comma (`,`) character.

* Arlon applications are still implemented as ArgoCD ApplicationSets.
  But the List generator replaces the Clusters generator used in Proposal 2.
  The List generator allows the Arlon controller to generate the precise
  list of clusters associated with an Arlon Application, which is computed from
  the profile-to-application and cluster-to-profile mappings at any given time.
  Every generated Application resource receives these two template paramenters:
  *

* New CLI commands and features
  * `arlon app create appName repoUrl repoPath [flags]` creates an ApplicationSet
     manifest that satisfies the requirements to serve as an initial Arlon Application.
     In particular, it includes a `List` generator with an `Elements` list of zero items.
     The Arlon AppProfile controller will update the list dynamically as needed.
  * `arlon cluster list` now displays the app profile list in the `APP_PROFILES` column.
  * `arlon cluster setappprofiles <clusterName> <commaSeparatedAppProfileNames` allows
     a user to set the list of app profiles associated with a cluster. Setting the
     list to an empty string removes all app profiles from the cluster.
