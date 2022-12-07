## Gen2 cluster overrides - Proposal 2
This is an update to proposal of [Gen2 cluster overrides proposal](https://github.com/arlonproj/arlon/blob/private/main/jayanth-reddy/overrides-doc/docs/overrides_proposal.md) design to add the ability to override the Gen 2 cluster. This update doc has a more clear version of the proposal.

### Broader idea of the proposal

- Basically, in the cluster overrides we want to generate different clusters with patches from the same base manifest. So, considering this feature in enterprise scope, the user who is uploading the base manifest in git might be an admin and another employee who wants to create a cluster form the base manifest might not have access to the git repo where the base manifest is present. 
- So, In this approach a user can use a different git repository to store the patches of a manifest and create the cluster from the base manifest which is in different repository.
- Let's consider an example where our manifest is a repo called arlon-bc and with the repo path bc1. So, these are the files which will be present in bc1 folder:
  1. Resouces file which is the base manifest
  2. Kustomization.yaml

    Contents of the kustomization.yaml file are as follows:

    ```
    resources:
    - capi-quickstart-eks.yaml
    ```

    In this case capi-quickstart-eks.yaml is the name of the base manifest.

- Now, let's say that our patch file is in another repository. This repository should caontain a folder named with the cluster's name and the files inside the cluster named folder are:
  
  1.  configurations.yaml
        configurations.yaml file corresponds to name suffix addition to the yaml file.
  2.  kustomization.yaml
        kustomization.yaml file contains fields for resouces, configurations and patches as shown in the example below:
        ```
        ---
        apiVersion: kustomize.config.k8s.io/v1beta1
        kind: Kustomization
        resources:
        - git::https://github.com/jayanth-tjvrr/arlon-bc//bc1
        configurations:
        - configurations.yaml
        patches:
        - target:
            group: controlplane.cluster.x-k8s.io
            version: v1beta1
            kind: MachineDeployment
            path: md-details.yaml

        ```
  3.  patch files
        These patch files contain the patches which we want to include for the cluster

- As we can observe in the above example, the resource field in kustomization.yaml is pointing to a different repository which contains our base manifest.

- This is how we first organize our repositories to deploy a cluster with patches
- Once, we organize the clusters, while creating the argocd app, the path of the argocd app should point to the cluster named folder. The code will then run kustomize build in the patch file directory and that will produce our final manifest.
- A user can just print out the manifest without letting the code apply it. This makes this approach declarative. At the same time, a user can even opt to apply the manifest to argocd app from the code.
- This is a clear proposal of the approach which we will be following for Gen2 cluster overrides.

### Limitations:

- Since, we are pointing to the repository where our patch file is present, the argocd won't be able to detect the changes in the base manifest repository. -
- This can be an added feature aw well in one way because whenever there is a change in the base configuration, it won't be immediately picked up. This brings user the ability to promote base changes sensibly through each of our environments.
- Clean up of the cluster named folders when a user deletes a cluster is one of the issue which is still being addressed and looked up on.

### UX (User experience):

A user can pass the patch file for the cluster as an argument while executing the `arlon cluster create ..` command with the --override flag. The command would look like:

`arlon cluster create <cluster-name> --repo-url <repo url where patch files should be present> --repo-path <repo path to cluster named folder> --override <path to the patch file>` 