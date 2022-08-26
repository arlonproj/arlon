# Optional environment variables
#
#                         Default
# GIT_SERVER_PORT         8188
# GIT_ROOT                Create new directory under /tmp
# GIT_CLONE_ROOT          Create new directory under /tmp
# ARGOCD_GIT_TAG          release-2.4
# ARGOCD_CONFIG_FILE      Create new one under /tmp
# ARGGOCD_FORWARDING_PORT 8189

set -e
set -o pipefail

# $1 = expression
# $2 = sleep period
# $3 = iterations
function wait_until()
{
    for i in `seq 1 $3`
    do
        eval $1 && return 0
        echo "Waiting for \"$1\" to evaluate to true ..."
        sleep $2
    done
    echo Timed out waiting for \"$1\"
    return 1
}

arlon_repo=`pwd`
if ! grep git@github.com:arlonproj/arlon.git .git/config &> /dev/null ; then
    echo "it doesn't look like we are in the arlon repository"
    exit 1
fi

if ! arlon &> /dev/null; then
    echo arlon command not found in $PATH
    exit 1
fi

if ! git version > /dev/null; then
    echo git not installed
    exit 1
fi

if ! docker ps > /dev/null; then
    echo Docker not installed or operational
    exit 1
fi

if ! kind version > /dev/null; then
    echo KIND not installed
    exit 2
fi

if ! kubectl version --client > /dev/null; then
    echo kubectl not installed
    exit 3
fi

if bridge_addr=$(ip addr |grep 'scope global docker0'|awk '{print $2}'|cut -d / -f 1) ; then
    echo docker bridge address is $bridge_addr
else
    echo failed to get docker bridge address
    exit 4
fi

git_server_port=${GIT_SERVER_PORT}
if [ -z "${git_server_port}" ]; then
    git_server_port=8188
fi
echo git server port: ${git_server_port}

git_root=${GIT_ROOT}
if [ -z "${git_root}" ]; then
    git_root=$(mktemp -d /tmp/arlon-testbed-git.XXXXX)
fi
echo git root: ${git_root}

git_repo_dir=${git_root}/myrepo.git
if [ ! -d "${git_repo_dir}" ]; then
    echo initializing git repo
    mkdir ${git_repo_dir}
    pushd ${git_repo_dir}
    git init --bare
    popd
fi
echo git repo at ${git_repo_dir}

gitserver_cntr_name="arlon-testbed-gitserver"
if ! docker inspect ${gitserver_cntr_name} &> /dev/null ; then
    if ! docker run -d -v ${git_root}:/var/lib/git -p ${git_server_port}:80 --name ${gitserver_cntr_name} --rm cirocosta/gitserver-http > /dev/null ; then
        echo failed to start git server container
        exit 5
    else echo started git server container
    fi
else
    echo git server container already running
fi

git_clone_root=${GIT_CLONE_ROOT}
if [ -z "${git_clone_root}" ]; then
    git_clone_root=$(mktemp -d /tmp/arlon-testbed-gitclone.XXXXX)
fi
echo git clone root: ${git_clone_root}

workspace_repo_url=http://${bridge_addr}:${git_server_port}/myrepo.git

workspace_repo=${git_clone_root}/myrepo
if [ ! -d "${workspace_repo}" ]; then
    echo cloning git repo
    pushd ${git_clone_root}
    git clone ${workspace_repo_url}
    popd
else
    echo git repo already cloned
fi

pushd ${workspace_repo}
if ! test -f README.md ; then
    echo adding README.md and creating main branch
    echo hello > README.md
    git add README.md
    git commit -m "add README.md"
    git push origin HEAD:main
else
    echo README.md already present
fi
popd

tb_cntr_name='kind-arlon-testbed'

if ! kind get clusters | grep ${tb_cntr_name}; then
    echo testbed container not found
    if ! kind create cluster --name ${tb_cntr_name}; then
        echo failed to create cluster
        exit 6
    fi
fi

ctx_name=kind-${tb_cntr_name}

if ! kubectl config use-context ${ctx_name}; then
    echo failed to switch kubectl context
    exit 7
fi

echo waiting for cluster control plane ...
# 'kubectl version' queries the server version by default
wait_until "kubectl version &> /dev/null" 2 30

if ! kubectl get ns argocd &> /dev/null ; then
    echo creating argocd namespace
    kubectl create ns argocd
fi

argocd_git_tag=${ARGOCD_GIT_TAG}
if [ -z "${argocd_git_tag}" ]; then
    argocd_git_tag="release-2.4"
fi
echo applying argocd manifest from git tag: ${argocd_git_tag}
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/${argocd_git_tag}/manifests/install.yaml > /dev/null

if pkill -f "kubectl port-forward svc/argocd-server" ; then
    echo terminated previous port forwarding session
fi

argocd_forwarding_port=${ARGGOCD_FORWARDING_PORT}
if [ -z "${argocd_forwarding_port}" ]; then
    argocd_forwarding_port=8189
fi

kubectl port-forward svc/argocd-server -n argocd ${argocd_forwarding_port}:443 &>/dev/null &
pwd=$(kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d; echo)

wait_until "argocd login localhost:${argocd_forwarding_port} --username admin --password ${pwd} --insecure" 2 30

# This is idempotent so no need to check whether repo already registered
argocd repo add ${workspace_repo_url} --username dummy-user --password dummy-password

if ! kubectl get ns arlon &> /dev/null ; then
    echo creating arlon namespace
    kubectl create ns arlon
fi

# Arlon CRDs
kubectl apply -f config/crd/bases

# ArgoCD config maps for configuring 'arlon' user
kubectl apply -f testing/manifests

# argocd config file for arlon controller
if ! kubectl get secret argocd-creds -n arlon &> /dev/null ; then
    wait_until "auth_token=$(argocd account generate-token --account arlon)" 2 10
    echo auth_token: ${auth_token}
    # The file name 'config' is important as that's how it'll appear when mounted in arlon container
    tmp_config=/tmp/config
    cp testing/argocd-config-for-controller.template.yaml ${tmp_config}
    echo "  auth-token: ${auth_token}" >> ${tmp_config}
    echo creating argocd-creds secret
    kubectl -n arlon create secret generic argocd-creds --from-file ${tmp_config}
    rm -f ${tmp_config}
else
    echo argo-creds secret already exists
fi

# Deploy arlon controller
kubectl apply -f deploy/manifests/

if ! arlon bundle list|grep guestbook-static ; then
    echo creating guestbook-static bundle
    arlon bundle create guestbook-static --tags applications --desc "guestbook app" --from-file examples/bundles/guestbook.yaml
fi

if ! arlon bundle list|grep guestbook-dynamic ; then
    echo creating guestbook-dynamic bundle
    pushd ${workspace_repo}
    mkdir -p bundles/guestbook
    cp ${arlon_repo}/examples/bundles/guestbook.yaml bundles/guestbook
    git add bundles/guestbook
    git commit -m "add guestbook"
    git push origin main
    arlon bundle create guestbook-dynamic --tags applications --desc "guestbook app (dynamic)" --repo-url ${workspace_repo_url} --repo-path bundles/guestbook
    popd
fi

echo --- All done ---
