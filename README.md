# Cluster Proxy Addon

`cluster-proxy-addon` uses a reverse proxy server (anp) to send the request from hub to managed cluster.
And it also contains e2e test for the overall cluster-proxy-addon case.

This feature has 7 relevant repos:
* [cluster-proxy-addon](https://github.com/stolostron/cluster-proxy-addon)
    * Contains the ANP image (currently version 0.0.24)
    * Contains the HTTP-User-Server image to support users using http way to access the cluster-proxy.
* [cluster-proxy](https://github.com/stolostron/cluster-proxy)
    * The main body of the addon.
    * The repo has some differences in build and deploy part with the upstream repo: [cluster-proxy](https://github.com/open-cluster-management-io/cluster-proxy) to suit the downstream needs.
* [backplane-operator](https://github.com/stolostron/backplane-operator/tree/main/pkg/templates/charts/toggle/cluster-proxy-addon)
    * The lastest chart changes should be made in backplane-operator template.
* [cluster-proxy-addon-chart](https://github.com/stolostron/cluster-proxy-addon-chart)
    * Only for release-2.5 and early version.
    * Using images from `cluster-proxy-addon` and `cluster-proxy` to deploy the addon.
    * The repo is leveraged by [multiclusterhub-operator](https://github.com/stolostron/multiclusterhub-operator) and [multiclusterhub-repo](https://github.com/stolostron/multiclusterhub-repo).
* [release](https://github.com/openshift/release)
    * Contains the cicd steps of the addon.
* [apiserver-network-proxy](https://github.com/stolostron/apiserver-network-proxy/tree/v0.1.6-patch)
    * The repo is a forked version to support the downstream needs.
* [grpc-go](https://github.com/stolostron/grpc-go)
    * The repo is a forked version to support the proxy-in-middle cases.
    * Used in: https://github.com/stolostron/apiserver-network-proxy/blob/d562699c78201daef7dec97cd1847e5abffbe2ab/go.mod#L5C43-L5C72

## The recommended way for **internal** operator/controller/service to leverage the cluster-proxy-addon

The cluster-proxy-addon exposed a `Route` for the users in outside world to access the managed clusters. But for the internal operator/controller/service, it's recommended to use the `Service` to access the managed clusters. The `Service` is also more efficient than the `Route` in the internal network.

Here is a piece of code to show how to use the cluster-proxy-addon `Service` to access the managed clusters' pod logs, the full example can be found in [multicloud-operators-foundation](https://github.com/stolostron/multicloud-operators-foundation/blob/main/pkg/proxyserver/getter/logProxyGetter.go):

```go
    // There must be a managedserviceaccount with proper rolebinding in the managed cluster.
    logTokenSecret, err := c.KubeClient.CoreV1().Secrets(clusterName).Get(ctx, helpers.LogManagedServiceAccountName, v1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("faield to get log token secret in cluster %s. %v", clusterName, err)
	}

    // Configure a kuberentes Config.
	clusterProxyCfg := &rest.Config{
        // The `ProxyServiceHost` normally is the service domain name of the cluster-proxy-addon user-server:
        // cluster-proxy-addon-user.<component namespace>.svc:9092
		Host: fmt.Sprintf("https://%s/%s", c.ProxyServiceHost, clusterName),
		TLSClientConfig: rest.TLSClientConfig{
            // The CAFile must be the openshift-service-ca.crt, because user-server using openshift service CA to sign the certificate.
            // You can mount the openshift-service-ca.crt to the pod, a configmap named `openshift-service-ca.crt` in the every namespace.
			CAFile: c.ProxyServiceCAFile,
		},
		BearerToken: string(logTokenSecret.Data["token"]),
	}
	clusterProxyKubeClient, err := kubernetes.NewForConfig(clusterProxyCfg)
	if err != nil {
		return nil, err
	}
```

The full example can be found in:

## Q&A

### The community version of [cluster-proxy](https://github.com/open-cluster-management-io/cluster-proxy) support `grpc` mode, does the `cluster-proxy-addon` support it?

No, the `cluster-proxy-addon` doesn't support `grpc` mode. The `cluster-proxy-addon` only support `http` mode.
This is because for security reasons, cluster-proxy-addon using the flag [`--enable-kube-api-proxy`](https://github.com/open-cluster-management-io/cluster-proxy/blob/ae20540551aaefc9a0f894795bd688ac6f5727aa/cmd/addon-manager/main.go#L96). By setting the flag to `false`, the cluster-proxy won't use the `managedcluster name` as one of the agent-identifiers.

The reason why we don't want to use the `managedcluster name` as one of the agent-identifiers is that in some customer's environment, the managedcluster name begins with numbers, which is not a valid domain name. But the agent identifier is used as the domain name in the `grpc` mode.

Currently, all requests from the hub to the managed cluster follow pattern:

```
 client -> user-server -> proxy-server(ANP) -> proxy-agent(ANP) -> service-proxy -> target-service
```