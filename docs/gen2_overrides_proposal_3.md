## Gen2 cluster overrides - final proposal

This is an update to proposal of [Gen2 cluster overrides proposal-2](gen2_overrides_proposal_2.md) design to add the ability to override the Gen 2 cluster. This update doc has the final version of clusterride feature.

### Design of overrides feature for gen2 cluster

- Basically, we want to construct various clusters with patches from the same cluster template in the cluster overrides. The person uploading the cluster template in git may be an administrator, and another employee who wants to construct a cluster from the cluster template may not have access to the git repository where the cluster template is located. This is because the capability is intended to be used in an enterprise setting.
- The cluster overrides feature is built on top of the existing cluster template design. So, there won't be any changes in the design of cluster template folder.
- Let's consider an example where our manifest is a repo called arlon-bc and with the repo path bc1. So, these are the files which will be present in bc1 folder:
  1. Resouces file which is the cluster template
  2. Kustomization.yaml
  3. configurations.yaml

    Contents of the kustomization.yaml file are as follows:

    ```
    resources:
    - capi-quickstart-eks.yaml
    configurations:
    - configurations.yaml
    ```

    In this case capi-quickstart-eks.yaml is the name of the cluster template.

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
- As we can observe in the above example, the resource field in kustomization.yaml is pointing to a different repository which contains our cluster template. Exampple of how a patch file looks like:
    ```
        ---
    apiVersion: cluster.x-k8s.io/v1beta1
    kind: MachineDeployment 
    metadata:
    name: .*
    spec:
    replicas: 2
  ```

- This is how we first organize our repositories to deploy a cluster with patches
- When we structure the clusters, the path of the resulting argocd app should direct users to the cluster-specific folder. After that, the code will launch kustomize build in the patch file directory, which will result in the creation of our final manifest.
- The manifest can be printed out directly by the user without having to go through the code. This technique is declarative as a result. A user can choose to apply the manifest to the Argocd app directly from the code at the same time.
- The above patch file directory structure is only applicable to clusters which have patch files associated with them. Other clusters which are built from the cluster template directly with no patch files attached won't have any directory created anywhere in git.
- When a user deletes a cluster which has patch files associated with it. The patch files get cleaned up from the repository as well.
- This is a clear proposal of the approach which we will be following for Gen2 cluster overrides.

### Limitations:

- Since, we are pointing to the repository where our patch file is present, the argocd won't be able to detect the changes in the cluster template repository. 
- This can be an added feature as well in one way because whenever there is a change in the cluster template configuration, it won't be immediately picked up. This brings user the ability to promote cluster template changes sensibly through each of our environments.
- The cluster created using overrides approach are not completely declarative.

### UX (User experience):

A user can pass the patch file for the cluster as an argument while executing the `arlon cluster create ..` command with the --override flag. The command would look like:

`arlon cluster create --cluster-name <cluster-name> --repo-url <repo url where cluster template is present> --repo-path <repo path to the cluster template> --overrides-path <path to the patch file> --patch-repo-url <repo url where patch file should be stored> --patch-repo-path <repo path to store the patch file>` 

The above command will create a cluster named folder in patch repo url which contains all the patch files and the argocd app created for the respective cluster will be pointing to the cluster named folder which has been created.

To delete a cluster, a user can follow the same command as usual,

`arlon cluster delete <cluster-name>`

This command checks if the cluster is overriden and if it is overrides, then the code first deletes the associated cluster named folder and then it deletes the argocd app.