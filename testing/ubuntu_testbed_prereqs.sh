
sudo apt -y update

cd
mkdir -p downloads
pushd downloads

if ! which argocd; then
    wget https://github.com/argoproj/argo-cd/releases/latest/download/argocd-linux-amd64
    chmod +x argocd-linux-amd64
    sudo mv argocd-linux-amd64 /usr/local/bin/argocd
fi

if ! which kubectl; then
    curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
    chmod +x kubectl
    sudo mv kubectl /usr/local/bin/
fi

if ! which clusterctl; then
    curl -L https://github.com/kubernetes-sigs/cluster-api/releases/download/v1.2.2/clusterctl-linux-amd64 -o clusterctl
    chmod +x clusterctl
    sudo mv clusterctl /usr/local/bin/
fi

if ! which kind; then
   curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.17.0/kind-linux-amd64
   chmod +x kind
   sudo mv kind /usr/local/bin/
fi

sudo apt -y install docker.io
sudo usermod -aG docker ${USER}
echo "Log out and back in or run `newgrp docker` to ensure you can run docker command ..."
