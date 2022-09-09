# Arlon
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Farlonproj%2Farlon.svg?ref=badge_small)
[![Go Report Card](https://goreportcard.com/badge/github.com/arlonproj/arlon)](https://goreportcard.com/report/github.com/arlonproj/arlon)
[![Documentation Status](https://readthedocs.org/projects/arlon/badge/?version=latest)](https://arlon.readthedocs.io/en/latest/?badge=latest)

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

# Contents

* [Architecture](#architecture)
* [Concepts](#concepts)
* [Installation](#installation)
* [Tutorial](#tutorial)

# Architecture

![architecture](./docs/architecture_diagram.png)

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
Given below are some examples for registering an unregistering a repository.
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

# Concepts

## Cluster spec

A cluster spec contains desired settings when creating a new cluster.
They currently include:
- API Provider: the cluster orchestration technology. Supported values are `CAPI` (Cluster API) and `xplane` (Crossplane)
- Cloud Provider: the infrastructure cloud provider. The currently supported values is `aws`, with `gcp` and `azure` support coming later.
- Type: the cluster type. Some API providers support more than one type. On `aws` cloud, Cluster API supports `kubeadm` and `eks`, whereas Crossplane only supports `eks`.
- The (worker) node instance type
- The initial (worker) node count
- The Kubernetes version

## Configuration bundle

A configuration bundle (or just "bundle") is grouping of data files that
produce a set of Kubernetes manifests via a *tool*. This closely follows ArgoCD's
definition of *tool types*. Consequently, the list of supported bundle
types mirrors ArgoCD's supported set of manifest-producing tools.
Each bundle is defined using a Kubernetes ConfigMap resource in the arlon namespace.

### Static bundle
A static bundle embeds the manifest's YAML data itself ("static bundle").
A cluster consuming a static bundle will always have a snapshot copy of
the bundle at the time the cluster was created, and is not affected by subsequent
changes to the bundle's manifest data.

### Dynamic bundle
A dynamic bunlde contains a reference to the manifest data stored in git.
A dynamic bundle is distinguished
by having these fields set to non-empty values:
- git URL of the repo
- Directory path within the repo

The git URL must be registered in ArgoCD as a valid repository. The content of
the specified directory can contain manifests in any of the *tool* formats supported
by ArgoCD, including plain YAML, Helm and Kustomize.

When the user updates a dynamic bundle in git, all clusters consuming that bundle
(through a profile specified at cluster creation time) will acquire the change.

### Other properties
A bundle can also have a comma-separated list of tags, and a description.
Tags can be useful for classifying bundles, for e.g. by type
("addon", "cni", "rbac", "app").

## Profile

A profile expresses a desired configuration for a Kubernetes cluster.
It is just a set of references to bundles (static, dynamic, or a combination).
A profile can be static or dynamic.

### Static profile
When a cluster consumes a static profile
at creation time, the set of bundles for the cluster is fixed at that time
and does not change over time even when the static bundle is updated.
(Note: the contents of some of those bundles referenced by the static
profile may however change over time if they are dynamic).
A static profile is stored as an item
in the Arlon database (specifically, as a configmap in the Management Cluster).

### Dynamic profile
A dynamic profile, on the other hand, has two components: the specification
stored in the Arlon database, and a *compiled* component living in the workspace
repository at a path specified by the user.
(Note: this repository is usually the workspace repo, but it technically doesn't
have to be, as long as it's a valid repo registered in ArgoCD)
The compiled component is essentially a
Helm chart of multiple ArgoCD app resources, each one pointing to a bundle.
Arlon automatically creates and maintains the compiled component.
When a user updates the composition of a dynamic profile, meaning redefines its
bundle set, the Arlon library updates the compiled component to point to the
bundles specified in the new set. Any cluster
consuming that dynamic profile will be affected by the change, meaning it may lose
or acquire new bundles in real time.

## Cluster

An Arlon cluster, also known as workload cluster, is a Kubernetes cluster
that Arlon creates and manages via a git directory structure store in
the workspace repository.

(Under construction)

# Installation

We plan to add a CLI command to simplify initial setup. Until then, please
follow these manual instructions.

## Management cluster

You can use any Kubernetes cluster that you have admin access to. Ensure:
- `kubectl` is in your path
- `KUBECONFIG` is pointing to the right file and the context set properly

## ArgoCD

- Follow steps 1-4 of the [ArgoCD installation guide](https://argo-cd.readthedocs.io/en/stable/getting_started/) to install ArgoCD onto your management cluster.
After this step, you should be logged in as `admin` and a config file was created at `${HOME}/.config/argocd/config`
- Create your workspace repository in your git provider if necessary, then register it.
  Example: `argocd repo add https://github.com/myname/arlon_workspace --username myname --password secret`.
   -  Note: type `argocd repo add --help` to see all available options.
   -  For Arlon developers, this is not your fork of the Arlon source code repository, 
       but a separate git repo where some artefacts like profiles created by Arlon will be stored. 
- Highly recommended: [configure a webhook](https://argo-cd.readthedocs.io/en/stable/operator-manual/webhook/)
  to immediately notify ArgoCD of changes to the repo. This will be especially useful
  during the tutorial. Without a webhook, repo changes may take up to 3 minutes
  to be detected, delaying cluster configuration updates. 
- [Create a local user](https://argo-cd.readthedocs.io/en/stable/operator-manual/user-management/) named `arlon` with the `apiKey` capability.
  This involves editing the `argocd-cm` ConfigMap using `kubectl`.
- Adjust the RBAC settings to grant admin permissions to the `arlon` user.
  This involves editing the `argocd-rbac-cm` ConfigMap to add the entry
  `g, arlon, role:admin` under the `policy.csv` section. Example:
```
apiVersion: v1
data:
  policy.csv: |
    g, arlon, role:admin
kind: ConfigMap
[...]
```
- Generate an account token: `argocd account generate-token --account arlon`
- Make a temporary copy of the config file: `cp ${HOME}/.config/argocd/config /tmp/config` then
  edit it to replace the value of `auth-token` with the token from
  the previous step. Save changes. This file will be used to configure the Arlon
  controller's ArgoCD credentials during the next steps.

NOTE: _On some operating systems, including Linux, it's possible the source configuration
file is located at `${HOME}/.argocd/config` instead. In any case, ensure that
the destination file is named `/tmp/config`, it's important for the secret creation step below_.

## Arlon controller
- Create the arlon namespace: `kubectl create ns arlon`
- Create the ArgoCD credentials secret from the temporary config file:
  `kubectl -n arlon create secret generic argocd-creds --from-file /tmp/config`
- Delete the temporary config file
- Clone the arlon git repo and cd to its top directory
- Create the CRDs: `kubectl apply -f config/crd/bases/`
- Deploy the controller: `kubectl apply -f deploy/manifests/`
- Ensure the controller eventually enters the Running state: `watch kubectl -n arlon get pod`

## Arlon CLI
- From the top directory, run `make build`
- Optionally create a symlink from a directory
  (e.g. `/usr/local/bin`) included in your ${PATH} to the `bin/arlon` binary
  to make it easy to invoke the command.

## Cluster orchestration API providers

Arlon currently supports Cluster API on AWS cloud. It also has experimental
support for Crossplane on AWS.

### Cluster API
Using the [Cluster API Quickstart Guide](https://cluster-api.sigs.k8s.io/user/quick-start.html)
as reference, complete these steps:
- Install `clusterctl`
- Initialize the management cluster.
  In particular, follow instructions for your specific cloud provider (AWS in this example)
  Ensure `clusterctl init` completes successfully and produces the expected output.

### Crossplane (experimental)

Using the [Upbound AWS Reference Platform Quickstart Guide](https://github.com/upbound/platform-ref-aws#quick-start)
as reference, complete these steps:
- [Install UXP on your management cluster](https://github.com/upbound/platform-ref-aws#installing-uxp-on-a-kubernetes-cluster)
- [Install Crossplane kubectl extension](https://github.com/upbound/platform-ref-aws#install-the-crossplane-kubectl-extension-for-convenience)
- [Install the platform configuration](https://github.com/upbound/platform-ref-aws#install-the-platform-configuration)
- [Configure the cloud provider credentials](https://github.com/upbound/platform-ref-aws#configure-providers-in-your-platform)

You do not need to go any further, but you're welcome to try the Network Fabric example.

FYI: *we noticed the dozens/hundreds of CRDs that Crossplane installs in the management
cluster can noticeably slow down kubectl, and you may see a warning that looks like*:
```
I0222 17:31:14.112689   27922 request.go:668] Waited for 1.046146023s due to client-side throttling, not priority and fairness, request: GET:https://AA61XXXXXXXXXXX.gr7.us-west-2.eks.amazonaws.com/apis/servicediscovery.aws.crossplane.io/v1alpha1?timeout=32s
``` 

# Tutorial

This assumes that you plan to deploy workload clusters on AWS cloud, with
Cluster API ("CAPI") as the cluster orchestration API provider.
Also ensure you have set up a [workspace repository](#workspace-repository),
and it is registered as a git repo in ArgoCD. The tutorial will assume
the existence of these environment variables:
- `${ARLON_REPO}`: where the arlon repo is locally checked out
- `${WORKSPACE_REPO}`: where the workspace repo is locally checked out
- `${WORKSPACE_REPO_URL}`: the workspace repo's git URL. It typically looks like `https://github.com/${username}/${reponame}.git`
- `${CLOUD_REGION}`: the region where you want to deploy example clusters and workloads (e.g. us-west-2)
- `${SSH_KEY_NAME}`: the name of a public ssh key name registered in your cloud account, to enable ssh to your cluster nodes

Additionally, for examples assuming `arlon git register`, "default" and a "prod" git repo aliases will also be given.

_Note: for the best experience, make sure your workspace repo is configured
to send change notifications to ArgoCD via a webhook. See the Installation section for details._
## Cluster specs

We first create a few cluster specs with different combinations of API providers
and cluster types (kubeadm vs EKS).
One of the cluster specs is for an unconfigured API provider (Crossplane);
this is for illustrative purposes, since we will not use it in this tutorial.

```
arlon clusterspec create capi-kubeadm-3node --api capi --cloud aws --type kubeadm --kubeversion v1.21.10 --nodecount 3 --nodetype t2.medium --tags devel,test --desc "3 node kubeadm for dev/test" --region ${CLOUD_REGION} --sshkey ${SSH_KEY_NAME}
arlon clusterspec create capi-eks --api capi --cloud aws --type eks --kubeversion v1.21.10 --nodecount 2 --nodetype t2.large --tags staging --desc "2 node eks for general purpose" --region ${CLOUD_REGION} --sshkey ${SSH_KEY_NAME}
arlon clusterspec create xplane-eks-3node --api xplane --cloud aws --type eks --kubeversion v1.21.10 --nodecount 4 --nodetype t2.small --tags experimental --desc "4 node eks managed by crossplane" --region ${CLOUD_REGION} --sshkey ${SSH_KEY_NAME}
```

Ensure you can now list the cluster specs:
```
$ arlon clusterspec list
NAME                APIPROV  CLOUDPROV  TYPE     KUBEVERSION  NODETYPE   NODECNT  MSTNODECNT  SSHKEY  CAS    CASMIN  CASMAX  TAGS          DESCRIPTION
capi-eks            capi     aws        eks      v1.21.10     t2.large   2        3           leb     false  1       9       staging       2 node eks for general purpose
capi-kubeadm-3node  capi     aws        kubeadm  v1.21.10     t2.medium  3        3           leb     false  1       9       devel,test    3 node kubeadm for dev/test
xplane-eks-3node    xplane   aws        eks      v1.21.10     t2.small   4        3           leb     false  1       9       experimental  4 node eks managed by crossplane
```

## Bundles

First create a static bundle containing raw YAML for the `guestbook`
sample application from this example file:
```
cd ${ARLON_REPO}
arlon bundle create guestbook-static --tags applications --desc "guestbook app" --from-file examples/bundles/guestbook.yaml
```
(_Note: the YAML is simply a concatenation of the files found in the
[ArgoCD Example Apps repo](https://github.com/argoproj/argocd-example-apps/tree/master/guestbook)_)

To illustrate the difference between static and dynamic bundles, we create
a dynamic version of the same application, this time using a reference to a
git directory containing the YAML. We could point it directly to the copy in the
[ArgoCD Example Apps repo](https://github.com/argoproj/argocd-example-apps/tree/master/guestbook),
but we'll want to make modifications to it, so we instead create a new directory
to host our own copy in our workspace directory:
```shell
cd ${WORKSPACE_REPO}
mkdir -p bundles/guestbook
cp ${ARLON_REPO}/examples/bundles/guestbook.yaml bundles/guestbook
git add bundles/guestbook
git commit -m "add guestbook"
git push origin main
```
```shell
arlon bundle create guestbook-dynamic --tags applications --desc "guestbook app (dynamic)" --repo-url ${WORKSPACE_REPO_URL} --repo-path bundles/guestbook
            # OR
# using repository aliases
  # using the default alias
arlon bundle create guestbook-dynamic --tags applications --desc "guestbook app (dynamic)" --repo-path bundles/guestbook
  # using the prod alias
arlon bundle create guestbook-dynamic --tags applications --desc "guestbook app (dynamic)" --repo-path bundles/guestbook --repo-alias prod
```


Next, we create a static bundle for another "dummy" application,
an Ubuntu pod (OS version: "Xenial") that does nothing but print the date-time
in an infinite sleep loop:
```
cd ${ARLON_REPO}
arlon bundle create xenial-static --tags applications --desc "xenial pod" --from-file examples/bundles/xenial.yaml
```
Finally, we create a bundle for the Calico CNI, which provides pod networking.
Some types of clusters (e.g. kubeadm) require a CNI provider to be installed
onto a newly created cluster, so encapsulating the provider as a bundle will
give us a flexible way to install it. We download a known copy from the
authoritative source and store it the workspace repo in order to create a
dynamic bundle from it:
```shell
cd ${WORKSPACE_REPO}
mkdir -p bundles/calico
curl https://docs.projectcalico.org/v3.21/manifests/calico.yaml -o bundles/calico/calico.yaml
git add bundles/calico
git commit -m "add calico"
git push origin main
```
```shell
arlon bundle create calico --tags networking,cni --desc "Calico CNI" --repo-url ${WORKSPACE_REPO_URL} --repo-path bundles/calico
            # OR
# using repository aliases
  # using the default alias
arlon bundle create calico --tags networking,cni --desc "Calico CNI" --repo-path bundles/calico
  # using the prod alias
arlon bundle create calico --tags networking,cni --desc "Calico CNI" --repo-path bundles/calico --repo-alias prod
```

List your bundles to verify they were correctly entered:
```
$ arlon bundle list
NAME               TYPE     TAGS                 REPO-URL                                             REPO-PATH              DESCRIPTION
calico             dynamic  networking,cni       ${WORKSPACE_REPO_URL}                                bundles/calico         Calico CNI
guestbook-dynamic  dynamic  applications         ${WORKSPACE_REPO_URL}                                bundles/guestbook      guestbook app (dynamic)
guestbook-static   static   applications         (N/A)                                                (N/A)                  guestbook app
xenial-static      static   applications         (N/A)                                                (N/A)                  ubuntu pod in infinite sleep loop
```

## Profiles
We can now create profiles to group bundles into useful, deployable sets.
First, create a static profile containing bundles xenial-static and guestbook-static:

```
arlon profile create static-1 --static --bundles guestbook-static,xenial-static --desc "static profile 1" --tags examples
```

Secondly, create a dynamic version of the same profile. We'll store the compiled
form of the profile in the `profiles/dynamic-1` directory of the workspace repo. We don't create
it manually; instead, the arlon CLI will create it for us, and it will push
the change to git:
```
arlon profile create dynamic-1 --repo-url ${WORKSPACE_REPO_URL} --repo-base-path profiles --bundles guestbook-static,xenial-static --desc "dynamic test 1" --tags examples
            # OR
# using repository aliases
  # using the default alias
arlon profile create dynamic-1 --repo-base-path profiles --bundles guestbook-static,xenial-static --desc "dynamic test 1" --tags examples
  # using the prod alias
arlon profile create dynamic-1 --repo-alias prod --repo-base-path profiles --bundles guestbook-static,xenial-static --desc "dynamic test 1" --tags examples
```
_Note: the `--repo-base-path profiles` option tells `arlon` to create the profile
under a base directory `profiles/` (to be created if it doesn't exist). That
is in fact the default value of that option, so it is not necessary to specify
it in this case._

To verify that the compiled profile was created correctly:
```
$ cd ${WORKSPACE_REPO}
$ git pull
$ tree profiles
profiles
├── dynamic-1
│   ├── mgmt
│   │   ├── Chart.yaml
│   │   └── templates
│   │       ├── guestbook-dynamic.yaml
│   │       ├── placeholder_configmap.yaml
│   │       └── xenial.yaml
│   └── workload
│       └── xenial
│           └── xenial.yaml
[...]
```
Since `xenial` is a static bundle, a copy of its YAML was stored in `workload/xenial/xenial.yaml`.
This is not done for `guestbook-dynamic` because it is dynamic.

Finally, we create another variant of the same profile, with the only difference
being the addition of Calico bundle. It'll be used on clusters that need a CNI provider:
```
arlon profile create dynamic-2-calico --repo-url ${WORKSPACE_REPO_URL} --repo-base-path profiles --bundles calico,guestbook-dynamic,xenial-static --desc "dynamic test 1" --tags examples
            # OR
# using repository aliases
  # using the default alias
arlon profile create dynamic-2-calico --repo-base-path profiles --bundles calico,guestbook-dynamic,xenial-static --desc "dynamic test 1" --tags examples
  # using the prod alias
arlon profile create dynamic-2-calico --repo-alias prod --repo-base-path profiles --bundles calico,guestbook-dynamic,xenial-static --desc "dynamic test 1" --tags examples
```
Listing the profiles should show:
```
$ arlon profile list
NAME              TYPE     BUNDLES                                 REPO-URL               REPO-PATH                  TAGS         DESCRIPTION
dynamic-1         dynamic  guestbook-static,xenial-static          ${WORKSPACE_REPO_URL}  profiles/dynamic-1         examples     dynamic test 1
dynamic-2-calico  dynamic  calico,guestbook-static,xenial-static   ${WORKSPACE_REPO_URL}  profiles/dynamic-2-calico  examples     dynamic test 1
static-1          static   guestbook-dynamic,xenial-static         (N/A)                  (N/A)                      examples     static profile 1
```

## Clusters (gen1)

We are now ready to deploy our first cluster. It will be of type EKS. Since
EKS clusters come configured with pod networking out of the box, we choose
a profile that does not include Calico: `dynamic-1`.
When deploying a cluster, arlon creates in git a Helm chart containing
the manifests for creating and bootstrapping the cluster.
Arlon then creates an ArgoCD App referencing the chart, thereby relying
on ArgoCD to orchestrate the whole process of deploying and configuring the cluster.
The arlon `deploy` command
accepts a git URL and path for this git location. Any git repo can be used (so long
as it's registered with ArgoCD), but we'll use the workspace cluster for
convenience:
```
arlon cluster deploy --repo-url ${WORKSPACE_REPO_URL} --cluster-name eks-1 --profile dynamic-1 --cluster-spec capi-eks
            # OR
# using repository aliases
  # using the default alias
arlon cluster deploy --cluster-name eks-1 --profile dynamic-1 --cluster-spec capi-eks
  # using the prod alias
arlon cluster deploy --repo-alias prod --cluster-name eks-1 --profile dynamic-1 --cluster-spec capi-eks
```
The git directory hosting the cluster Helm chart is created as a subdirectory
of a base path in the repo. The base path can be specified with `--base-path`, but
we'll leave it unspecified in order to use the default value of `clusters`.
Consequently, this example produces the directory `clusters/eks-1/` in the repo.
To verify its presence:
```
$ cd ${WORKSPACE_REPO}
$ git pull
$ tree clusters/eks-1
clusters/eks-1
└── mgmt
    ├── charts
    │   ├── capi-aws-eks
    │   │   ├── Chart.yaml
    │   │   └── templates
    │   │       └── cluster.yaml
    │   ├── capi-aws-kubeadm
    │   │   ├── Chart.yaml
    │   │   └── templates
    │   │       └── cluster.yaml
    │   └── xplane-aws-eks
    │       ├── Chart.yaml
    │       └── templates
    │           ├── cluster.yaml
    │           └── network.yaml
    ├── Chart.yaml
    ├── templates
    │   ├── clusterregistration.yaml
    │   ├── ns.yaml
    │   ├── profile.yaml
    │   └── rbac.yaml
    └── values.yaml
```
The chart contains several subcharts under `mgmt/charts/`,
one for each supported type of cluster. Only one of them will be enabled,
in this case `capi-aws-eks` (Cluster API on AWS with type EKS).

At this point, the cluster is provisioning and can be seen in arlon and AWS EKS:
```
$ arlon cluster list
NAME       CLUSTERSPEC  PROFILE  
eks-1      capi-eks     dynamic-1

$ aws eks list-clusters
{
    "clusters": [
        "eks-1_eks-1-control-plane",
    ]
}
```
Eventually, it will also be seen as a registered cluster in argocd, but this
won't be visible for a while, because the cluster is not registered until
its control plane (the Kubernetes API) is ready:
```
$ argocd cluster list
SERVER                                                                    NAME        VERSION  STATUS      MESSAGE
https://9F07DC211252C6F7686F90FA5B8B8447.gr7.us-west-2.eks.amazonaws.com  eks-1       1.18+    Successful  
https://kubernetes.default.svc                                            in-cluster  1.20+    Successful  
```

To monitor the progress of the cluster deployment, check the status of
the ArgoCD app of the same name:
```
$ argocd app list
NAME                         CLUSTER                         NAMESPACE  PROJECT  STATUS  HEALTH   SYNCPOLICY  CONDITIONS  REPO                   PATH                                          TARGET
eks-1                        https://kubernetes.default.svc  default    default  Synced  Healthy  Auto-Prune  <none>      ${WORKSPACE_REPO_URL}  clusters/eks-1/mgmt                           main
eks-1-guestbook-static                                       default    default  Synced  Healthy  Auto-Prune  <none>      ${WORKSPACE_REPO_URL}  profiles/dynamic-1/workload/guestbook-static  HEAD
eks-1-profile-dynamic-1      https://kubernetes.default.svc  argocd     default  Synced  Healthy  Auto-Prune  <none>      ${WORKSPACE_REPO_URL}  profiles/dynamic-1/mgmt                       HEAD
eks-1-xenial                                                 default    default  Synced  Healthy  Auto-Prune  <none>      ${WORKSPACE_REPO_URL}  profiles/dynamic-1/workload/xenial            HEAD
```
The top-level app `eks-1` is the root of all argocd apps that make up the
cluster and its configuration contents. The next level app `eks-1-profile-dynamic-1`
represents the profile, and its children apps `eks-1-guestbook-static` and `eks-1-xenial`
correspond to the bundles.

_Note: The overall tree-like organization of the apps and their health status
can be visually observed in the ArgoCD web user interface._

The cluster is fully deployed once those apps are all `Synced` and `Healthy`.
An EKS cluster typically takes 10-15 minutes to finish deploying.

## Behavioral differences between static and dynamic bundles & profiles

**Static bundle**

A change to a static bundle does not affect existing clusters using that bundle
(through a profile). To illustrate this, bring up the ArgoCD UI and
open the detailed view of the `eks-1-guestbook-static` application,
which applies the `guestbook-static` bundle to the `eks-1` cluster.
Note that there is only one `guestbook-ui` pod.

Next, update the `guestbook-static` bundle to have 3 replicas of the pod:
```
arlon bundle update guestbook-static --from-file examples/bundles/guestbook-3replicas.yaml
```
Note that the UI continues to show one pod. Only new clusters consuming
this bundle will have the 3 replicas.

**Dynamic profile**

Before discussing dynamic bundles, we take a small detour to introduce
dynamic profiles, since this will help understand the relationship between
profiles and bundles.
To illustrate how a profile can be updated, we remove `guestbook-static` bundle
from `dynamic-1` by specifying a new bundle set:
```
arlon profile update dynamic-1 --bundles xenial
```
Since the old bundle set was `guestbook-static,xenial-static`, that command resulted
in the removal of `guestbook-static` from the profile.
In the UI, observe the `eks-1-profile-dynamic-1` app going through Sync
and Progressing phases, eventually reaching the healthy (green) state.
And most importantly, the `eks-1-guestbook-static` app is gone. The reason
this real-time change to the cluster was possible is that the `dynamic-1`
profile is dynamic, meaning any change to its composition results in arlon
updating the corresponding compiled Helm chart in git. ArgoCD detects this
git change and propagates the app / configuration updates to the cluster.

If the profile were of the static type, a change in its composition (the
set of bundles) would _not_ have affected existing clusters using that profile.
It would only affect new clusters created with the profile.

**Dynamic bundle**

To illustrate the defining characteristic of a dynamic bundle, we first add
`guestbook-dynamic` to `dynamic-1`:
```
arlon profile update dynamic-1 --bundles xenial,guestbook-dynamic
```
Observe the re-appearance of the guestbook application, which is managed
by the `eks-1-guestbook-dynamic` ArgoCD app. A detailed view of the app
shows 1 guestbook-ui pod. Remember that a dynamic bundle's
manifest content is stored in git. Use these commands to change the number
of pod replicas to 3:
```
cd ${WORKSPACE_REPO}
git pull # to get all latest changes pushed by arlon
vim bundles/guestbook/guestbook.yaml # edit to change deployment's replicas to 3
git commit -am "increase guestbook replicas"
git push origin main
```
Observe the number of pods increasing to 3 in the UI. Any existing cluster
consuming this dynamic bundle will be updated similarly, regardless of whether
the bundle is consumed via a dynamic or static profile.

**Static profile**

Finally, a profile can be static. It means that it has no corresponding
"compiled" component (a Helm chart) living in git. When a cluster is
deployed using a static profile, the set of bundles (whether static or
dynamic) it receives is determined by the bundle set defined by the profile at deployment
time, and will not change in the future, even if the profile is updated to
a new set at a later time.

## Cluster updates and upgrades

The `arlon cluster update [flags]` command allows you to make changes to
an existing cluster. The clusterspec, profile, or both can change, provided
that the following rules and guidelines are followed.

### Clusterspec

There are two scenarios. In the first, the clusterspec name associated with the
cluster hasn't changed, meaning the cluster is using the same clusterspec.
However, some properties of the clusterspec's properties have changed since
the cluster was deployed or last updated, using `arlon clusterspec update`
Arlon supports updating the cluster
to use updated values of the following properties:
- kubernetesVersion
- nodeCount
- nodeType

_Note: Updating the cluster is not allowed if other properties of its clusterspec
(e.g. cluster orchestration API provider, cloud, cluster type, region, pod CIDR block, etc...)
have changed, however new clusters can always be created/deployed using the
changed clusterspec._

A change in `kubernetesVersion` will result in a cluster upgrade/downgrade.
There are some restrictions and caveats you need to be aware of:
- The specific Kubernetes version must be supported by the particular 
  implementation and release of the underlying cluster orchestration API provider,
  cloud, and cluster type.
- In general, the control plane will be upgraded first
- Existing nodes are not typically not upgraded to the new Kubernetes version.
  Only new nodes (added as part of manual `nodeCount` change or autoscaling)
  
In the second scenario, as part of an update operation, you may choose to
associate the cluster with a different clusterspec altogether. 
The rule governing the allowed property changes remains the same: the cluster
update operation is allowed if, relative to the previously associated clusterspec, 
the new clusterspec's properties differ only in the values listed above.

### Profile
You can specify a completely different profile when updating a cluster.
All bundles previously used will be removed from the cluster, and new ones
specified by the new profile will be applied. This is regardless of whether
the old and new profiles are static or dynamic.

### Examples
These sequence of commands updates a clusterspec to a newer Kubernetes version
and a higher node count, then upgrades the cluster to the newer specifications:
```
arlon clusterspec update capi-eks --nodecount 3 --kubeversion v1.19.15
arlon cluster update eks-1
```
Note that the 2nd command didn't need any flags because the clusterspec used
is the same as before.

This example updates a cluster to use a new profile `my-new-profile`:
```
arlon cluster update eks-1 --profile my-new-profile
```

# Next-Generation (gen2) Clusters - New in version 0.9.x

Gen1 clusters are limited in capability by the Helm chart used to deploy the infrastructure resources.
Advanced Cluster API configurations, such as those using multiple MachinePools, each with different
instance types, is not supported.

Gen2 clusters solve this problem by allowing you to create workload clusters from a *base cluster*
that you design and provide in the form of a manifest file stored in a git directory. The manifest
typically contains multiple related resources that together define an arbitrarily complex cluster.
If you make subsequent changes to the base cluster, workload clusters originally created from it
will automatically acquire the changes.

Here is an example of a manifest file that we can use to create a *base cluster*. This manifest file helps in 
deploying an EKS cluster with 'machine deployment' component from the cluster API (CAPI). This file has been generated by the following command

```shell
clusterctl generate cluster capi-quickstart --flavor eks \
  --kubernetes-version v1.24.0 \
  --control-plane-machine-count=3 \
  --worker-machine-count=3 \
  > capi-quickstart.yaml
```

```yaml
# YAML
apiVersion: cluster.x-k8s.io/v1beta1
kind: Cluster
metadata:
  name: capi-quickstart
  namespace: default
spec:
  clusterNetwork:
    pods:
      cidrBlocks:
      - 192.168.0.0/16
  controlPlaneRef:
    apiVersion: controlplane.cluster.x-k8s.io/v1beta1
    kind: AWSManagedControlPlane
    name: capi-quickstart-control-plane
  infrastructureRef:
    apiVersion: controlplane.cluster.x-k8s.io/v1beta1
    kind: AWSManagedControlPlane
    name: capi-quickstart-control-plane
---
apiVersion: controlplane.cluster.x-k8s.io/v1beta1
kind: AWSManagedControlPlane
metadata:
  name: capi-quickstart-control-plane
  namespace: default
spec:
  region: {REGION}
  sshKeyName: {SSH_KEYNAME}
  version: v1.24.0
---
apiVersion: cluster.x-k8s.io/v1beta1
kind: MachineDeployment
metadata:
  name: capi-quickstart-md-0
  namespace: default
spec:
  clusterName: capi-quickstart
  replicas: 3
  selector:
    matchLabels: null
  template:
    spec:
      bootstrap:
        configRef:
          apiVersion: bootstrap.cluster.x-k8s.io/v1beta1
          kind: EKSConfigTemplate
          name: capi-quickstart-md-0
      clusterName: capi-quickstart
      infrastructureRef:
        apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
        kind: AWSMachineTemplate
        name: capi-quickstart-md-0
      version: v1.24.0
---
apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
kind: AWSMachineTemplate
metadata:
  name: capi-quickstart-md-0
  namespace: default
spec:
  template:
    spec:
      iamInstanceProfile: nodes.cluster-api-provider-aws.sigs.k8s.io
      instanceType: t2.medium
      sshKeyName: {SSH_KEYNAME}
---
apiVersion: bootstrap.cluster.x-k8s.io/v1beta1
kind: EKSConfigTemplate
metadata:
  name: capi-quickstart-md-0
  namespace: default
spec:
  template: {}
```

Before a manifest directory can be used as a base cluster, it must first be "prepared" or "prepped"
by Arlon. The "prep" phase makes minor changes to the directory and manifest to help Arlon deploy
multiple copies of the cluster without naming conflicts.

To determine if a git directory is eligible to serve as base cluster, run the `basecluster validategit` command:
```shell
arlon basecluster validategit --repo-url <repoUrl> --repo-path <pathToDirectory> [--repo-revision revision]
            # OR
# using repository aliases
  # using the default alias
arlon basecluster validategit --repo-path <pathToDirectory> [--repo-revision revision]
  # using the prod alias
arlon basecluster validategit --repo-alias prod --repo-path <pathToDirectory> [--repo-revision revision]
```

To prepare a git directory to serve as base cluster, use the `basecluster preparegit` command:
```shell
arlon basecluster preparegit --repo-url <repoUrl> --repo-path <pathToDirectory> [--repo-revision revision]
            # OR
# using repository aliases
  # using the default alias
arlon basecluster preparegit --repo-path <pathToDirectory> [--repo-revision revision]
  # using the prod alias
arlon basecluster preparegit --repo-alias prod --repo-path <pathToDirectory> [--repo-revision revision]
```

To create a gen2 workload cluster from the base cluster:
```shell
arlon cluster create --cluster-name <clusterName> --repo-url <repoUrl> --repo-path <pathToDirectory> [--output-yaml] [--profile <profileName>] [--repo-revision <repoRevision>]
            # OR
# using repository aliases
  # using the default alias
arlon cluster create --cluster-name <clusterName> --repo-path <pathToDirectory> [--output-yaml] [--profile <profileName>] [--repo-revision <repoRevision>]
  # using the prod alias
arlon cluster create --cluster-name <clusterName> --repo-alias prod --repo-path <pathToDirectory> [--output-yaml] [--profile <profileName>] [--repo-revision <repoRevision>]
```

To destroy a gen2 workload cluster:
```
arlon cluster delete <clusterName>
```

Arlon creates between 2 and 3 ArgoCD application resources to compose a gen2 cluster (the 3rd application, called "profile app", is used when
an optional profile is specified at cluster creation time). When you destroy a gen2 cluster, Arlon will find all related ArgoCD applications
and clean them up.

## Known issues and limitations.
Gen2 clusters are powerful because the base cluster can be arbitrarily complex and feature rich. Since they are fairly
new and still evolving, gen2 clusters have several known limitations relative to gen1.
* You cannot customize/override any property of the base cluster on the fly when creating a workload cluster,
  which is an exact clone of the base cluster except for the names of its resources and their namespace.
  The work-around is to make a copy of the base cluster directory, push the new directory, make
  the desired changes, commit & push the changes, and register the directory as a new base cluster.
* If you modify and commit a change to one or more properties of the base cluster that the underlying Cluster API provider deems as "immutable", new
  workload clusters created from the base cluster will have the modified propert(ies), but ArgoCD will flag existing clusters as OutOfSync, since
  the provider will continually reject attempts to apply the new property values. The existing clusters continue to function, despite appearing unhealthy
  in the ArgoCD UI and CLI outputs.

Examples of mutable properties in Cluster API resources:
- Number of replicas (modification will result in a scale-up / down)
- Kubernetes version (modification will result in an upgrade)

Examples of immutable properties:
- Most fields of AWSMachineTemplate (instance type, labels, etc...)
 
## For more information

For more details on gen2 clusters, refer to the [design document](docs/baseclusters.md).

# Implementation details

## Cluster chart (gen1)

The cluster chart is a Helm chart that creates (and optionally applies) the
manifests necessary to create a cluster and deploy desired configurations
and applications to it. When a user uses Arlon to create and configure a cluster,
he or she specifies a profile. The profile's cluster specification, bundle
list and other settings are used to generate values for the chart, and the
chart is deployed as a Helm release into the *arlon* namespace in the
management cluster.

Here is a summary of the kinds of resources generated and deployed by the chart:
- A unique namespace with a name based on the cluster's name. All subsequent
  resources below are created inside that namespace.
- The stack-specific resources to create the cluster (for e.g. Cluster API resources)
- A `clusterregistration` to automatically register the cluster with ArgoCD
- A GitRepoDir to automatically create a git repo and/or directory to host a copy
  of the expanded bundles. Every bundle referenced by the profile is
  copied/unpacked into its own subdirectory.
- One ArgoCD Application resource for each bundle.


## License
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fplatform9%2Farlon.svg?type=large)](https://app.fossa.com/projects/git%2Bgithub.com%2Fplatform9%2Farlon?ref=badge_large)
