# Installation

Arlon CLI downloads are provided on GitHub. The CLI is not a self-contained standalone executable though.
It is required to point the CLI to a management cluster and set up the Arlon controller in this management cluster.

# Customised Setup

Please follow the instructions in this section for a customised setup that includes installation of ArgoCD, Arlon CLI and Arlon controller.

## Management cluster

As a prerequisite, you need a Kubernetes cluster as management cluster for Arlon. You can use any Kubernetes cluster that you have admin access to. Ensure:

- `kubectl` is in your path
- `KUBECONFIG` is pointing to the right file and the context set properly

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

### Automatic setup

Starting from version 0.10 (v0.10), Arlon CLI provides an init command to install "itself" on a management cluster. This command performs a basic setup of argocd(if needed) and arlon controller. Refer documentation for v0.10+ for the details.
