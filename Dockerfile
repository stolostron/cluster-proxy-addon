FROM registry.ci.openshift.org/open-cluster-management/builder:go1.16-linux AS builder

WORKDIR /go/src/github.com/open-cluster-management/cluster-proxy-addon

COPY . .

ENV GO_PACKAGE github.com/open-cluster-management/cluster-proxy-addon
RUN make build-all

FROM registry.access.redhat.com/ubi8/ubi-minimal:latest


ENV USER_UID=10001

COPY --from=builder /go/src/github.com/open-cluster-management/cluster-proxy-addon/cluster-proxy /
COPY --from=builder /go/src/github.com/open-cluster-management/cluster-proxy-addon/proxy-agent /
COPY --from=builder /go/src/github.com/open-cluster-management/cluster-proxy-addon/proxy-server /

RUN microdnf update && microdnf clean all

USER ${USER_UID}
