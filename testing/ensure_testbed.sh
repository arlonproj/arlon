# Optional environment variables
#
#                     Default
# GIT_SERVER_PORT     8188
# GIT_ROOT            Create new directory under /tmp
# GIT_CLONE_ROOT      Create new directory under /tmp
# ARGOCD_GIT_TAG      release-2.4
# ARGOCD_CONFIG_FILE  Create new one under /tmp

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

git_url=http://${bridge_addr}:${git_server_port}/myrepo.git

git_clone_dir=${git_clone_root}/myrepo
if [ ! -d "${git_clone_dir}" ]; then
    echo cloning git repo
    pushd ${git_clone_root}
    git clone ${git_url}
    popd
else
    echo git repo already cloned
fi

pushd ${git_clone_dir}
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
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/${argocd_git_tag}/manifests/install.yaml

echo --- All done ---
