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

GIT_HOST ?= github.com/open-cluster-management
BASE_DIR := $(shell basename $(PWD))
DEST := $(GOPATH)/src/$(GIT_HOST)/$(BASE_DIR)

# CSV_VERSION is used to generate new CSV manifests
CSV_VERSION?=0.1.0

OPERATOR_SDK?=$(PERMANENT_TMP_GOPATH)/bin/operator-sdk
OPERATOR_SDK_VERSION?=v1.1.0
OPERATOR_SDK_ARCHOS:=x86_64-linux-gnu
ifeq ($(GOHOSTOS),darwin)
	ifeq ($(GOHOSTARCH),amd64)
		OPERATOR_SDK_ARCHOS:=x86_64-apple-darwin
	endif
endif
operatorsdk_gen_dir:=$(dir $(OPERATOR_SDK))

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

update-csv: ensure-operator-sdk
	cd deploy && rm -rf olm-catalog/manifests && ../$(OPERATOR_SDK) generate bundle --manifests --deploy-dir config/ --crds-dir config/crds/ --output-dir olm-catalog/ --version $(CSV_VERSION)

deploy-addon: ensure-operator-sdk
	$(OPERATOR_SDK) run packagemanifests deploy/olm-catalog/ --namespace open-cluster-management --version $(CSV_VERSION) --install-mode OwnNamespace --timeout=10m

clean-addon: ensure-operator-sdk
	$(OPERATOR_SDK) cleanup submariner-addon --namespace open-cluster-management --timeout 10m

ensure-operator-sdk:
ifeq "" "$(wildcard $(OPERATOR_SDK))"
	$(info Installing operator-sdk into '$(OPERATOR_SDK)')
	mkdir -p '$(operatorsdk_gen_dir)'
	curl -s -f -L https://github.com/operator-framework/operator-sdk/releases/download/$(OPERATOR_SDK_VERSION)/operator-sdk-$(OPERATOR_SDK_VERSION)-$(OPERATOR_SDK_ARCHOS) -o '$(OPERATOR_SDK)'
	chmod +x '$(OPERATOR_SDK)';
else
	$(info Using existing operator-sdk from "$(OPERATOR_SDK)")
endif

# TODO
# include ./test/integration-test.mk

ANP_DIR ?= dependencymagnet/anp

build-addon: $(ANP_DIR)/cmd/server/main.go $(ANP_DIR)/cmd/agent/main.go cmd/cluster-proxy/main.go
	rm -rf bin/
	cd $(ANP_DIR) && go build -o bin/proxy-agent cmd/agent/main.go
	cd $(ANP_DIR) && go build -o bin/proxy-server cmd/server/main.go
	cp -r $(ANP_DIR)/bin bin/
	go build -o bin/cluster-proxy cmd/cluster-proxy/main.go
	