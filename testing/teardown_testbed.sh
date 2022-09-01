
kind delete cluster --name kind-arlon-testbed || true
docker stop arlon-testbed-gitserver || true
sudo rm -rf /tmp/arlon-testbed-git*
