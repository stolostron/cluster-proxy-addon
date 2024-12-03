FROM registry.ci.openshift.org/stolostron/builder:go1.22-linux AS builder

WORKDIR /go/src/github.com/stolostron/cluster-proxy-addon

COPY . .

RUN make build-all

FROM registry.access.redhat.com/ubi9/ubi-minimal:latest


ENV USER_UID=10001

COPY --from=builder /go/src/github.com/stolostron/cluster-proxy-addon/cluster-proxy /
COPY --from=builder /go/src/github.com/stolostron/cluster-proxy-addon/proxy-agent /
COPY --from=builder /go/src/github.com/stolostron/cluster-proxy-addon/proxy-server /

RUN microdnf update && microdnf clean all

USER ${USER_UID}
