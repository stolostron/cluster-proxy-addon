all: build
.PHONY: all

HELM?=_output/linux-amd64/helm

IMAGE_CLUSTER_PROXY?=quay.io/stolostron/cluster-proxy:backplane-2.7
IMAGE_PULL_POLICY=Always
IMAGE_TAG?=latest

# Using the following command to get the base domain of a OCP cluster
# export CLUSTER_BASE_DOMAIN=$(kubectl get ingress.config.openshift.io cluster -o=jsonpath='{.spec.domain}')
CLUSTER_BASE_DOMAIN?=

export CGO_ENABLED = 1

export GOPATH ?= $(shell go env GOPATH)

export DOCKER_BUILDER ?= docker

# Image URL to use all building/pushing image targets;
IMAGE ?= cluster-proxy-addon
IMAGE_REGISTRY ?= quay.io/stolostron
IMAGE_TAG ?= latest

# ANP source code
ANP_NAME ?= apiserver-network-proxy
ANP_VERSION ?= 0.1.6.patch-03
ANP_SRC_CODE ?= dependencymagnet/${ANP_NAME}/${ANP_VERSION}.tar.gz
PERMANENT_TMP ?= _output

# Add packages to do unit test
GO_TEST_PACKAGES :=./pkg/...
KUBECTL ?= kubectl

CLUSTER_PROXY_ADDON_IMAGE?=${IMAGE_REGISTRY}/${IMAGE}:${IMAGE_TAG}

build-all: build build-anp
.PHONY: build-all

build:
	go build -o cluster-proxy ./cmd/cluster-proxy/main.go
.PHONY: build

build-anp:
	mkdir -p $(PERMANENT_TMP)
	cp $(ANP_SRC_CODE) $(PERMANENT_TMP)/$(ANP_NAME).tar.gz
	cd $(PERMANENT_TMP) && tar -xf $(ANP_NAME).tar.gz
	cd $(PERMANENT_TMP)/$(ANP_NAME) && go build -o proxy-agent cmd/agent/main.go
	cd $(PERMANENT_TMP)/$(ANP_NAME) && go build -o proxy-server cmd/server/main.go
	mv $(PERMANENT_TMP)/$(ANP_NAME)/proxy-agent ./
	mv $(PERMANENT_TMP)/$(ANP_NAME)/proxy-server ./
.PHONY: build-anp

# e2e
build-e2e:
	go test -c ./test/e2e
.PHONY: build-e2e

deploy-ocm:
	curl -L https://raw.githubusercontent.com/open-cluster-management-io/clusteradm/main/install.sh | INSTALL_DIR=$(PWD) bash
	$(PWD)/clusteradm init --output-join-command-file join.sh --wait
	echo " loopback --force-internal-endpoint-lookup" >> join.sh && sh join.sh
	$(PWD)/clusteradm accept --clusters loopback --wait 30
	$(KUBECTL) wait --for=condition=ManagedClusterConditionAvailable managedcluster/loopback
.PHONY: deploy-ocm

ensure-helm:
	mkdir -p _output
	cd _output && curl -s -f -L https://get.helm.sh/helm-v3.2.4-linux-amd64.tar.gz -o helm-v3.2.4-linux-amd64.tar.gz
	cd _output && tar -xvzf helm-v3.2.4-linux-amd64.tar.gz
.PHONY: ensure-helm

# CLUSTER_PROXY_ADDON_IMAGE is passed in by prow, represents the image of cluster-proxy-addon built with the current snapshot.
deploy-addon-for-e2e: ensure-helm
	$(KUBECTL) apply -f chart/cluster-proxy-addon/crds/managedproxyconfigurations.yaml
	$(KUBECTL) apply -f chart/cluster-proxy-addon/crds/managedproxyserviceresolvers.yaml
	$(HELM) install \
	-n open-cluster-management-addon --create-namespace \
	cluster-proxy-addon chart/cluster-proxy-addon \
	--set global.pullPolicy="$(IMAGE_PULL_POLICY)" \
	--set global.imageOverrides.cluster_proxy_addon="$(CLUSTER_PROXY_ADDON_IMAGE)" \
	--set global.imageOverrides.cluster_proxy="$(IMAGE_CLUSTER_PROXY)" \
	--set cluster_basedomain="$(shell $(KUBECTL) get ingress.config.openshift.io cluster -o=jsonpath='{.spec.domain}')"
	$(KUBECTL) apply -f test/e2e/placement/ns.yaml
	$(KUBECTL) apply -f test/e2e/placement/
.PHONY: deploy-addon-for-e2e

test-e2e: deploy-ocm deploy-addon-for-e2e build-e2e
	export CLUSTER_BASE_DOMAIN=$(shell $(KUBECTL) get ingress.config.openshift.io cluster -o=jsonpath='{.spec.domain}') && ./e2e.test -test.v -ginkgo.v
.PHONY: test-e2e

images:
	$(DOCKER_BUILDER) build -f Dockerfile . -t $(CLUSTER_PROXY_ADDON_IMAGE)
.PHONY: images

images-amd64:
	$(DOCKER_BUILDER) buildx build --platform linux/amd64 --load -f Dockerfile . -t $(CLUSTER_PROXY_ADDON_IMAGE)
.PHONY: images-amd64
