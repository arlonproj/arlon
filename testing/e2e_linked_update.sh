if [ -z "${GIT_USER}" ]; then
  echo "Set the GIT_USER env variable"
  exit
fi

if [ -z "${GIT_PASSWORD}" ]; then
  echo "Set the GIT_PASSWORD env variable"
  exit
fi
dir=$(pwd)
export git_server_port=8188
export GIT_CLONE_ROOT=/tmp/arlon-testbed-git-clone
export workspace_repo_url="http://localhost:${git_server_port}/${GIT_USER}/myrepo.git"
export workspace_repo="${GIT_CLONE_ROOT}/myrepo"
cd ${workspace_repo}
git pull
cd bundles/guestbook
sed -i -e "7s/replicas: 1/replicas: 3/g" guestbook.yaml
git add guestbook.yaml
git commit -m "added 3 replica line"
git push "http://${GIT_USER}:${GIT_PASSWORD}@localhost:${git_server_port}/${GIT_USER}/myrepo.git"
cd ${dir}

cp ~/.kube/temp.config ~/.kube/config
rm -rf /home/runner/work/arlon/arlon/kubeconfig





