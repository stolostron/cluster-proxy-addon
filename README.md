# Cluster Proxy Addon

`cluster-proxy-addon` uses a reverse proxy server (anp) to send the request from hub to managed cluster.
And it also contains e2e test for the overall cluster-proxy-addon case.

This feature has 5 relevant repos:
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
* [hub-crds](https://github.com/stolostron/hub-crds)
    * As cicd required, we can use CRD in chart and all crds mush add into this repo.

## Notes

### Update CRD in the `hub-crds` repo.

We should update the CRD file in `hub-crds` after we changed any thing in `cluster-proxy` CRD file.

The path of crd file in cluster-proxy is: `hack/crd/bases/proxy.open-cluster-management.io_managedproxyconfigurations.yaml`
