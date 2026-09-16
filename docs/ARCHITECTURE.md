# Architecture

## Purpose and scope

`cluster-proxy-addon` is the downstream HTTP-only portion of the cluster proxy used by MCE 2.10 and earlier. It connects hub-cluster callers to services and Kubernetes APIs on managed clusters through an apiserver-network-proxy (ANP) tunnel. The repository is deprecated for newer MCE releases; new functionality belongs in `cluster-proxy`.

## Runtime components

The `cluster-proxy` binary exposes three independent Cobra subcommands, deployed as separate workloads by the Helm chart:

- **User server** (`pkg/userserver`): accepts HTTPS requests from hub users and internal clients. It parses the request target, maps the managed cluster to the hashed service-proxy hostname, creates a single-use gRPC ANP tunnel to the proxy server, and reverse-proxies the request through that tunnel.
- **Service proxy** (`pkg/serviceproxy`): accepts HTTPS traffic from ANP proxy agents. It resolves the target service, optionally authenticates Kubernetes API requests with TokenReview calls against both the managed cluster and hub, adds impersonation headers for hub users, replaces the bearer token with its service-account token, and forwards the request to the target service.
- **Controller manager** (`pkg/controllers`): runs the certificate controller and controller-runtime health/metrics endpoints. It watches Kubernetes Secrets used for certificate signing and supports leader election.

The chart also deploys the ANP manager and agent components supplied by the cluster-proxy image, creates the service accounts/RBAC and certificates, and installs the `ManagedProxyConfiguration` and `ManagedProxyServiceResolver` CRDs.

## Request flow

The normal request path is:

```text
hub client
  -> user-server HTTPS
  -> ANP proxy-server through a single-use gRPC tunnel
  -> ANP proxy-agent on the managed cluster
  -> service-proxy HTTPS
  -> target service or managed-cluster Kubernetes API
```

The user server derives the service-proxy host from a SHA-256 hash of the managed cluster name. The service proxy uses the in-cluster service-account CA and optional OpenShift service CA to establish TLS to downstream services. HTTP/2 upgrade is disabled on proxy transports because SPDY is required for operations such as `kubectl exec`.

## Authentication and trust boundaries

- The user server terminates client TLS and authenticates its ANP connection with configured client certificates.
- For Kubernetes API requests, the service proxy first validates the supplied bearer token against the managed cluster. If that fails, it validates it against the hub and converts an authenticated hub identity into Kubernetes impersonation headers.
- The service proxy uses its mounted service-account token for the impersonating request. Kubernetes RBAC and the chart's service-account permissions therefore define the effective authorization boundary.
- Certificates and keys are mounted into the workloads by the Helm deployment. Health probes use the add-on framework configuration checker to detect invalid certificate material.

## Configuration and validation

Runtime flags are defined beside each subcommand. Helm templates supply certificate paths, ANP endpoints, namespaces, image overrides, and kubeconfig data. Unit tests cover utility behavior under `pkg/...`; e2e tests exercise OCM installation, chart deployment, and end-to-end proxy traffic on an OpenShift cluster.
