
# Image URL to use all building/pushing image targets
IMG ?= macea/controller:latest
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true,preserveUnknownFields=false"
AUTOGENMSG="# This is an auto-generated file. DO NOT EDIT"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

KUBECTL = $(shell which kubectl)
ARGOCD = $(shell which argocd)
MINIKUBE = $(shell which minikube)
KUBENS = $(shell which kubens)
DOCKER = $(shell which docker)

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

manifests: kustomize controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) $(CRD_OPTIONS) paths="./..." output:crd:artifacts:config=manifests/crd/bases
	cd manifests/deployment && $(KUSTOMIZE) edit set image controller=${IMG}
	touch manifests/install.yaml
	echo "# This is an auto-generated file. DO NOT EDIT" > manifests/install.yaml
	${KUSTOMIZE} build manifests/namespace-install/. >> manifests/install.yaml
	chmod 644 manifests/install.yaml

generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

fmt: ## Run go fmt against code.
	go fmt ./...

vet: ## Run go vet against code.
	go vet ./...

##@ Build

build: generate fmt vet ## Build manager binary.
	go build -o bin/manager main.go

run: manifests generate fmt vet ## Run a controller from your host.
	go run ./main.go

docker-build: ## Build docker image with the manager.
	docker build -t ${IMG} .

docker-push: ## Push docker image with the manager.
	docker push ${IMG}

delete-deployment:
	-$(KUBENS) argocd
	$(KUBECTL) delete deployment argocd-appsource-controller

logs:
	-$(KUBENS) argocd
	$(KUBECTL) logs --follow deploy/argocd-appsource-controller

token-secret:
	-$(KUBENS) argocd
	-$(KUBECTL) delete secret argocd-appsource-secret
	TOKEN=$($(ARGOCD) account generate-token --account appsource)
	$(KUBECTL) create secret generic argocd-appsource-secret --from-literal argocd-token=$(TOKEN)

deployment: manifests
	-$(KUBECTL) apply -f manifests/install.yaml

sample-1:
	-$(KUBECTL) create namespace my-project-us-west-2
	-$(KUBENS) my-project-us-west-2
	$(KUBECTL) apply -f manifests/samples/sample_appsource_instance_1.yaml

sample-2:
	-$(KUBECTL) create namespace my-project-us-east-2
	-$(KUBENS) my-project-us-east-2
	$(KUBECTL) apply -f manifests/samples/sample_appsource_instance_2.yaml

samples: sample-1 sample-2

delete-sample1:
	-$(KUBECTL) delete appsource sample1 -n my-project-us-west-2

delete-sample2:
	-$(KUBECTL) delete appsource sample2 -n my-project-us-east-2

delete-samples: delete-sample1 delete-sample2

clean-test: delete-samples delete-deployment

image:
	$(DOCKER) build --progress=plain -t $(IMG) .
	$(DOCKER) push $(IMG)

CONTROLLER_GEN = $(shell pwd)/bin/controller-gen
controller-gen: ## Download controller-gen locally if necessary.
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.4.1)

KUSTOMIZE = $(shell which kustomize)
kustomize: ## Download kustomize locally if necessary.
	$(call go-get-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v3@v3.8.7)

# go-get-tool will 'go get' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go get $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef
