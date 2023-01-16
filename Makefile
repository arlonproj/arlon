REPO_ROOT := $(shell pwd)
VERSION ?= $(shell cat $(REPO_ROOT)$*/version;)
GIT_SHORT_COMMIT := $(shell cd $(REPO_ROOT); git rev-parse --short HEAD)
REPO_SERVER ?= ghcr.io
REPO_ORG ?= arlonproj
REPO_NAME ?= arlon
CAPI_VERSION := $(shell cat $(REPO_ROOT)$*/capirc)
ARGO_VERSION := $(shell cat $(REPO_ROOT)$*/argorc)
ARLON_CLI_VERSION := $(shell cat $(REPO_ROOT)$*/version)
CAPI_LD_FLAG := -X github.com/arlonproj/arlon/cmd/install.capiCoreProvider=$(CAPI_VERSION)
ARGO_LD_FLAG := -X github.com/arlonproj/arlon/cmd/initialize.argocdGitTag=$(ARGO_VERSION)
ARLON_CLI_LD_FLAG := -X github.com/arlonproj/arlon/cmd/version.cliVersion=$(ARLON_CLI_VERSION)
LD_FLAGS := $(CAPI_LD_FLAG) $(ARGO_LD_FLAG) $(ARLON_CLI_LD_FLAG) -s -w
# Image URL to use all building/pushing image targets
IMG ?= $(REPO_SERVER)/$(REPO_ORG)/$(REPO_NAME)/controller:$(VERSION)
# Produce CRDs with multiversion enabled for v1 APIs - fixes failure in make test
# See https://book.kubebuilder.io/reference/generating-crd.html#multiple-versions 
CRD_OPTIONS ?= "crd"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen

## Tool Versions
CONTROLLER_TOOLS_VERSION ?= v0.10.0

# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

fmt: ## Run go fmt against code.
	go fmt ./...

vet: ## Run go vet against code.
	go vet ./...

ENVTEST_ASSETS_DIR=$(shell pwd)/testbin
test: manifests generate fmt vet ## Run tests.
	mkdir -p ${ENVTEST_ASSETS_DIR}
	test -f ${ENVTEST_ASSETS_DIR}/setup-envtest.sh || curl -sSLo ${ENVTEST_ASSETS_DIR}/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.8.3/hack/setup-envtest.sh
	source ${ENVTEST_ASSETS_DIR}/setup-envtest.sh; export GOARCH=amd64; fetch_envtest_tools $(ENVTEST_ASSETS_DIR); setup_envtest_env $(ENVTEST_ASSETS_DIR); go test ./...  -coverprofile cover.out -v -race -covermode=atomic

##@ Build
clean:
	rm -rf ./testbin; rm -rf ./bin

build: generate fmt vet cluster-config ## Build manager binary.
	go build -ldflags '$(LD_FLAGS)' -o bin/arlon main.go

cluster-config:
	mkdir -p bin
	tar cvfz bin/setup_arlon.tar.gz ./setup_arlon.sh ./config/crd/bases/*.yaml ./deploy/manifests/*.yaml ./testing/manifests/*.yaml

# goreleaser can invoke this target to produce binaries for different OS and CPU arch combinations
build-cli: fmt vet ## Build CLI binary (with the current OS and CPU architecture) from the go env.
	go build -o bin/arlon -ldflags '$(LD_FLAGS)' main.go

build-cli-linux: fmt vet ## Build CLI binary for Linux
	GOOS=linux GOARCH=amd64 go build -o bin/arlon -ldflags '$(LD_FLAGS)' main.go

build-cli-mac-amd64: fmt vet ## Build CLI binary for Mac (AMD/ Intel CPU)
	GOOS=darwin GOARCH=amd64 go build -o bin/arlon -ldflags '$(LD_FLAGS)' main.go

build-cli-mac-arm64: fmt vet ## Build CLI binary for Mac (Apple Silicon)
	GOOS=darwin GOARCH=arm64 go build -o bin/arlon -ldflags '$(LD_FLAGS)' main.go

ifeq (GOARCH,"arm64")
build-cli-mac: build-cli-mac-arm64
else
build-cli-mac: build-cli-mac-amd64
endif

# Arlon has not been tested on Windows yet.
build-cli-win: fmt vet ## Build CLI binary for Windows.
	GOOS=windows GOARCH=amd64 go build -o bin/arlon -ldflags '$(LD_FLAGS)' main.go

run: manifests generate fmt vet ## Run a controller from your host.
	go run ./main.go

docker-build: test ## Build docker image with the manager.
	docker build --label 'origin=$(REPO_ORG)/$(REPO_NAME)@$(GIT_SHORT_COMMIT)' -t ${IMG} .

docker-build-notest:
	docker build --label 'origin=$(REPO_ORG)/$(REPO_NAME)@$(GIT_SHORT_COMMIT)' -t ${IMG} .

docker-push: ## Push docker image with the manager.
	docker push ${IMG}

##@ Deployment

install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | kubectl apply -f -

undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/default | kubectl delete -f -

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	test -s $(LOCALBIN)/controller-gen || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

KUSTOMIZE = $(shell pwd)/bin/kustomize
kustomize: ## Download kustomize locally if necessary.
	$(call go-get-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v4@v4.5.7)

# go-get-tool will 'go get' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go install $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef

pkgtest:
	go test -v ./pkg/...

test-e2e:
	./testing/e2e_setup.sh
	kubectl kuttl test --start-kind=false ./testing/e2e/ --test 00-deploy
	kubectl kuttl test --start-kind=false ./testing/e2e/ --test 01-update
	kubectl kuttl test --start-kind=false ./testing/e2e/ --test 02-linkedupdate
	kubectl kuttl test --start-kind=false ./testing/e2e/ --test 03-linkedbundleupdate
	kubectl kuttl test --start-kind=false ./testing/e2e/ --test 04-linkedprofileupdate
	kubectl kuttl test --start-kind=false ./testing/e2e/ --test 05-delete
	kubectl kuttl test --start-kind=false ./testing/e2e/ --test 06-manage --kind-context arlon-e2e-testbed --timeout 300

test-e2e-appprofiles:
	./testing/e2e_setup.sh
	kubectl kuttl test --start-kind=false ./testing/e2e-appprofiles/ --test 00-deploy

test-e2e-cas:
	./testing/e2e_cas_setup.sh
	kubectl kuttl test --start-kind=false ./testing/e2e-cas/ --test 00-cas

e2e-teardown:
	./testing/e2e_setup_teardown.sh

e2e-teardown-cas:
	./testing/e2e_cas_teardown.sh
