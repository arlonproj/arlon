
# Installation

Arlon CLI downloads are provided on GitHub. The CLI is not a self-contained standalone executable though.
It is required to point the CLI to a management cluster and set up the Arlon controller in this management cluster.
It is also recommended to have the tools described [here](#setting-up-tools) to be installed for a seamless experience.

For a quickstart minimal demonstration setup, follow the instructions to set up a KIND based testbed with Arlon and ArgoCD running  [here](https://github.com/arlonproj/arlon/blob/main/testing/README.md).

Please follow the manual instructions in [this](#customised-setup) section for a customised setup or refer the instructions for automated installation [here](#automatic-setup).

# Setting up tools
To leverage the complete features provided by ArgoCD and Arlon, it is recommended to have the following tools installed:

1. `git`
1. ArgoCD CLI
1. `kubectl`

Barring the `git` installation, the Arlon CLI has the ability to install `argocd` and `kubectl` CLIs on a user's machine for Linux and macOS based systems. 
To install these tools, run `arlon install --tools-only` to download and place these executables in `~/.local/bin`. It however, falls to the user to add the 
aforementioned directory to `$PATH` if not present. This command also verifies the presence of `git` on your `$PATH`.

# Customised Setup

## Management cluster

You can use any Kubernetes cluster that you have admin access to. Ensure:

- `kubectl` is in your path
- `KUBECONFIG` is pointing to the right file and the context set properly

## ArgoCD

- Follow steps 1-4 of the [ArgoCD installation guide](https://argo-cd.readthedocs.io/en/stable/getting_started/) to install ArgoCD onto your management cluster.
-  After this step, you should be logged in as `admin` and a config file was created at `${HOME}/.config/argocd/config`
- Create your workspace repository in your git provider if necessary, then register it.
  Example: `argocd repo add https://github.com/myname/arlon_workspace --username myname --password secret`.
  -  Note: type `argocd repo add --help` to see all available options.
  -  For Arlon developers, this is not your fork of the Arlon source code repository,
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
- Copy this [config file template](https://github.com/arlonproj/arlon/blob/main/testing/argocd-config-for-controller.template.yaml) to `/tmp/config`.
  (Note: The destination file name is important: it must be `config`).
- Edit the file to add this line to the end:
  - `  auth-token: ${ACCOUNT_TOKEN}` ()
  - Note 1: replace ${ACCOUNT_TOKEN} with the token generated 2 steps above
  - Note 2: the spacing & indentation is important, ensure `auth-token:` is aligned with `name:` from the preceding line

The final file should look something like:
```
contexts:
- name: argocd-server.argocd.svc.cluster.local
  server: argocd-server.argocd.svc.cluster.local
  user: argocd-server.argocd.svc.cluster.local
current-context: argocd-server.argocd.svc.cluster.local
servers:
- grpc-web-root-path: ""
  insecure: true
  server: argocd-server.argocd.svc.cluster.local
users:
- name: argocd-server.argocd.svc.cluster.local
  auth-token: XXXXXXXX
```

It will be used to create the secret containing the Arlon
controller's ArgoCD credentials during the next step.

## Arlon controller

- Create the arlon namespace: `kubectl create ns arlon`
- Create the ArgoCD credentials secret from the temporary config file:
  `kubectl -n arlon create secret generic argocd-creds --from-file /tmp/config`
- Delete `/tmp/config`
- Clone the arlon git repo and cd to its top directory
- Create the CRDs: `kubectl apply -f config/crd/bases/`
- Deploy the controller: `kubectl apply -f deploy/manifests/`
- Ensure the controller eventually enters the Running state: `watch kubectl -n arlon get pod`

## Arlon CLI

Download the CLI for the [latest release](https://github.com/arlonproj/arlon/releases/latest) from GitHub.
Currently, Linux and macOS operating systems are supported.
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
support for Crossplane on AWS. `cluster-api` cloud provider components on 
a management cluster can be installed by following the official guide, as instructed [here](#cluster-api).
In addition to this, the Arlon CLI also ships with an `install` command to facilitate, the installation of supported 
infrastructure providers by mimicking the behaviour of `clusterctl` CLI used in the official setup instructions. The details 
for which can be found [here](#using-arlon-cli).

### Cluster API

#### Manual Installation
Using the [Cluster API Quickstart Guide](https://cluster-api.sigs.k8s.io/user/quick-start.html)
as reference, complete these steps:

- Install `clusterctl`
- Initialize the management cluster.
  In particular, follow instructions for your specific cloud provider (AWS in this example)
  Ensure `clusterctl init` completes successfully and produces the expected output.

#### Using Arlon CLI
To install `cluster-api` components on the management cluster, the `install` command provides a 
helpful wrapper around `clusterctl` CLI tool.

To install a provider, all the pre-requisites must be met as mentioned [here](#pre-requisites-for-cluster-api-providers).
After which, simply running `arlon install --capi-only --infrastructure aws,docker` will install the latest available version 
of AWS and Docker provider components onto the management cluster.


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

# Automatic Setup

Arlon CLI provides an `init` command to install "itself" on a management cluster.
This command performs a basic setup of `argocd`(if needed) and `arlon` controller.
It makes the following assumptions while installing `arlon`:

- For ArgoCD:
    - If ArgoCD is present, it is present the in namespace `argocd` and the `admin` password is the same as in `argocd-initial-admin-secret` ConfigMap.

- For Arlon:
    - assuming that the existence of `arlon` namespace means Arlon controller exists.

To install Arlon controller using the init command these pre-requisites need to be met:

1. A valid kubeconfig pointing to the management cluster.
1. Port **8080** should not be in use by other programs. Arlon init uses it to port-forward `argocd`.
1. A hosted Git repository with at least a `README` file present and a valid GitHub token([detailed here](#setting-up-the-workspace-repository)) for:
  1. Adding a repository to ArgoCD.
  1. Avoiding rate limiting of GitHub API while fetching `cluster-api` related manifests.
  1. Pushing cluster template manifests to the workspace repository.
1. Pre-requisites for supported CAPI infrastructure providers(AWS and Docker as of now) as [described below](#pre-requisites-for-cluster-api-providers).

## Setting up the workspace repository
1. Create a GitHub repository if you don't already have it.
2. Ensure that the repository at least has a README file because empty repository cannot be added to ArgoCD.
3. Create a Personal Access Token for authentication, the token will need `repo:write` scope to push the cluster template example manifests.
4. Set the `GITHUB_TOKEN` environment variable to the token create in the previous step.

## Pre-requisites for cluster-api providers
This section outlines the requirements that need to be fulfilled for installing the `cluster-api` provider components that the `init` or the `install` command installs on the management cluster.

### Docker
There are no special requirements for docker provider, as it is largely used in an experimental setups.

### AWS
The following environment variables need to be set:
- `AWS_SSH_KEY_NAME` (the SSH key name to use)
- `AWS_REGION` (region where the cluster is deployed)
- `AWS_ACCESS_KEY_ID` (access key id for the associated AWS account)
- `AWS_SECRET_ACCESS_KEY` (secret access key for the associated AWS account)
- `AWS_NODE_MACHINE_TYPE` (machine type for cluster nodes)
- `AWS_CONTROL_PLANE_MACHINE_TYPE` (machine type for control plane)
- `AWS_SESSION_TOKEN` (optional: only for MFA enabled accounts)

## Starting the installation
Once the above requirements are met, start the installation process, simply running `arlon init -e --username <GIT_USER> --repoURL <WORKSPACE_URL> --password <GIT_PASSWORD> --examples -y`.
This installs the controller, argocd(if not already present) `-e` flag adds cluster template manifests to the <WORKSPACE_URL> for using the given credentials. To not add examples, just remove the `-e` flag.
The `-y` flag refers to silent installation, which is useful for scripts.
For an interactive installation, exclude the `-y` or `--no-confirm` flag.

This command does the following:

- Installs ArgoCD if not present.
- Installs Arlon if not present.
- Creates the Arlon user account in ArgoCD with `admin` rights.
- Installs `cluster-api` with the latest versions of `docker` and `aws` providers.
- Removes the `repoctx` file created by `arlon git register` if present.
- Registers a `default` alias against the provided `repoUrl`.
- Checks for pre-existing examples, and prompts to delete and proceed(if `-y` or `--no-confirm` flag is not set, else deletes the existing examples).
- Pushes cluster template manifests to the provided GitHub repository.

The Arlon repository also hosts some examples in the `examples` directory. In particular, the `examples/declarative` directory is a set of ready to use manifests 
which allows us to deploy a cluster with the new `app` and `app-profiles`. To run these examples, simply clone the source code and run these commands:
```shell
cd <ARLON_SOURCE_REPO_PATH>
kubectl apply -f examples/declarative
```

This creates an EKS cluster with managed machine pools in the `us-west-1` region, and attaches a few example "apps" to it in the form of two `app-profiles` namely `frontent` and `backend`.
The `frontend` app-profile consists of the [guestbook](https://github.com/argoproj/argocd-example-apps/tree/master/guestbook) application and a sentinel non-existing app appropriately named `nonexistent-1`. 
Similarly, the `backend` app-profile consists of the [redis](https://github.com/bitnami/charts/tree/main/bitnami/redis) and another non-existent app called `nonexistent-2`. 
The sentinel, `nonexistent` apps are simply present to demonstrate the `description` field for the health field and the `invalidAppNames` which lists down the apps which do not exist.