package constant

const (
	AgentInstallNamespace = "open-cluster-management-agent-addon"

	ServiceProxyPort = 7443

	ServerCertSecretName = "cluster-proxy-service-proxy-server-cert"

	ServiceProxyName = "cluster-proxy-service-proxy"

	AddonName = "cluster-proxy"

	// ExposedServicesConfigMapName is the default name of the ConfigMap that
	// controls which services are reachable via the service proxy path.
	ExposedServicesConfigMapName = "cluster-proxy-exposed-services"
)
