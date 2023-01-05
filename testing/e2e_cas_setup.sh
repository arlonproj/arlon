#!/usr/bin/env bash
set -e
set -x
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
  echo "Set the AWS_SSH_KEY_NAME env variable"
  exit
fi

# Setting os for kubectl download
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

if ! which jq; then
  sudo apt install jq -y
fi

if ! which helm; then
  curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
fi

if ! which kubectl &>/dev/null; then
  curl -sLO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/${os}/${arch}/kubectl"
  chmod +x kubectl
  mv kubectl "${HOME}/.local/bin/"
fi

if ! which kubectl-kuttl &>/dev/null; then
  echo Downloading kuttl plugin to run e2e tests
  curl -s -Lo "${HOME}/.local/bin/kubectl-kuttl" "https://github.com/kudobuilder/kuttl/releases/download/v0.14.0/kubectl-kuttl_0.14.0_${os}_${cpu}"
  chmod +x "${HOME}/.local/bin/kubectl-kuttl"
fi

if ! which eksctl &>/dev/null; then
  curl --silent --location "https://github.com/weaveworks/eksctl/releases/latest/download/eksctl_$(uname -s)_amd64.tar.gz" | tar xz -C /tmp
  sudo mv /tmp/eksctl "${HOME}/.local/bin/"
fi

if ! which aws &>/dev/null; then
  curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip"
  unzip awscliv2.zip
  sudo ./aws/install
fi

arlon_repo=$(pwd)

if [ ! -d ~/.aws ]; then
    mkdir -p ~/.aws
fi 

configFile=~/.aws/config
credentialsFile=~/.aws/credentials

cat <<EOF > ${configFile}
[default]
region = ${AWS_REGION}
output = json
EOF

cat <<EOF > ${credentialsFile}
[default]
aws_access_key_id = ${AWS_ACCESS_KEY_ID}
aws_secret_access_key = ${AWS_SECRET_ACCESS_KEY}
EOF

chmod 0644 ${configFile} ${credentialsFile}

export clusterNameFile=~/clustername
export dateSuffix=$(date "+%F-%H-%M-%S")
export clusterName=eksctl-${dateSuffix}

echo $clusterName > $clusterNameFile
eksConfig=${arlon_repo}/testing/eksctl_config.yaml
envsubst < $eksConfig > ${arlon_repo}/testing/eksctl.yaml
eksctl create cluster --config-file ${arlon_repo}/testing/eksctl.yaml

aws eks update-kubeconfig --region ${AWS_REGION} --name $clusterName

git_server_port=${GIT_SERVER_PORT}
if [ -z "${git_server_port}" ]; then
  git_server_port=3000
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
  helm install gitea gitea-charts/gitea --set gitea.admin.username="${GIT_USER}" --set gitea.admin.password="${GIT_PASSWORD}" --set service.http.type=LoadBalancer --wait --timeout 10m --wait-for-jobs
fi

git config --global user.email "${GIT_EMAIL}"
git config --global user.name "${GIT_USER}"

export service_ip=$(kubectl get svc gitea-http --template "{{ range (index .status.loadBalancer.ingress 0) }}{{.}}{{ end }}")

gittea_token=$(curl -v -H "Content-Type: application/json" -u "${GIT_USER}":"${GIT_PASSWORD}" -d '{"name": "tokens000778887_12344444009"}' --connect-timeout 10 --max-time 20 --retry 10 --retry-connrefused "http://${service_ip}:${git_server_port}/api/v1/users/${GIT_USER}/tokens" | jq -r '.sha1')
  curl -H "Content-Type: application/json" -H "Authorization: token ${gittea_token}" -d '{"auto_init": true, "default_branch": "main", "name": "myrepo"}' "http://${service_ip}:${git_server_port}/api/v1/user/repos"


if [ -z "${GIT_CLONE_ROOT}" ]; then
  GIT_CLONE_ROOT=/tmp/arlon-testbed-git-clone
fi
echo git clone root: ${GIT_CLONE_ROOT}

if [ ! -d "${GIT_CLONE_ROOT}" ]; then
  mkdir ${GIT_CLONE_ROOT}
fi

workspace_repo_url="http://${service_ip}:${git_server_port}/${GIT_USER}/myrepo.git"
workspace_repo="${GIT_CLONE_ROOT}/myrepo"

if [ ! -d "${workspace_repo}" ]; then
  echo cloning git repo
  pushd ${GIT_CLONE_ROOT}
  git clone ${workspace_repo_url}
  cd "${workspace_repo}"
  echo hello >>README.md
  git add README.md
  git commit -m README.md
  git push "http://${GIT_USER}:${GIT_PASSWORD}@${service_ip}:${git_server_port}/${GIT_USER}/myrepo.git"
  git checkout main
  popd
else
  echo git repo already cloned
fi

pushd ${workspace_repo}
if ! test -f README.md; then
  echo adding README.md and creating main branch
  echo hello >>README.md
  git add README.md
  git commit -m "add README.md"
  git push "http://${GIT_USER}:${GIT_PASSWORD}@${service_ip}:${git_server_port}/${GIT_USER}/myrepo.git"
else
  echo README.md already present
fi
popd

if ! which arlon &>/dev/null; then
  make build
  sudo ln -s -f "$(pwd)/bin/arlon" /usr/local/bin/arlon
fi

arlon init --username "${GIT_USER}" --password "${gittea_token}" --repoUrl "${workspace_repo_url}" -e -y

forwarding_port=8080
kubectl port-forward svc/argocd-server -n argocd ${forwarding_port}:443 &>/dev/null &

clusterctl generate cluster capi-quickstart --flavor eks \
  --kubernetes-version v1.23.10 \
  --control-plane-machine-count=1 \
  --worker-machine-count=1 \
  --infrastructure aws \
  >${arlon_repo}/testing/capi-quickstart-cas-e2e-test.yaml

repodir="${GIT_CLONE_ROOT}/myrepo"
repopath=basecluster/cas-cluster
baseclusterdir=${repodir}/${repopath}
manifestfile=capi-quickstart-cas-e2e-test.yaml

echo adding basecluster directory
mkdir -p ${baseclusterdir}
mv "${arlon_repo}/testing/${manifestfile}" "${baseclusterdir}/${manifestfile}"
pushd ${baseclusterdir}
git pull
git add ${manifestfile}
git commit -m ${manifestfile}
git push "http://${GIT_USER}:${GIT_PASSWORD}@${service_ip}:${git_server_port}/${GIT_USER}/myrepo.git"
popd
