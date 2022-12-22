set -e
set -o pipefail

mkdir -p ~/downloads
pushd ~/downloads

# Uninstall go package just in case. Idempotent
sudo apt -y update
sudo apt -y purge golang-go

# 22.04 does not come out of the box with gcc, required by go build
sudo apt -y install gcc

wget https://go.dev/dl/go1.19.4.linux-amd64.tar.gz
tar xf go1.19.4.linux-amd64.tar.gz

sudo ln -s /home/ubuntu/downloads/go/bin/go /usr/local/bin/go
sudo ln -s /home/ubuntu/downloads/go/bin/gofmt /usr/local/bin/gofmt

popd


go build
sudo ln -s `pwd`/arlon /usr/local/bin/arlon


