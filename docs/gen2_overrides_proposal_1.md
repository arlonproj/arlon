# Gen2 Cluster Overrides - Proposal 1

This is a design proposal doc for gen2 cluster overrides. Right now, according to our gen2 design, we can deploy multiple clusters with same specifications from one base cluster. But what if we want to deploy cluster with a different sshkeyname from the same manifest?. To allow deploying clusters with different specifications from the same base clsuter we are introducing the concept of clusteroverrides. So, clusteroverrides is being able to deploy clusters with different specs using same manifest and overriding the specs which we want to change.

We have 2 different approches to override in a cluster:

1. Converting the git repo where base cluster is present to helm charts
2. Overriding specifies fields using kustomize

In the first approach, We first let user upload the base manifest in the repo and deploy the cluster from it and then convert it into helm chart, so that we will be able to override fields in the manifest. The downside of this approach is that we don't have a specific template for base manifest, a user can use any form of the template in which case we will not be able to convert the manifest to heml chart.

So, continuing with the 2nd approach, in the kustomize approach, we create an overlay folder parallel to the basecluster folder which contains folders named with the cluster name. These cluster named folders contain the specific override files to the cluster. An example of the folder structure is as belows:

### A directory layout

    .
    ├── Basecluster                  # Basecluster folder(Contains base manifest)
    ├── Overlays                     # Contains folders specific to each cluster created from base manifest
        ├── Cluster1                 # Contains overrides corresponding to cluster1 
        ├── Cluster2                 # Contains overrides corresponding to cluster2

### Let's consider an example case to understand the kustomize approach

1. Let's consider three different clusters on AWS. The management cluster already exists.

2. Two of these clusters will run in the AWS “us-west-2” region, while the third will run in the “us-east-2” region.

3. One of the two “us-west-2” clusters will use larger instance types to accommodate more resource-intensive workloads.

Now, we need to get our gitrepo ready by pushing the basemanifest into a folder named base and for overriding, we need to create a folder for each cluster with the cluster name and place them in overlays folder which is parallel to the base folder

## Setting up the Directory Structure

To accommodate this use case, we will need to use a directory structure that supports the use of kustomize overlays. Therefore, the directory structure would look like this for the project:

    (parent)
    |- base
    |- overlays
        |- usw2-cluster1
        |- usw2-cluster2
        |- use2-cluster1

The base directory will store the base manifest for the final Cluster API manifests, as well as a kustomization.yaml file that identifies these Cluster API manifests as resources for kustomize to use.

The contents of kustomize file in base folder is as follows:

    ```yaml
    resources:
    - basemanifest.yaml
    ```

The kustomization.yaml states that the resources for cluster is in basemanifest.yaml file

## What consists in the cluster named folders?

The intriguing parts begin with the overlays. You will need to provide the cluster-specific patches kustomize will use to generate the YAML for each cluster in its own directory under the overlays directory.

With the "usw2-cluster1" overlay, let's begin. You must first comprehend what modifications must be made to the basic configuration in order to develop the appropriate configuration for this specific cluster in order to grasp what will be required.

We can use two methods for patches

1. JSON patches
2. YAML patches

In JSON patches, we have to write a JSON file to replace fields in the manifests. So, we need to write a different file for each replace and that would become hectic.

    Example of a JSON patch:
    [
    { "op": "replace",
        "path": "/metadata/name",
        "value": "usw2-cluster-1" },
    { "op": "replace",
        "path": "/spec/infrastructureRef/name",
        "value": "usw2-cluster-1" }
    ]

So, let's discuss the YAML approach which will be much easier to handle the overrides

    ---
    apiVersion: cluster.x-k8s.io/v1alpha2
    kind: Machine
    metadata:
    name: .*
    labels:
        cluster.x-k8s.io/cluster-name: "usw2-cluster1"

This will add a cluster name field to label in the manifest which is an advantage over JSON approach. We can both add and replace fields in manifest unlike just just replace in JSON aproach.

You would once more require a reference to the patch file in kustomization for this last example. Both the patch file itself and kustomization.yaml

This kustomization.yaml file will be pointing to both the basecluster manifest and patch file basically working like a link between both the basecluster and manifest.

Using this particular approach the present basecluster approach will need to take a redesign as we will need to skip the name suffix method we using before to create a manifest for each cluster respectively with their own names.

In this approach, Instead of the configurations.yaml(Needed for name suffix), we will have a folder for each cluster and argocd path pointing to the cluster folder. This will help us in skipping the name suffix method we were using before.

We will be able to basically override any of the field in manifest without any limitations before creating a cluster using this approach.

### UX(User experience)

To provide a user the freedom to completely override any part of the base manifest, we ask the user to point to a yaml file in which the fields have been overridden.

This would be easier to user as well because he/she would generate the manifest file anyway. So, they need to make changes to the already generated and point it.

But we should even take care of the point that the base manifest in the git and the overriden manifest file are comparable. Example of a command:

    ```shell
    arlon cluster create <cluster name> --repo-url <repo url> --repo-path <repo path> --overrides <path to overriden manifest file>
    ```

## Limitations

- Manifests (base and overlays) for the base cluster as well as workload clusters reside in the same repository. This means those who create the workload cluster will need write access to the base cluster repository which might not be the case in enterprises.
- So, if we consider having the manifests (base and overlays) are in different repositories, they will need a link to each other and as of now, if we update the base cluster while having manifest in one repo and patches in another repo. Argocd will not be able to take up the updated changes in base cluster manifest  
- The main goal of gen2 clusters was to remove the dependency on git to store metadata and make the clusters completely declarative unlike gen1 clusters. But here, we re-introduce a dependency on Arlon API (library) and git (state in git with dir structure)
- Although we can make this approach declarative by introducing another controller (CRD), this would increase the whole complexity of the issue and arlon.
- Using this approach, we might not be able to prefix a name in the base manifest which is an issue because, Some resources generate external resources, like AWS load balancer and we need to avoid naming conflict - hence name prefix (not sufficient) + name reference in gen2 is required
