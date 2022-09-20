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

if [[ "$OSTYPE" == "linux-gnu"* ]]; then
    os="linux"
    arlon_os="Linux"
elif [[ "$OSTYPE" == "darwin"* ]]; then
    os="darwin"   
    arlon_os="Darwin"
fi
cpu=$(uname -m)
if [[ "$cpu" == "x86_64" ]]; then
    arlon_arch="x86_64"
    arch="amd64"
elif [[ "$cpu" == "arm64"* ]]; then
    arlon_arch="arm64"
    arch="arm64"
fi
if [ ! -d "$HOME/.local/bin" ] ; then
  mkdir -p "$HOME/.local/bin"
fi
PATH=$PATH:$HOME/.local/bin
if ! which kubectl &> /dev/null ; then
    curl -sLO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/${os}/${arch}/kubectl"
    chmod +x kubectl
    mv kubectl ${HOME}/.local/bin/
fi

if [ -z "${KUBECONFIG}" ]; then
    echo "Set the KUBECONFIG variable to your management cluster's config"
    exit
fi


if [ -z "${AWS_REGION}" ]; then
    echo "Set the AWS_REGION env variable"
    exit
fi

if [ -z "${AWS_ACCESS_KEY_ID}" ]; then
    echo "Set the AWS_ACCESS_KEY_ID env variable"
    exit
fi

if [ -z "${AWS_SECRET_ACCESS_KEY}" ]; then
    echo "Set the AWS_SECRET_ACCESS_KEY env variable"
    exit
fi


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

if ! argocd &> /dev/null; then
    echo downloading argocd CLI
    curl -sSL -o ${HOME}/.local/bin/argocd https://github.com/argoproj/argo-cd/releases/latest/download/argocd-${os}-${arch}
    chmod +x ${HOME}/.local/bin/argocd
fi


kubectl patch svc argocd-server -n argocd -p '{"spec": {"type": "LoadBalancer"}}'

if pkill -f "kubectl port-forward svc/argocd-server" ; then
    echo terminated previous port forwarding session
fi

wait_until 'set -o pipefail; pwd=$(kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d)' 6 20
forwarding_port=8189
kubectl port-forward svc/argocd-server -n argocd ${forwarding_port}:443 &>/dev/null &


wait_until "argocd login localhost:${forwarding_port} --username admin --password ${pwd} --insecure" 10 20

if ! kubectl get ns arlon &> /dev/null ; then
    echo creating arlon namespace
    kubectl create ns arlon
fi



mkdir manifests
cd manifests
wget -qc https://raw.githubusercontent.com/arlonproj/arlon/main/testing/manifests/argocd-cm.yaml --output-document=argocd-cm.yaml 
wget -qc https://raw.githubusercontent.com/arlonproj/arlon/main/testing/manifests/argocd-rbac-cm.yaml --output-document=argocd-rbac-cm.yaml 
cd ..
kubectl apply -f manifests
rm -r manifests

if ! kubectl get secret argocd-creds -n arlon &> /dev/null ; then
    wait_until "auth_token=$(argocd account generate-token --account arlon)" 2 10
    echo auth_token: ${auth_token}
    # The file name 'config' is important as that's how it'll appear when mounted in arlon container
    tmp_config=/tmp/config
    wget -qc https://raw.githubusercontent.com/arlonproj/arlon/main/testing/argocd-config-for-controller.template.yaml --output-document=argocd-config-for-controller.template.yaml 
    mv argocd-config-for-controller.template.yaml ${tmp_config}
    echo "  auth-token: ${auth_token}" >> ${tmp_config}
    echo creating argocd-creds secret
    kubectl -n arlon create secret generic argocd-creds --from-file ${tmp_config}
    rm -f ${tmp_config}
else
    echo argo-creds secret already exists
fi

# Arlon CRDs
kubectl apply -f config/crd/bases

# Deploy arlon controller
kubectl apply -f deploy/manifests/

echo '------- waiting for arlon controller to become ready ---------'
wait_until 'set -o pipefail; kubectl get pods -n arlon | grep Running &> /dev/null' 10 30

echo Arlon controller is up and running

if ! which arlon &> /dev/null; then
    echo Downloading arlon CLI
    wget -qc https://github.com/arlonproj/arlon/releases/download/v0.9.9/arlon_${arlon_os}_${arlon_arch}_0.9.9.tar.gz
    tar -xf arlon_${arlon_os}_${arlon_arch}_0.9.9.tar.gz
    mv arlon_${os}_${arch}_v0.9.9 ${HOME}/.local/bin/arlon
    rm arlon_${arlon_os}_${arlon_arch}_0.9.9.tar.gz
fi


if ! which clusterctl &> /dev/null; then
    echo Downloading clusterctl CLI
    curl -L -o ${HOME}/.local/bin/clusterctl https://github.com/kubernetes-sigs/cluster-api/releases/download/v1.2.1/clusterctl-${os}-${arch} -o clusterctl
    chmod +x ${HOME}/.local/bin/clusterctl
fi

if ! which clusterawsadm &> /dev/null; then
    echo Downloading clusterawsadm CLI
    curl -L -o ${HOME}/.local/bin/clusterawsadm https://github.com/kubernetes-sigs/cluster-api-provider-aws/releases/download/v1.5.0/clusterawsadm-${os}-${arch} -o clusterawsadm
    chmod +x ${HOME}/.local/bin/clusterawsadm
fi

clusterawsadm bootstrap iam create-cloudformation-stack

export AWS_B64ENCODED_CREDENTIALS=$(clusterawsadm bootstrap credentials encode-as-profile)

clusterctl init --infrastructure aws
echo "To access ArgoCD UI, run: kubectl port-forward svc/argocd-server -n argocd ${forwarding_port}:443"
echo "Login as admin: ${pwd} into ArgoCD at http://localhost:${forwarding_port}"
echo "Run the following command to use kubectl, argocd, clusterctl, clusterawsadm, arlon (If not already installed)"
echo 'PATH=$PATH:$HOME/.local/bin'
echo Installation successfull