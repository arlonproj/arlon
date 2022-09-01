set -e
set -o pipefail

cd
mkdir -p ~/downloads
pushd downloads

# Uninstall go package just in case. Idempotent
sudo apt -y update
sudo apt -y purge golang-go

wget https://go.dev/dl/go1.19.linux-amd64.tar.gz
tar xf go1.19.linux-amd64.tar.gz

sudo ln -s /home/ubuntu/downloads/go/bin/go /usr/local/bin/go
sudo ln -s /home/ubuntu/downloads/go/bin/gofmt /usr/local/bin/gofmt

go build
sudo ln -s ~/devel/arlon/arlon /usr/local/bin/arlon


