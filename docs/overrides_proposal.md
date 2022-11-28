# Cluster Overrides - Proposal
This is a design proposal for gen2 cluster overrides. As of now, we have 2 different approaches discussed for overriding the clusters. This proposal is for the kustomize approach. In the kustomize approach, we create an overlay folder parallel to the basecluster folder which contains folders named with the cluster name. These cluster named folders contain the specific overrides to the cluster. An example of the folder structure is as belows:

### A folder directory layout

    .
    ├── Basecluster                  # Basecluster folder(Contains base manifest)
    ├── Overlays                     # Contains folders specific to each cluster created from base manifest
        ├── Cluster1                 # Contains overrides corresponding to cluster1 
        ├── Cluster2                 # Contains overrides corresponding to cluster2


In the argocd app, instead of where previously we used to point our path to the basemanifest folder we will point it to the specific cluster folder in overlays folder.

This will help argocd to create a manifest which has all the overrides specific to a cluster roped in.

###Let's consider an example case to understand the information

1. Three different clusters on AWS are needed. The management cluster already exists.

2. Two of these clusters will run in the AWS “us-west-2” region, while the third will run in the “us-east-2” region.

3. One of the two “us-west-2” clusters will use larger instance types to accommodate more resource-intensive workloads.

## Setting up the Directory Structure

To accommodate this use case, we will need to use a directory structure that supports the use of kustomize overlays. Therefore, the directory structure would look like this for the project:

    (parent)
    |- base
    |- overlays
        |- usw2-cluster1
        |- usw2-cluster2
        |- use2-cluster1

The base directory will store the base manifest for the final Cluster API manifests, as well as a kustomization.yaml file that identifies these Cluster API manifests as resources for kustomize to use.

Every overlay subdirectory will include a customization as well. The base resources will be patched using the yaml file and different patch files to create the final manifests.

## Creating the overlays

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