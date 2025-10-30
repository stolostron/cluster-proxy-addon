# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

The cluster-proxy-addon is a Go-based Kubernetes addon that uses a reverse proxy server (apiserver-network-proxy/ANP) to enable secure communication from the hub cluster to managed clusters in a multi-cluster environment. This is part of the Open Cluster Management (OCM) ecosystem.

## Development Commands

### Building
```bash
# Build the main cluster-proxy binary
make build

# Build all components (includes ANP proxy-server and proxy-agent)
make build-all

# Build container images
make images

# Build images for AMD64 architecture specifically
make images-amd64
```

### Testing
```bash
# Build end-to-end tests
make build-e2e

# Run full end-to-end test suite (requires cluster setup)
make test-e2e

# Build locally with environment variable
export BUILD_LOCALLY=1
make
```

### Development Setup
```bash
# Deploy OCM (Open Cluster Management) for local testing
make deploy-ocm

# Deploy the addon for e2e testing
make deploy-addon-for-e2e
```

## Architecture

### Core Components

The main binary (`cmd/cluster-proxy/main.go`) provides three primary commands:

1. **userserver** (`pkg/userserver/`) - HTTP reverse proxy server that handles external requests and routes them through the ANP tunnel
2. **serviceproxy** (`pkg/serviceproxy/`) - Service proxy component that handles internal cluster-to-cluster communication
3. **controllers** (`pkg/controllers/`) - Kubernetes controllers for managing certificates and addon configuration

### Key Packages

- `pkg/constant/` - Shared constants and configuration values
- `pkg/utils/` - Common utility functions
- `pkg/version/` - Version information management

### Request Flow

All requests from hub to managed clusters follow this pattern:
```
client -> user-server -> proxy-server(ANP) -> proxy-agent(ANP) -> service-proxy -> target-service
```

### Helm Chart

The addon is deployed via Helm chart located in `chart/cluster-proxy-addon/` which includes:
- CRDs for ManagedProxyConfigurations and ManagedProxyServiceResolvers
- Templates for all components including the ClusterManagementAddon
- Default values configuration

### Dependencies

- **Go 1.23.6** (specified in go.mod)
- **Kubernetes client-go** for API interactions
- **Open Cluster Management** addon framework
- **apiserver-network-proxy** for secure tunneling (custom fork v0.1.6.patch-02)
- **grpc-go** (custom fork for proxy-in-the-middle support)

## Important Notes

### Security Model
- Only supports HTTP mode (not gRPC) for security reasons
- Uses `--enable-kube-api-proxy=false` to avoid using managedcluster names as agent-identifiers
- Relies on OpenShift service CA for certificate management

### Multi-Repository Context
This repository is part of a larger ecosystem with 7 related repositories. The main components are built here, but deployment configuration may come from:
- `backplane-operator` (for current releases)
- `cluster-proxy-addon-chart` (for release-2.5 and earlier)

### Testing Framework
- Uses Ginkgo/Gomega for BDD-style testing
- E2E tests validate proxy functionality with real cluster setups
- Tests include authentication, authorization, and network connectivity scenarios

## File Structure Context

- `/cmd/` - Main entry points and command definitions
- `/pkg/` - Core application logic organized by functionality
- `/test/e2e/` - End-to-end test suite with placement configurations
- `/chart/` - Helm chart for Kubernetes deployment
- `/vendor/` - Go module dependencies (vendored)
- `/.tekton/` - CI/CD pipeline definitions