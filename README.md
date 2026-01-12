# Cluster Proxy Addon

> **⚠️ Deprecation Notice**
>
> Starting from **MCE 2.11**, this repository has been deprecated. All functionality has been integrated into the [cluster-proxy](https://github.com/stolostron/cluster-proxy) repository.
>
> This repository only serves **MCE 2.10 and earlier versions**. For MCE 2.11 and later, please refer to [cluster-proxy](https://github.com/stolostron/cluster-proxy).

`cluster-proxy-addon` uses a reverse proxy server (ANP) to send requests from the hub to managed clusters.
It also contains end-to-end tests for the overall cluster-proxy-addon functionality.

This feature has 7 relevant repos:
* [cluster-proxy-addon](https://github.com/stolostron/cluster-proxy-addon)
    * Contains the ANP image (currently version 0.0.24)
    * Contains the HTTP-User-Server image to support users accessing the cluster-proxy via HTTP.
* [cluster-proxy](https://github.com/stolostron/cluster-proxy)
    * The main body of the addon.
    * This repository has some differences in the build and deployment components compared to the upstream repository: [cluster-proxy](https://github.com/open-cluster-management-io/cluster-proxy) to suit downstream needs.
* [backplane-operator](https://github.com/stolostron/backplane-operator/tree/main/pkg/templates/charts/toggle/cluster-proxy-addon)
    * The latest chart changes should be made in the backplane-operator template.
* [cluster-proxy-addon-chart](https://github.com/stolostron/cluster-proxy-addon-chart)
    * Only for release-2.5 and earlier versions.
    * Uses images from `cluster-proxy-addon` and `cluster-proxy` to deploy the addon.
    * This repository is leveraged by [multiclusterhub-operator](https://github.com/stolostron/multiclusterhub-operator) and [multiclusterhub-repo](https://github.com/stolostron/multiclusterhub-repo).
* [release](https://github.com/openshift/release)
    * Contains the CI/CD pipeline steps for the addon.
* [apiserver-network-proxy](https://github.com/stolostron/apiserver-network-proxy/tree/v0.1.6-patch)
    * This repository is a forked version to support downstream needs.
* [grpc-go](https://github.com/stolostron/grpc-go)
    * This repository is a forked version to support proxy-in-the-middle use cases.
    * Used in: https://github.com/stolostron/apiserver-network-proxy/blob/d562699c78201daef7dec97cd1847e5abffbe2ab/go.mod#L5C43-L5C72

## Recommended Usage for Internal Operators/Controllers/Services

The cluster-proxy-addon exposes a `Route` for external users to access managed clusters. However, for internal operators/controllers/services, it's recommended to use the `Service` to access managed clusters. The `Service` is more efficient than the `Route` within the internal network.

Here is a code example showing how to use the cluster-proxy-addon `Service` to access managed cluster pod logs. The complete example can be found in [multicloud-operators-foundation](https://github.com/stolostron/multicloud-operators-foundation/blob/main/pkg/proxyserver/getter/logProxyGetter.go):

```go
    // There must be a managedserviceaccount with proper rolebinding in the managed cluster.
    logTokenSecret, err := c.KubeClient.CoreV1().Secrets(clusterName).Get(ctx, helpers.LogManagedServiceAccountName, v1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get log token secret in cluster %s. %v", clusterName, err)
	}

    // Configure a Kubernetes Config.
	clusterProxyCfg := &rest.Config{
        // The `ProxyServiceHost` normally is the service domain name of the cluster-proxy-addon user-server:
        // cluster-proxy-addon-user.<component namespace>.svc:9092
		Host: fmt.Sprintf("https://%s/%s", c.ProxyServiceHost, clusterName),
		TLSClientConfig: rest.TLSClientConfig{
            // The CAFile must be the openshift-service-ca.crt, because the user-server uses the OpenShift service CA to sign the certificate.
            // You can mount the openshift-service-ca.crt to the pod from a configmap named `openshift-service-ca.crt` in every namespace.
			CAFile: c.ProxyServiceCAFile,
		},
		BearerToken: string(logTokenSecret.Data["token"]),
	}
	clusterProxyKubeClient, err := kubernetes.NewForConfig(clusterProxyCfg)
	if err != nil {
		return nil, err
	}
```

For more detailed examples and usage patterns, please refer to the [multicloud-operators-foundation](https://github.com/stolostron/multicloud-operators-foundation/blob/main/pkg/proxyserver/getter/logProxyGetter.go) repository.

## Q&A

### Does the `cluster-proxy-addon` support `grpc` mode like the community version of [cluster-proxy](https://github.com/open-cluster-management-io/cluster-proxy)?

No, the `cluster-proxy-addon` doesn't support `grpc` mode. The `cluster-proxy-addon` only supports `http` mode.
This is due to security considerations. The cluster-proxy-addon uses the flag [`--enable-kube-api-proxy`](https://github.com/open-cluster-management-io/cluster-proxy/blob/ae20540551aaefc9a0f894795bd688ac6f5727aa/cmd/addon-manager/main.go#L96) set to `false`, which prevents the cluster-proxy from using the `managedcluster name` as one of the agent-identifiers.

The reason we avoid using the `managedcluster name` as one of the agent-identifiers is that in some customer environments, the managedcluster name begins with numbers, which is not a valid domain name. However, the agent identifier is used as a domain name in `grpc` mode.

Currently, all requests from the hub to managed clusters follow this pattern:

```
 client -> user-server -> proxy-server(ANP) -> proxy-agent(ANP) -> service-proxy -> target-service
```