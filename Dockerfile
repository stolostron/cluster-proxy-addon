FROM registry.ci.openshift.org/open-cluster-management/builder:go1.16-linux AS builder
WORKDIR /go/src/github.com/open-cluster-management/cluster-proxy-addon
COPY . .
ENV GO_PACKAGE github.com/open-cluster-management/cluster-proxy-addon
RUN make build --warn-undefined-variables

FROM registry.access.redhat.com/ubi8/ubi-minimal:latest
COPY --from=builder /go/src/github.com/open-cluster-management/cluster-proxy-addon/cluster-proxy /
RUN microdnf update && microdnf clean all
