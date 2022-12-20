# Tutorial (gen1)

This assumes that you plan to deploy workload clusters on AWS cloud, with
Cluster API ("CAPI") as the cluster orchestration API provider.
Also ensure you have set up a [workspace repository](https://github.com/arlonproj/arlon/blob/main/docs/architecture.md#workspace-repository), and it is registered as a git repo in ArgoCD. The tutorial will assume the existence of these environment variables:

- `${ARLON_REPO}`: where the arlon repo is locally checked out
- `${WORKSPACE_REPO}`: where the workspace repo is locally checked out
- `${WORKSPACE_REPO_URL}`: the workspace repo's git URL. It typically looks like `https://github.com/${username}/${reponame}.git`
- `${CLOUD_REGION}`: the region where you want to deploy example clusters and workloads (e.g. us-west-2)
- `${SSH_KEY_NAME}`: the name of a public ssh key name registered in your cloud account, to enable ssh to your cluster nodes

Additionally, for examples assuming `arlon git register`, "default" and a "prod" git repo aliases will also be given.

_Note: for the best experience, make sure your workspace repo is configured
to send change notifications to ArgoCD via a webhook. See the Installation section for details.

## Cluster specs

We first create a few cluster specs with different combinations of API providers
and cluster types (kubeadm vs EKS).
One of the cluster specs is for an unconfigured API provider (Crossplane);
this is for illustrative purposes, since we will not use it in this tutorial.

```shell
arlon clusterspec create capi-kubeadm-3node --api capi --cloud aws --type kubeadm --kubeversion v1.21.10 --nodecount 3 --nodetype t2.medium --tags devel,test --desc "3 node kubeadm for dev/test" --region ${CLOUD_REGION} --sshkey ${SSH_KEY_NAME}
arlon clusterspec create capi-eks --api capi --cloud aws --type eks --kubeversion v1.21.10 --nodecount 2 --nodetype t2.large --tags staging --desc "2 node eks for general purpose" --region ${CLOUD_REGION} --sshkey ${SSH_KEY_NAME}
arlon clusterspec create xplane-eks-3node --api xplane --cloud aws --type eks --kubeversion v1.21.10 --nodecount 4 --nodetype t2.small --tags experimental --desc "4 node eks managed by crossplane" --region ${CLOUD_REGION} --sshkey ${SSH_KEY_NAME}
```

Ensure you can now list the cluster specs:

```shell
$ arlon clusterspec list
NAME                APIPROV  CLOUDPROV  TYPE     KUBEVERSION  NODETYPE   NODECNT  MSTNODECNT  SSHKEY  CAS    CASMIN  CASMAX  TAGS          DESCRIPTION
capi-eks            capi     aws        eks      v1.21.10     t2.large   2        3           leb     false  1       9       staging       2 node eks for general purpose
capi-kubeadm-3node  capi     aws        kubeadm  v1.21.10     t2.medium  3        3           leb     false  1       9       devel,test    3 node kubeadm for dev/test
xplane-eks-3node    xplane   aws        eks      v1.21.10     t2.small   4        3           leb     false  1       9       experimental  4 node eks managed by crossplane
```

## Bundles

First create a static bundle containing raw YAML for the `guestbook`
sample application from this example file:

```shell
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

```shell
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

```shell
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

```shell
arlon profile create static-1 --static --bundles guestbook-static,xenial-static --desc "static profile 1" --tags examples
```

Secondly, create a dynamic version of the same profile. We'll store the compiled
form of the profile in the `profiles/dynamic-1` directory of the workspace repo. We don't create
it manually; instead, the arlon CLI will create it for us, and it will push
the change to git:

```shell
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

```shell
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

```shell
arlon profile create dynamic-2-calico --repo-url ${WORKSPACE_REPO_URL} --repo-base-path profiles --bundles calico,guestbook-dynamic,xenial-static --desc "dynamic test 1" --tags examples
            # OR
# using repository aliases
  # using the default alias
arlon profile create dynamic-2-calico --repo-base-path profiles --bundles calico,guestbook-dynamic,xenial-static --desc "dynamic test 1" --tags examples
  # using the prod alias
arlon profile create dynamic-2-calico --repo-alias prod --repo-base-path profiles --bundles calico,guestbook-dynamic,xenial-static --desc "dynamic test 1" --tags examples
```

Listing the profiles should show:

```shell
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

```shell
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

```shell
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

```shell
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

```shell
$ argocd cluster list
SERVER                                                                    NAME        VERSION  STATUS      MESSAGE
https://9F07DC211252C6F7686F90FA5B8B8447.gr7.us-west-2.eks.amazonaws.com  eks-1       1.18+    Successful  
https://kubernetes.default.svc                                            in-cluster  1.20+    Successful  
```

To monitor the progress of the cluster deployment, check the status of
the ArgoCD app of the same name:

```shell
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

Note: The overall tree-like organization of the apps and their health status
can be visually observed in the ArgoCD web user interface._

The cluster is fully deployed once those apps are all `Synced` and `Healthy`.
An EKS cluster typically takes 10-15 minutes to finish deploying.

## Behavioral differences between static and dynamic bundles & profiles

### Static bundle

A change to a static bundle does not affect existing clusters using that bundle
(through a profile). To illustrate this, bring up the ArgoCD UI and
open the detailed view of the `eks-1-guestbook-static` application,
which applies the `guestbook-static` bundle to the `eks-1` cluster.
Note that there is only one `guestbook-ui` pod.

Next, update the `guestbook-static` bundle to have 3 replicas of the pod:

```shell
arlon bundle update guestbook-static --from-file examples/bundles/guestbook-3replicas.yaml
```

Note that the UI continues to show one pod. Only new clusters consuming
this bundle will have the 3 replicas.

### Dynamic profile

Before discussing dynamic bundles, we take a small detour to introduce
dynamic profiles, since this will help understand the relationship between
profiles and bundles.
To illustrate how a profile can be updated, we remove `guestbook-static` bundle
from `dynamic-1` by specifying a new bundle set:

```shell
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

### Dynamic bundle

To illustrate the defining characteristic of a dynamic bundle, we first add
`guestbook-dynamic` to `dynamic-1`:

```shell
arlon profile update dynamic-1 --bundles xenial,guestbook-dynamic
```

Observe the re-appearance of the guestbook application, which is managed
by the `eks-1-guestbook-dynamic` ArgoCD app. A detailed view of the app
shows 1 guestbook-ui pod. Remember that a dynamic bundle's
manifest content is stored in git. Use these commands to change the number
of pod replicas to 3:

```shell
cd ${WORKSPACE_REPO}
git pull # to get all latest changes pushed by arlon
vim bundles/guestbook/guestbook.yaml # edit to change deployment's replicas to 3
git commit -am "increase guestbook replicas"
git push origin main
```

Observe the number of pods increasing to 3 in the UI. Any existing cluster
consuming this dynamic bundle will be updated similarly, regardless of whether
the bundle is consumed via a dynamic or static profile.

### Static profile

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

- The specific Kubernetes version must be supported by the particular implementation and release of the underlying cluster orchestration API provider cloud, and cluster type.
- In general, the control plane will be upgraded first
- Existing nodes are not typically not upgraded to the new Kubernetes version.
  Only new nodes (added as part of manual `nodeCount` change or autoscaling)
  
In the second scenario, as part of an update operation, you may choose to
associate the cluster with a different clusterspec altogether.
The rule governing the allowed property changes remains the same: the cluster
update operation is allowed if, relative to the previously associated clusterspec, the new clusterspec's properties differ only in the values listed above.

### Profile

You can specify a completely different profile when updating a cluster.
All bundles previously used will be removed from the cluster, and new ones
specified by the new profile will be applied. This is regardless of whether
the old and new profiles are static or dynamic.

### Examples

These sequence of commands updates a clusterspec to a newer Kubernetes version
and a higher node count, then upgrades the cluster to the newer specifications:

```shell
arlon clusterspec update capi-eks --nodecount 3 --kubeversion v1.19.15
arlon cluster update eks-1
```

Note that the 2nd command didn't need any flags because the clusterspec used
is the same as before.

This example updates a cluster to use a new profile `my-new-profile`:

```shell
arlon cluster update eks-1 --profile my-new-profile
```
## Enabling Cluster Autoscaler in the workload cluster:

### Bundle creation:
Register a dynamic bundle pointing to the bundles/capi-cluster-autoscaler in the Arlon repo.

To enable the cluster-autoscaler bundle, add one more parameter during cluster creation: `srcType`. This is the ArgoCD-defined application source type (Helm, Kustomize, Directory). In addition to this, the `repo-revision` parameter should also be set to a stable arlon release branch ( in this case v0.10 ).

This example creates a bundle pointing to the bundles/capi-cluster-autoscaler in Arlon repo

```shell
arlon bundle create cas-bundle --tags cas,devel,test --desc "CAS Bundle" --repo-url https://github.com/arlonproj/arlon.git --repo-path bundles/capi-cluster-autoscaler --srctype helm --repo-revision v0.10.0
```

### Profile creation:
Create a profile that contains this capi-cluster-autoscaler bundle. 

```shell
arlon profile create dynamic-cas --repo-url ${WORKSPACE_REPO_URL} --repo-base-path profiles --bundles cas-bundle --desc "dynamic cas profile" --tags examples
```

### Clusterspec creation:
Create a clusterspec with CAPI as ApiProvider and autoscaling enabled.In addition to this, the ClusterAutoscaler(Min|Max)Nodes properties are used to set 2 annotations on MachineDeployment required by the cluster autoscaler for CAPI.

```shell
arlon clusterspec create cas-spec --api capi --cloud aws --type eks --kubeversion v1.21.10 --nodecount 2 --nodetype t2.medium --tags devel,test --desc "dev/test"  --region ${REGION} --sshkey ${SSH_KEY_NAME} --casenabled
```

### Cluster creation:
Deploy a cluster from this cluster spec and profile created in the previous steps.

```shell
arlon cluster deploy --repo-url ${WORKSPACE_REPO_URL} --cluster-name cas-cluster --profile dynamic-cas --cluster-spec cas-spec
```
Consequently, this example produces the directory `clusters/cas-cluster/` in the repo. This will contain the capi-autoscaler subchart and manifests `mgmt/charts/`.
To verify its contents:

```shell
$ cd ${WORKSPACE_REPO_URL}
$ tree clusters/cas-cluster
clusters/cas-cluster
└── mgmt
    ├── Chart.yaml
    ├── charts
    │   ├── capi-aws-eks
    │   │   ├── Chart.yaml
    │   │   └── templates
    │   │       └── cluster.yaml
    │   ├── capi-aws-kubeadm
    │   │   ├── Chart.yaml
    │   │   └── templates
    │   │       └── cluster.yaml
    │   ├── capi-cluster-autoscaler
    │   │   ├── Chart.yaml
    │   │   └── templates
    │   │       ├── callhomeconfig.yaml
    │   │       └── rbac.yaml
    │   └── xplane-aws-eks
    │       ├── Chart.yaml
    │       └── templates
    │           ├── cluster.yaml
    │           └── network.yaml
    ├── templates
    │   ├── clusterregistration.yaml
    │   ├── ns.yaml
    │   ├── profile.yaml
    │   └── rbac.yaml
    └── values.yaml
```

At this point, the cluster is provisioning. To monitor the progress of the cluster deployment, check the status of
the ArgoCD app of the same name. Eventually, the ArgoCD apps will be synced and healthy.

```shell
$ argocd app list

NAME                           CLUSTER                         NAMESPACE  PROJECT  STATUS     HEALTH   SYNCPOLICY  CONDITIONS  REPO                                            PATH                             TARGET
cas-cluster                    https://kubernetes.default.svc  default    default Synced  Healthy  Auto-Prune  <none>  ${WORKSPACE_REPO_URL}   clusters/cas-cluster/mgmt        main
cas-cluster-cas-bundle           cas-cluster                 default    default  Synced     Healthy  Auto-Prune  <none>   https://github.com/arlonproj/arlon.git   bundles/capi-cluster-autoscaler  HEAD
cas-cluster-profile-dynamic-cas  https://kubernetes.default.svc  argocd     default  Synced     Healthy  Auto-Prune  <none>      ${WORKSPACE_REPO_URL} profiles/dynamic-cas/mgmt

```
