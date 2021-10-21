VERSION ?= v0.2.0
NUM ?= 0
OVNOVSVERSION ?= 2.15.1_ubuntu_20.12.1
COMMIT := $(shell git rev-parse HEAD)
BRANCH := $(shell git branch --contains ${COMMIT})
# BRANCH := $(shell git branch --show-current)
# $(shell echo ${VERSION}-${COMMIT}-${BRANCH} > VERSION)

GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

TESTREPO=myregistry.com/kuryr
DEVREPO=192.168.169.2:5000/spider

ifdef DEV
    DISTREPO=${DEVREPO}
else
    DISTREPO=${TESTREPO}
endif

# ifneq ($(which nerdctl|wc -l), 0)
# 	BUILDCMD=nerdctl --insecure-registry build
# 	BUILDCLI=nerdctl --insecure-registry
# else
    BUILDCMD=docker build --network=host
    BUILDCLI=docker
# endif
    DOCKER_BUILDKIT=$(env DOCKER_BUILDKIT)

ifdef DEBUG
    LDFLAGS=
else
    LDFLAGS=-ldflags='-s -w'
endif

default: build

.PHONY: clean
clean:
	rm -rf dist/*
	rm -rf debug/*

distdir:
	mkdir -p dist/${GOARCH}

debugdir:
	mkdir -p debug/${GOARCH}

all: distdir init crd cni wooshnet 
alldebug: debugdir init crd cni wooshnet 

build: distdir buildcni buildwooshnet 
image: imgcrd imgwooshnet 

init: imginit
crd: imgcrd
cni: buildcni
wooshnet: buildwooshnet imgwooshnet crd

tools:
	CGO_ENABLED=0 GOOS=${GOOS} GOARCH=${GOARCH} go build -v ${LDFLAGS} -o ./tools/envtotext ./cmd/build/

buildwooshnet: buildcni
	VERSION=${VERSION} ./tools/envtotext -o ./pkg/version/VERSION -v
	CGO_ENABLED=0 GOOS=${GOOS} GOARCH=${GOARCH} go build -v ${LDFLAGS} -o ./dist/${GOARCH}/wooshnet .

buildcni:
	CGO_ENABLED=0 GOOS=${GOOS} GOARCH=${GOARCH} go build -v ${LDFLAGS} -o ./dist/${GOARCH}/wooshcni ./cmd/wooshcni/
	zstd -f ./dist/${GOARCH}/wooshcni

buildcrd:
	make install
	cd ../

imgcrd:
	${BUILDCMD} --build-arg GOARCH=${GOARCH} --build-arg REPO=${DISTREPO} -t $(DISTREPO)/networkcrd:${GOARCH} -f docker/Dockerfile.crd .
	${BUILDCLI} tag $(DISTREPO)/networkcrd:${GOARCH} $(DISTREPO)/networkcrd:$(VERSION)-${GOARCH}
	${BUILDCLI} push $(DISTREPO)/networkcrd:${GOARCH}
	${BUILDCLI} push $(DISTREPO)/networkcrd:$(VERSION)-${GOARCH}

imginit:
	${BUILDCMD} --build-arg GOARCH=${GOARCH} --build-arg REPO=${DISTREPO} -t $(DISTREPO)/networkinit:${GOARCH} -f docker/Dockerfile.init .
	${BUILDCLI} tag $(DISTREPO)/networkinit:${GOARCH} $(DISTREPO)/networkinit:$(VERSION)-${GOARCH}
	${BUILDCLI} push $(DISTREPO)/networkinit:${GOARCH}
	${BUILDCLI} push $(DISTREPO)/networkinit:$(VERSION)-${GOARCH}

imgwooshnet: 
	${BUILDCMD} --build-arg GOARCH=${GOARCH} --build-arg REPO=${DISTREPO} -t $(DISTREPO)/wooshnet:${GOARCH} -f ./docker/Dockerfile.wooshnet .
	${BUILDCLI} tag $(DISTREPO)/wooshnet:${GOARCH} $(DISTREPO)/wooshnet:$(VERSION)-${GOARCH}
	${BUILDCLI} push $(DISTREPO)/wooshnet:${GOARCH}
	${BUILDCLI} push $(DISTREPO)/wooshnet:$(VERSION)-${GOARCH}
	rm ./dist/${GOARCH}/wooshnet

imgovn:
	${BUILDCMD} --build-arg GOARCH=${GOARCH} --build-arg REPO=${DISTREPO} -t $(DISTREPO)/ovn-ovs:${GOARCH} -f ./docker/${GOARCH}/Dockerfile.ovn-ovs .
	${BUILDCLI} tag $(DISTREPO)/ovn-ovs:${GOARCH} $(DISTREPO)/ovn-ovs:$(OVNOVSVERSION)-${GOARCH}
	${BUILDCLI} push $(DISTREPO)/ovn-ovs:${GOARCH}
	${BUILDCLI} push $(DISTREPO)/ovn-ovs:$(OVNOVSVERSION)-${GOARCH}


# Image URL to use all building/pushing image targets
IMG ?= controller:latest
# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.22

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

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
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

fmt: ## Run go fmt against code.
	go fmt ./...

vet: ## Run go vet against code.
	go vet ./...

test: manifests generate fmt vet envtest ## Run tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" go test ./... -coverprofile cover.out

##@ Build

run: manifests generate fmt vet ## Run a controller from your host.
	go run ./main.go

##@ Deployment

ifndef ignore-not-found
  ignore-not-found = no
endif

install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/crd | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | kubectl apply -f -

undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/default | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

CONTROLLER_GEN = $(GOBIN)/controller-gen
controller-gen: ## Download controller-gen locally if necessary.
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.7.0)

KUSTOMIZE = $(GOBIN)/kustomize
kustomize: ## Download kustomize locally if necessary.
	$(call go-get-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v3@v3.10.0)

ENVTEST = $(GOBIN)/setup-envtest
envtest: ## Download envtest-setup locally if necessary.
	$(call go-get-tool,$(ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest@latest)

# go-get-tool will 'go get' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(GOBIN) go get $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef
