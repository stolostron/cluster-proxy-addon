all: build
.PHONY: all

export GOPATH ?= $(shell go env GOPATH)

# Include the library makefile
include $(addprefix ./vendor/github.com/openshift/build-machinery-go/make/, \
	golang.mk \
	targets/openshift/deps.mk \
	targets/openshift/images.mk \
	targets/openshift/bindata.mk \
	lib/tmp.mk \
)

# Image URL to use all building/pushing image targets;
IMAGE ?= cluster-proxy-addon
IMAGE_REGISTRY ?= quay.io/open-cluster-management

# ANP source code
ANP_NAME ?= apiserver-network-proxy
ANP_VERSION ?= 0.0.24
ANP_SRC_CODE ?= dependencymagnet/${ANP_NAME}/${ANP_VERSION}.tar.gz

# Add packages to do unit test
GO_TEST_PACKAGES :=./pkg/...

# This will call a macro called "build-image" which will generate image specific targets based on the parameters:
# $0 - macro name
# $1 - target suffix
# $2 - Dockerfile path
# $3 - context directory for image build
# It will generate target "image-$(1)" for building the image and binding it as a prerequisite to target "images".
$(call build-image,$(IMAGE),$(IMAGE_REGISTRY)/$(IMAGE),./Dockerfile,.)

$(call add-bindata,addon-agent,./pkg/hub/addon/manifests/...,bindata,bindata,./pkg/hub/addon/bindata/bindata.go)

build-all: build build-anp
.PHONY: build-all

build-anp:
	mkdir -p $(PERMANENT_TMP)
	cp $(ANP_SRC_CODE) $(PERMANENT_TMP)/$(ANP_NAME).tar.gz
	cd $(PERMANENT_TMP) && tar -xf $(ANP_NAME).tar.gz
	cp -r vendor $(PERMANENT_TMP)/$(ANP_NAME)
	cd $(PERMANENT_TMP)/$(ANP_NAME) && mv modules.txt.bak vendor/modules.txt
	cd $(PERMANENT_TMP)/$(ANP_NAME) && rm -rf vendor/sigs.k8s.io/apiserver-network-proxy
	cd $(PERMANENT_TMP)/$(ANP_NAME) && go build -o proxy-agent cmd/agent/main.go
	cd $(PERMANENT_TMP)/$(ANP_NAME) && go build -o proxy-server cmd/server/main.go
	mv $(PERMANENT_TMP)/$(ANP_NAME)/proxy-agent ./
	mv $(PERMANENT_TMP)/$(ANP_NAME)/proxy-server ./
.PHONY: build-anp

# TODO include ./test/integration-test.mk
build-e2e: 
	go test -c ./test/e2e

deploy-ocm:
	install-ocm.sh

HELM?=_output/linux-amd64/helm
KUBECTL?=kubectl

IMAGE=quay.io/open-cluster-management/cluster-proxy-addon:latest
IMAGE_PULL_POLICY=Always
CLUSTER_BASE_DOMAIN=

ensure-helm:
	mkdir -p _output
	cd _output && curl -s -f -L https://get.helm.sh/helm-v3.2.4-linux-amd64.tar.gz -o helm-v3.2.4-linux-amd64.tar.gz
	cd _output && tar -xvzf helm-v3.2.4-linux-amd64.tar.gz
.PHONY: setup

lint: ensure-helm
	$(HELM) lint stable/cluster-proxy-addon
.PHONY: lint

deploy-cluster-proxy: ensure-helm
	$(KUBECTL) get ns open-cluster-management ; if [ $$? -ne 0 ] ; then $(KUBECTL) create ns open-cluster-management ; fi
	$(HELM) install -n open-cluster-management cluster-proxy-addon test/deploy/cluster-proxy-addon \
	--set global.pullPolicy="$(IMAGE_PULL_POLICY)" \
	--set global.imageOverrides.cluster_proxy_addon="$(IMAGE)" \
	--set cluster_basedomain="$(CLUSTER_BASE_DOMAIN)" 
.PHONY: deploy-cluster-proxy

test-e2e: build-e2e deploy-ocm deploy-cluster-proxy
	./e2e.test -test.v -ginkgo.v

clean-e2e:
# TODO: delete all stuff in open-cluster-management and open-cluster-management-agent-addon