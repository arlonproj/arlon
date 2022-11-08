# Architecture

![architecture](./images/architecture_diagram.png)

Arlon is composed of a controller, a library, and a CLI that exposes the library's
functions as commands. In the future, an API server may be built from
the library as well. Arlon adds CRDs (custom resource definitions) for several
custom resources such as ClusterRegistration and Profile.

## Management cluster
The management cluster is a Kubernetes cluster hosting all the components
needed by Arlon, including:

- The ArgoCD server
- The Arlon "database" (implemented as Kubernetes secrets and configmaps)
- The Arlon controller
- Cluster management API providers: Cluster API or Crossplane
- Custom resources (CRs) that drive the involved providers and controllers
- Custom resource definitions (CRDs) for all the involved CRs

The user is responsible for supplying the management cluster, and to have
access to a kubeconfig granting administrator permissions on the cluster.

## Controller

The Arlon controller observes and responds to changes in `clusterregistration`
custom resources. The Arlon library creates a `clusterregistration` at the
beginning of workload cluster creation,
causing the controller to wait for the cluster's kubeconfig
to become available, at which point it registers the cluster with ArgoCD to
enable manifests described by bundles to be deployed to the cluster.

## Library
The Arlon library is a Go module that contains the functions that communicate
with the Management Cluster to manipulate the Arlon state (bundles, profiles, clusterspecs)
and transforms them into git directory structures to drive ArgoCD's gitops engine. Initially, the
library is exposed via a CLI utility. In the future, it may also be embodied
into a server an exposed via a network API.

## Workspace repository
As mentioned earlier, Arlon creates and maintains directory structures in a git
repository to drive ArgoCD *sync* operations.
The user is responsible for supplying
this *workspace repository* (and base paths) hosting those structures.
Arlon relies on ArgoCD for repository registration, therefore the user should
register the workspace registry in ArgoCD before referencing it from Arlon data types.

Starting from release v0.9.0, Arlon now includes two commands to help with managing
various git repository URLs. With these commands in place, the `--repo-url` flag in 
commands requiring a hosted git repository is no longer needed. A more detailed explanation 
is given in the next [section](#repo-aliases).

### Repo Aliases
A repo(repository) alias allows an Arlon user to register a GitHub repository with ArgoCD and store 
a local configuration file on their system that can be referenced by the CLI to then determine 
a repository URL and fetch its credentials when needed. All commands that require a repository, support a `--repo-url` 
flag also support a `repo-alias` flag to specify an alias instead of an alias, such commands will consider the "default" 
alias to be used when no `--repo-alias` and no `--repo-url` flags are given.
There are two subcommands i.e., `arlon git register` and 
`arlon git unregister` which allow for a basic form of git repository context management.

When `arlon git register` is run it requires a repo URL, the username, the access token and 
an optional alias(which defaults to “default”)- if a “default” alias already exists, the 
repo isn’t registered with `argocd` and the alias creation fails saying that the default 
alias already exists otherwise, the repo is registered with `argocd`. 
Lastly we also write this repository information to the local configuration file. 
This contains two pieces of information for each repository- it’s URL and the alias.
The structure of the file is as shown:
```json
{
    "default": {
        "url": "",
        "alias": "default"
    },
    "repos": [
        {
            "url": "",
            "alias": "default"
        },
        {
            "url": "",
            "alias": "not_default"
       }, {}
    ]
}
```
On running `arlon git unregister ALIAS`, it removes that entry from the configuration file. 
However, it does NOT remove the repository from `argocd`. When the "default" alias is deleted, 
we also clear the "default" entry from the JSON file.

#### Examples
Given below are some examples for registering and unregistering a repository.
##### Registering Repositories
Registering a repository requires the repository link, the GitHub username(`--user`), and a personal access token(`--password`).
When the `--password` flag isn't provided at the command line, the CLI will prompt for a password(this is the recommended approach).
```shell
arlon git register https://github.com/GhUser/manifests --user GhUser
arlon git register https://github.com/GhUser/prod-manifests --user GhUser --alias prod
```
For non-interactive registrations, the `--password` flag can be used.
```shell
export GH_PAT="..."
arlon git register https://github.com/GhUser/manifests --user GhUser --password $GH_PAT
arlon git register https://github.com/GhUser/prod-manifests --user GhUser --alias prod --password $GH_PAT
```

##### Unregistering Repositories
Unregistering an alias only requires a positional argument: the repository alias.
```shell
# unregister the default alias locally
arlon git unregister default
# unregister some other alias locally
arlon git unregister prod
```
