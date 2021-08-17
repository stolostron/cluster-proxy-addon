package config

const (
	ADDON            = "cluster-proxy"
	Organization     = "open-cluster-management"
	HUB_NAMESPACE    = "open-cluster-management"
	ADDON_AGENT_NAME = "cluster-proxy-addon-agent"
	SIGNER_NAME      = "open-cluster-management.io/cluster-proxy-signer"
)

const (
	APISERVER_PROXY_PORT = 8080
	ANP_SERVICE_NAME     = "cluster-proxy-service"
)

const (
	SignerSecret             = "cluster-proxy-signer"
	CaBundleConfigmap        = "cluster-proxy-ca-bundle"
	CaBundleConfigmapDataKey = "ca-bundle.crt"
)

const (
	AGENT_MANIFEST_FILES_DIR = "pkg/hub/addon/manifests"
	AGENT_NAMESPACE          = "open-cluster-management-agent-addon"
)

const DefaultImage = "quay.io/open-cluster-management/cluster-proxy-addon:latest"

const (
	AgentNamespaceFile = "namespace.yaml"
)

var AgentFiles = []string{
	"addon_deployment.yaml",
	"anp_deployment.yaml",
	"clusterrole.yaml",
	"clusterrolebinding.yaml",
	"configmap.yaml",
	"role.yaml",
	"rolebinding.yaml",
	"serviceaccount.yaml",
}
