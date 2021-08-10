FROM registry.ci.openshift.org/open-cluster-management/builder:go1.16-linux AS builder

WORKDIR /go/src/github.com/open-cluster-management/cluster-proxy-addon
COPY . .
ENV GO_PACKAGE github.com/open-cluster-management/cluster-proxy-addon

RUN git submodule init
RUN git submodule update
WORKDIR /go/src/github.com/open-cluster-management/cluster-proxy-addon/dependencymagnet/anp 
RUN git checkout v0.0.22

WORKDIR /go/src/github.com/open-cluster-management/cluster-proxy-addon
RUN make build-addon

FROM registry.access.redhat.com/ubi8/ubi-minimal:latest
COPY --from=builder /go/src/github.com/open-cluster-management/cluster-proxy-addon/bin /
RUN microdnf update && microdnf clean all
