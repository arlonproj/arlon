
# Installation

For a quickstart minimal demonstration setup, follow the instructions to set up a KIND based testbed with Arlon and ArgoCD running  [here](https://github.com/arlonproj/arlon/blob/main/testing/README.md).

Please follow the manual instructions in [this](#customised-setup) section for a customised setup or refer the instructions for automated installation [here](#automatic-setup).

# Pre-requisites

- A 'Management cluster'. You can use any Kubernetes cluster that you have admin access to.
- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) command line tool is installed and is in your path
- Have a valid [kubeconfig](https://kubernetes.io/docs/tasks/access-application-cluster/configure-access-multiple-clusters/) file (default location is `~/.kube/config`).
- `KUBECONFIG` environment variable is pointing to the right file and the context is set properly
- A hosted Git repository that will be used to store arlon artifacts, with at least a `README` file present.
- Pre-requisites for supported Cluster API infrastructure providers (AWS and Docker as of now).

# Automatic Setup

## 1. Download Arlon CLI

Arlon CLI downloads are provided on GitHub. The CLI is not a self-contained standalone executable though.
It is required to point the CLI to a management cluster and set up the Arlon controller in this management cluster.

* Download the CLI for the [latest release](https://github.com/arlonproj/arlon/releases/latest) from GitHub.
Currently, Linux and MacOS operating systems are supported.
* Uncompress the tarball, rename it as `arlon` and add to your PATH
* Run `arlon verify` to check for prerequisites.
* Run `arlon install` to install any missing prerequisites.


## 2. Setup Arlon  

Arlon CLI provides an `init` command to install "itself" on a management cluster.
This command performs a basic setup of `argocd`(if needed) and `arlon` controller.
If `argocd` is already installed, it assumes that `admin` password is the same as in `argocd-initial-admin-secret` ConfigMap and that `argocd` resides in the `argocd` namespace.
Similar assumptions are made for detecting Arlon installation as well: assuming that the existence of `arlon` namespace means Arlon controller exists.

* To start the installation process, run the following command 
`arlon init -e --username <GIT_USER> --repoURL <WORKSPACE_URL> --password <GIT_PASSWORD> --examples -y`.
* This installs the controller, argocd(if not already present) 
* `-e` flag adds basecluster manifests to the <WORKSPACE_URL> for using the given credentials. To not add examples, just remove the `-e` flag.
* The `-y` flag refers to silent installation, which is useful for scripts. For an interactive installation, exclude the `-y` or `--no-confirm` flag.

# Customized Setup

Use the customized setup if you would like to understand and potentially customize the different steps of the Arlon installation. For example, if you'd like to use Arlon with an existing instalation of ArgoCD. 

## Management cluster

You can use any Kubernetes cluster that you have admin access to. Ensure:

- `kubectl` is in your path
- `KUBECONFIG` is pointing to the right file and the context is set properly

## ArgoCD

- Follow steps 1-4 of the [ArgoCD installation guide](https://argo-cd.readthedocs.io/en/stable/getting_started/) to install ArgoCD onto your management cluster.
After this step, you should be logged in as `admin` and a config file was created at `${HOME}/.config/argocd/config`
- Create your workspace repository in your git provider if necessary, then register it.
  Example: `argocd repo add https://github.com/myname/arlon_workspace --username myname --password secret`.
   --  Note: type `argocd repo add --help` to see all available options.
   --  For Arlon developers, this is not your fork of the Arlon source code repository,
       but a separate git repo where some artifacts like profiles created by Arlon will be stored.
- Highly recommended: [configure a webhook](https://argo-cd.readthedocs.io/en/stable/operator-manual/webhook/)
  to immediately notify ArgoCD of changes to the repo. This will be especially useful
  during the tutorial. Without a webhook, repo changes may take up to 3 minutes
  to be detected, delaying cluster configuration updates.
- [Create a local user](https://argo-cd.readthedocs.io/en/stable/operator-manual/user-management/) named `arlon` with the `apiKey` capability.
  This involves editing the `argocd-cm` ConfigMap using `kubectl`.
- Adjust the RBAC settings to grant admin permissions to the `arlon` user.
  This involves editing the `argocd-rbac-cm` ConfigMap to add the entry
  `g, arlon, role:admin` under the `policy.csv` section. Example:

```yaml
apiVersion: v1
data:
  policy.csv: |
    g, arlon, role:admin
kind: ConfigMap
[...]
```

- Generate an account token: `argocd account generate-token --account arlon`
- Make a temporary copy of this [config-file](https://github.com/arlonproj/arlon/blob/main/testing/argocd-config-for-controller.template.yaml) in `/tmp/config` then
  edit it to replace the value of `auth-token` with the token from
  the previous step. Save changes. This file will be used to configure the Arlon
  controller's ArgoCD credentials during the next steps.

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

Download the CLI for the [latest release](https://github.com/arlonproj/arlon/releases/latest) from GitHub.
Currently, Linux and MacOS operating systems are supported.
Uncompress the tarball, rename it as `arlon` and add to your PATH

Run `arlon verify` to check for prerequisites.
Run `arlon install` to install any missing prerequisites.

The following instructions are to manually build CLI from this code repository.

### Building the CLI

- Clone this repository and pull the latest version of a branch (main by default)
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

```shell
I0222 17:31:14.112689   27922 request.go:668] Waited for 1.046146023s due to client-side throttling, not priority and fairness, request: GET:https://AA61XXXXXXXXXXX.gr7.us-west-2.eks.amazonaws.com/apis/servicediscovery.aws.crossplane.io/v1alpha1?timeout=32s
```

