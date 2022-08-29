# Optional environment variables
#
#                         Default
# GIT_SERVER_PORT         8188
# GIT_ROOT                /tmp/arlon-testbed-git
# GIT_CLONE_ROOT          /tmp/arlon-testbed-git-clone
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

if ! clusterctl version > /dev/null; then
    echo clusterctl not installed
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

if [ -z "${GIT_ROOT}" ]; then
    GIT_ROOT=/tmp/arlon-testbed-git
fi
echo git root: ${GIT_ROOT}

if [ ! -d "${GIT_ROOT}" ]; then
    mkdir ${GIT_ROOT}
    chmod og+rwx ${GIT_ROOT}
fi

gitserver_cntr_name="arlon-testbed-gitserver"
if ! docker inspect ${gitserver_cntr_name} &> /dev/null ; then
    if ! docker run -d -v ${GIT_ROOT}:/var/lib/git -p ${git_server_port}:80 --name ${gitserver_cntr_name} --rm cirocosta/gitserver-http > /dev/null ; then
        echo failed to start git server container
        exit 5
    else
        echo started git server container
        sleep 2
    fi
else
    echo git server container already running
fi

git_repo_dir=${GIT_ROOT}/myrepo.git
if [ ! -d "${git_repo_dir}" ]; then
    echo initializing git repo
    mkdir ${git_repo_dir}
    pushd ${git_repo_dir}
    git init --bare
    sed -i s/master/main/ HEAD
    popd
fi
echo git repo at ${git_repo_dir}

if [ -z "${GIT_CLONE_ROOT}" ]; then
    GIT_CLONE_ROOT=/tmp/arlon-testbed-git-clone
fi
echo git clone root: ${GIT_CLONE_ROOT}

if [ ! -d "${GIT_CLONE_ROOT}" ]; then
    mkdir ${GIT_CLONE_ROOT}
fi

workspace_repo_url=http://${bridge_addr}:${git_server_port}/myrepo.git

workspace_repo=${GIT_CLONE_ROOT}/myrepo
if [ ! -d "${workspace_repo}" ]; then
    echo cloning git repo
    pushd ${GIT_CLONE_ROOT}
    git clone ${workspace_repo_url}
    cd myrepo
    echo hello > README.md
    git add README.md
    git commit -m README.md
    git push origin HEAD:main
    git checkout main
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

wait_until 'set -o pipefail; pwd=$(kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d)' 6 20

argocd_forwarding_port=${ARGGOCD_FORWARDING_PORT}
if [ -z "${argocd_forwarding_port}" ]; then
    argocd_forwarding_port=8189
fi
kubectl port-forward svc/argocd-server -n argocd ${argocd_forwarding_port}:443 &>/dev/null &

wait_until "argocd login localhost:${argocd_forwarding_port} --username admin --password ${pwd} --insecure" 6 20

# This is idempotent so no need to check whether repo already registered
wait_until "argocd repo add ${workspace_repo_url} --username dummy-user --password dummy-password" 2 30

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

if ! arlon bundle list|grep guestbook-static > /dev/null ; then
    echo creating guestbook-static bundle
    arlon bundle create guestbook-static --tags applications --desc "guestbook app" --from-file examples/bundles/guestbook.yaml
fi

if ! arlon bundle list|grep xenial-static > /dev/null ; then
    echo creating xenial-static bundle
    arlon bundle create xenial-static --tags applications --desc "xenial pod" --from-file examples/bundles/xenial.yaml
fi

if ! arlon bundle list|grep guestbook-dynamic > /dev/null ; then
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

if ! arlon bundle list|grep calico > /dev/null ; then
    echo creating calico bundle
    pushd ${workspace_repo}
    mkdir -p bundles/calico
    curl https://docs.projectcalico.org/v3.21/manifests/calico.yaml -o bundles/calico/calico.yaml
    git add bundles/calico
    git commit -m "add calico"
    git push origin main
    arlon bundle create calico --tags networking,cni --desc "Calico CNI" --repo-url ${workspace_repo_url} --repo-path bundles/calico
    popd
fi

if ! arlon profile list|grep static-1 > /dev/null ; then
    echo creating static-1 profile
    arlon profile create static-1 --static --bundles guestbook-static,xenial-static --desc "static profile 1" --tags examples
fi

if ! arlon profile list|grep dynamic-1 > /dev/null ; then
    echo creating dynamic-1 profile
    arlon profile create dynamic-1 --repo-url ${workspace_repo_url} --repo-base-path profiles --bundles guestbook-static,xenial-static --desc "dynamic test 1" --tags examples
fi

if ! arlon profile list|grep dynamic-2-calico > /dev/null ; then
    echo creating dynamic-2-calico profile
    arlon profile create dynamic-2-calico --repo-url ${workspace_repo_url} --repo-base-path profiles --bundles calico,guestbook-dynamic,xenial-static --desc "dynamic test 2" --tags examples
fi

if ! arlon clusterspec list|grep capi-kubeadm-3node > /dev/null ; then
    echo creating capi-kubeadm-3node clusterspec
    arlon clusterspec create capi-kubeadm-3node --api capi --cloud aws --type kubeadm --kubeversion v1.18.16 --nodecount 3 --nodetype t2.medium --tags devel,test --desc "3 node kubeadm for dev/test" --region us-west-2 --sshkey leb
fi

if ! arlon clusterspec list|grep capi-eks > /dev/null ; then
    echo creating capi-eks clusterspec
    arlon clusterspec create capi-eks --api capi --cloud aws --type eks --kubeversion v1.18.16 --nodecount 2 --nodetype t2.large --tags staging --desc "2 node eks for general purpose"  --region us-west-2 --sshkey leb
fi

if ! arlon clusterspec list|grep xplane-eks-3node > /dev/null ; then
    echo creating xplane-eks-3node clusterspec
    arlon clusterspec create xplane-eks-3node --api xplane --cloud aws --type eks --kubeversion v1.18.16 --nodecount 4 --nodetype t2.small --tags experimental --desc "4 node eks managed by crossplane"  --region us-west-2 --sshkey leb
fi

echo --- All done ---
