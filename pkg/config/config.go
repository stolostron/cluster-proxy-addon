package config

const (
	ADDON             = "cluster-proxy"
	Organization      = "open-cluster-management"
	DEFAULT_NAMESPACE = "open-cluster-management"
	ADDON_NAME        = "cluster-proxy-addon"
	ADDON_AGENT_NAME  = "cluster-proxy-addon-agent"
	SIGNER_NAME       = "open-cluster-management.io/cluster-proxy-addon"
	AGENT_IMAGE       = "open-cluster-management.io/cluster-proxy-addon"
)

const (
	APISERVER_PROXY_PORT = 8080
)

const (
	SignerSecret            = "cluster-proxy-signer"
	CaBundleConfigmap       = "cluster-proxy-ca-bundle"
	ClusterProxyAddOnSecret = "cluster-proxy-addon-serving-cert"
	SignerNamePrefix        = "cluster-proxy-addon"
)

const (
	AGENT_MANIFEST_FILES_DIR = "deploy/managedcluster/manifest"
)

const DefaultImage = "quay.io/open-cluster-management/cluster-proxy-addon:latest"
