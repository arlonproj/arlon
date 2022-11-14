#!/usr/bin/env bash
set -e
set -o pipefail

# $1 = expression
# $2 = sleep period
# $3 = iterations

function wait_until() {
  for _ in $(seq 1 "$3"); do
    eval "$1" && return 0
    echo "Waiting for \"$1\" to evaluate to true ..."
    sleep "$2"
  done
  echo Timed out waiting for \""$1"\"
  return 1
}

if [ -z "${GIT_USER}" ]; then
  echo "Set the GIT_USER env variable"
  exit
fi

if [ -z "${GIT_PASSWORD}" ]; then
  echo "Set the GIT_PASSWORD env variable"
  exit
fi

if [ -z "${GIT_EMAIL}" ]; then
  echo "Set the GIT_EMAIL env variable"
  exit
fi

# Check for variables required for creating CAPA clusters
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

if [ -z "${AWS_CONTROL_PLANE_MACHINE_TYPE}" ]; then
  echo "Set the AWS_CONTROL_PLANE_MACHINE_TYPE env variable"
  exit
fi

if [ -z "${AWS_NODE_MACHINE_TYPE}" ]; then
  echo "Set the AWS_NODE_MACHINE_TYPE env variable"
  exit
fi

if [ -z "${AWS_SSH_KEY_NAME}" ]; then
  echo "Set the AWS_NODE_MACHINE_TYPE env variable"
  exit
fi

# Setting os for arlon binary download
if [[ "$OSTYPE" == "linux-gnu"* ]]; then
  os="linux"
#  arlon_os="Linux"
elif [[ "$OSTYPE" == "darwin"* ]]; then
  os="darwin"
#  arlon_os="Darwin"
fi

cpu=$(uname -m)
if [[ "$cpu" == "x86_64" ]]; then
  arlon_arch="x86_64"
  arch="amd64"
elif [[ "$cpu" == "arm64"* ]]; then
  #  arlon_arch="arm64"
  arch="arm64"
fi
if [ ! -d "$HOME/.local/bin" ]; then
  mkdir -p "$HOME/.local/bin"
fi
PATH=$PATH:$HOME/.local/bin

# Check for Kind and docker
if ! which kind; then
  curl -Lo ./kind "https://kind.sigs.k8s.io/dl/v0.14.0/kind-${os}-${arch}"
  chmod +x ./kind
  mv ./kind /usr/local/bin/kind
fi

if ! which jq; then
  sudo apt install jq -y
fi

if ! which helm; then
  curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
fi

if ! which docker; then
  sudo apt -y install docker.io
  sudo usermod -aG docker "${USER}"
  echo "Log out and back in or run \`newgrp docker\` to ensure you can run docker command ..."
fi

if ! docker ps >/dev/null; then
  echo 'Docker not installed or operational (make sure your user can access /var/run/docker.sock - logout and back in if necessary)'
  exit 1
fi

if ! which kubectl &>/dev/null; then
  curl -sLO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/${os}/${arch}/kubectl"
  chmod +x kubectl
  mv kubectl "${HOME}/.local/bin/"
fi

# Creating a kind-based management cluster
tb_cntr_name='arlon-e2e-testbed'

if ! kind get clusters | grep ${tb_cntr_name}; then
  echo testbed container not found
  if ! kind create cluster --config testing/kind_config.yaml --name ${tb_cntr_name}; then
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

# Installing argocd in the management cluster
if ! kubectl get ns argocd &>/dev/null; then
  echo creating argocd namespace
  kubectl create ns argocd
fi

argocd_git_tag="${ARGOCD_GIT_TAG}"
if [ -z "${argocd_git_tag}" ]; then
  argocd_git_tag="release-2.4"
fi
echo applying argocd manifest from git tag: ${argocd_git_tag}
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/${argocd_git_tag}/manifests/install.yaml >/dev/null

if ! argocd &>/dev/null; then
  echo downloading argocd CLI
  curl -sSL -o "${HOME}/.local/bin/argocd" "https://github.com/argoproj/argo-cd/releases/latest/download/argocd-${os}-${arch}"
  chmod +x "${HOME}/.local/bin/argocd"
fi

kubectl patch svc argocd-server -n argocd -p '{"spec": {"type": "LoadBalancer"}}'

if pkill -f "kubectl port-forward svc/argocd-server"; then
  echo terminated previous port forwarding session
fi

wait_until 'set -o pipefail; pwd=$(kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d)' 6 20
forwarding_port=8189
kubectl port-forward svc/argocd-server -n argocd ${forwarding_port}:443 &>/dev/null &

wait_until "argocd login localhost:${forwarding_port} --username admin --password ${pwd} --insecure" 10 20

if ! kubectl get ns arlon &>/dev/null; then
  echo creating arlon namespace
  kubectl create ns arlon
fi

arlon_repo=$(pwd)

kubectl apply -f testing/manifests/

if ! kubectl get secret argocd-creds -n arlon &>/dev/null; then
  wait_until "auth_token=$(argocd account generate-token --account arlon)" 2 10
  echo auth_token: "${auth_token}"
  # The file name 'config' is important as that's how it'll appear when mounted in arlon container
  tmp_config=/tmp/config
  wget -qc https://raw.githubusercontent.com/arlonproj/arlon/main/testing/argocd-config-for-controller.template.yaml --output-document=argocd-config-for-controller.template.yaml
  mv argocd-config-for-controller.template.yaml ${tmp_config}
  echo "  auth-token: ${auth_token}" >>${tmp_config}
  echo creating argocd-creds secret
  kubectl -n arlon create secret generic argocd-creds --from-file ${tmp_config}
  rm -f ${tmp_config}
else
  echo argo-creds secret already exists
fi

if bridge_addr=$(ip addr | grep 'scope global docker0' | awk '{print $2}' | cut -d / -f 1); then
  echo docker bridge address is "$bridge_addr"
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

if ! kubectl get svc gitea-http; then
  helm repo add gitea-charts https://dl.gitea.io/charts/
  helm repo update
  helm install gitea gitea-charts/gitea --set gitea.admin.username="${GIT_USER}" --set gitea.admin.password="${GIT_PASSWORD}" --wait --wait-for-jobs --timeout 10m
  if pkill -f "kubectl --namespace default port-forward svc/gitea-http"; then
    echo "terminated previous port forwarding session for gitea"
  fi
  kubectl --namespace default port-forward svc/gitea-http ${git_server_port}:3000 --address 0.0.0.0 &>/dev/null &

  git config --global user.email "${GIT_EMAIL}"
  git config --global user.name "${GIT_USER}"

  gittea_token=$(curl -H "Content-Type: application/json" -u "${GIT_USER}":"${GIT_PASSWORD}" -d '{"name": "tokens"}' --connect-timeout 5 --max-time 10 --retry 5 --retry-connrefused "http://localhost:${git_server_port}/api/v1/users/${GIT_USER}/tokens" | jq -r '.sha1')
  curl -H "Content-Type: application/json" -H "Authorization: token ${gittea_token}" -d '{"auto_init": true, "default_branch": "main", "name": "myrepo"}' "http://localhost:${git_server_port}/api/v1/user/repos"
fi

if [ -z "${GIT_CLONE_ROOT}" ]; then
  GIT_CLONE_ROOT=/tmp/arlon-testbed-git-clone
fi
echo git clone root: ${GIT_CLONE_ROOT}

if [ ! -d "${GIT_CLONE_ROOT}" ]; then
  mkdir ${GIT_CLONE_ROOT}
fi

workspace_repo_url="http://localhost:${git_server_port}/${GIT_USER}/myrepo.git"
workspace_repo="${GIT_CLONE_ROOT}/myrepo"

git config --global user.email "${GIT_EMAIL}"
git config --global user.name "${GIT_USER}"

if [ ! -d "${workspace_repo}" ]; then
  echo cloning git repo
  pushd ${GIT_CLONE_ROOT}
  git clone ${workspace_repo_url}
  cd "${workspace_repo}"
  echo hello >>README.md
  git add README.md
  git commit -m README.md
  git push "http://${GIT_USER}:${GIT_PASSWORD}@localhost:${git_server_port}/${GIT_USER}/myrepo.git"
  git checkout main
  popd
else
  echo git repo already cloned
fi
echo printing the current directory
ls
pushd ${workspace_repo}
if ! test -f README.md; then
  echo adding README.md and creating main branch
  echo hello >>README.md
  git add README.md
  git commit -m "add README.md"
  git push "http://${GIT_USER}:${GIT_PASSWORD}@localhost:${git_server_port}/${GIT_USER}/myrepo.git"
else
  echo README.md already present
fi
popd

# Arlon CRDs
kubectl apply -f config/crd/bases

# Deploy arlon controller
kubectl apply -f deploy/manifests/

echo '------- waiting for arlon controller to become ready ---------'
wait_until 'set -o pipefail; kubectl get pods -n arlon | grep Running &> /dev/null' 10 30

echo Arlon controller is up and running

if ! which arlon &>/dev/null; then
  make build
  sudo ln -s "$(pwd)/bin/arlon" /usr/local/bin/arlon
fi

echo "${workspace_repo_url}"
global_workspace_repo_url="http://${bridge_addr}:${git_server_port}/${GIT_USER}/myrepo.git"
wait_until "arlon git register ${global_workspace_repo_url} --user ${GIT_USER} --password ${GIT_PASSWORD} --alias default" 2 30

if ! which clusterctl &>/dev/null; then
  echo Downloading clusterctl CLI
  curl -L -o "${HOME}/.local/bin/clusterctl" https://github.com/kubernetes-sigs/cluster-api/releases/download/v1.1.6/clusterctl-${os}-${arch} -o clusterctl
  chmod +x "${HOME}/.local/bin/clusterctl"
fi

if ! which clusterawsadm &>/dev/null; then
  echo Downloading clusterawsadm CLI
  curl -L -o "${HOME}/.local/bin/clusterawsadm" https://github.com/kubernetes-sigs/cluster-api-provider-aws/releases/download/v1.5.0/clusterawsadm-${os}-${arch} -o clusterawsadm
  chmod +x "${HOME}/.local/bin/clusterawsadm"
fi

if ! which kubectl-kuttl &>/dev/null; then
  echo Downloading kuttl plugin to run e2e tests
  curl -s -LO "https://github.com/kudobuilder/kuttl/releases/download/v0.13.0/kubectl-kuttl_0.13.0_${os}_${cpu}"
  mv "kubectl-kuttl_0.13.0_${os}_${cpu}" "kubectl-kuttl"
  chmod +x kubectl-kuttl
  mv kubectl-kuttl "${HOME}/.local/bin/kubectl-kuttl"
fi

# not needed for us...	
#clusterawsadm bootstrap iam create-cloudformation-stack

export AWS_B64ENCODED_CREDENTIALS=$(clusterawsadm bootstrap credentials encode-as-profile)

clusterctl init --infrastructure aws --wait-providers
echo "To access ArgoCD UI, run: kubectl port-forward svc/argocd-server -n argocd ${forwarding_port}:443"
echo "Login as admin: ${pwd} into ArgoCD at http://localhost:${forwarding_port}"
echo "Run the following command to use kubectl, argocd, clusterctl, clusterawsadm, arlon (If not already installed)"
echo 'PATH=$PATH:$HOME/.local/bin'
echo Installation successfull

clusterctl generate cluster capi-quickstart --flavor eks \
  --kubernetes-version v1.23.10 \
  --control-plane-machine-count=3 \
  --worker-machine-count=2 \
  --infrastructure aws \
  >${arlon_repo}/testing/capi-quickstart-e2e-test.yaml

repodir="${GIT_CLONE_ROOT}/myrepo"

repopath=basecluster/ec2-cluster
baseclusterdir=${repodir}/${repopath}
manifestfile=capi-quickstart-e2e-test.yaml
# Add bundle manifests to the workspace repo url
if ! arlon bundle list | grep xenial >/dev/null; then
  echo "Adding xenial manifests"
  pushd ${workspace_repo}
  mkdir -p bundles/xenial
  cp "${arlon_repo}/examples/bundles/xenial.yaml" bundles/xenial
  git add bundles/xenial
  git commit -m "add xenial bundle"
  git push "http://${GIT_USER}:${GIT_PASSWORD}@localhost:${git_server_port}/${GIT_USER}/myrepo.git"
  popd
fi

if ! arlon bundle list | grep guestbook >/dev/null; then
  echo "Adding guestbook manifests"
  pushd ${workspace_repo}
  mkdir -p bundles/guestbook
  cp "${arlon_repo}/examples/bundles/guestbook.yaml" bundles/guestbook
  git add bundles/guestbook
  git commit -m "add guestbook bundle"
  git push "http://${GIT_USER}:${GIT_PASSWORD}@localhost:${git_server_port}/${GIT_USER}/myrepo.git"
  popd
fi

echo adding basecluster directory
mkdir -p ${baseclusterdir}
mv "${arlon_repo}/testing/${manifestfile}" "${baseclusterdir}/${manifestfile}"
pushd ${baseclusterdir}
git pull
git add ${manifestfile}
git commit -m ${manifestfile}
git push "http://${GIT_USER}:${GIT_PASSWORD}@localhost:${git_server_port}/${GIT_USER}/myrepo.git"
popd
