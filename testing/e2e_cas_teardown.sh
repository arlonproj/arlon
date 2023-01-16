#!/usr/bin/env bash
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

git_server_port=3000

if which arlon &>/dev/null; then
  arlon cluster delete cas-e2e-cluster
  wait_until "set -o pipefail; arlon cluster list 2> /dev/null | grep -v cas-e2e-cluster" 60 20
else
  echo "arlon not installed on PATH"
  echo "cluster probably not created. Skipping cluster delete..."
fi

rm -rf ~/.arlon ~/.config/arlon ~/.aws /tmp/arlon*
workspace_repo="/tmp/arlon-testbed-git-clone/myrepo"

helm uninstall gitea
kubectl delete deploy -n kube-system coredns ebs-csi-controller

clusterName=`cat ~/.clustername`
eksctl delete cluster --name $clusterName --region ${AWS_REGION} --disable-nodegroup-eviction
