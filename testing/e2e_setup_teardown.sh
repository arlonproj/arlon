kind delete cluster --name arlon-e2e-testbed || true
docker stop arlon-e2e-testbed-gitserver || true
sudo rm -rf /tmp/arlon-testbed-git*