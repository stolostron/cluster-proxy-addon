package hub

import (
	"crypto/sha256"
	"fmt"
	"strings"

	utilnet "k8s.io/apimachinery/pkg/util/net"
)

const (
	HEADERSERVICECA   = "Service-Root-Ca"
	HEADERSERVICECERT = "Service-Client-Cert"
	HEADERSERVICEKEY  = "Service-Client-Key"
)

// parseKubeAPIServerRequestURL
func parseKubeAPIServerRequestURL(requestURL string) (targetHost string, kubeAPIPath string, err error) {
	paths := strings.Split(requestURL, "/")
	if len(paths) <= 2 {
		err = fmt.Errorf("requestURL format not correct, path more than 2: %s", requestURL)
		return
	}
	targetHost = fmt.Sprintf("https://%s", paths[1]) // paths[1] <clusterID>
	kubeAPIPath = strings.Join(paths[2:], "/")       // api/pods?timeout=32s
	kubeAPIPath = strings.Split(kubeAPIPath, "?")[0] // api/pods note: we only need path here, the proxy pkg would add params back
	return
}

// parseServiceRequestURL
func parseServiceRequestURL(requestURL string) (targetHost string, servicePath string, err error) {
	urlparams := strings.Split(requestURL, "/")

	// get targetHost
	namespace := urlparams[5]

	proto, service, port, valid := utilnet.SplitSchemeNamePort(urlparams[7])
	if !valid {
		return "", "", fmt.Errorf("invalid service name %q", urlparams[7])
	}
	if proto == "" {
		proto = "https" // set a default value for proto
	}

	host := GenerateServiceURL(urlparams[1], namespace, service)

	// get servicePath
	servicePath = strings.Join(urlparams[9:], "/")
	servicePath = strings.Split(servicePath, "?")[0] //we only need path here, the proxy pkg would add params back

	targetHost = fmt.Sprintf("%s://%s:%s", proto, host, port)
	return targetHost, servicePath, nil
}

// TODO: replace with the util pkg provided by cluster-proxy later.
func GenerateServiceURL(cluster, namespace, service string) string {
	// Using hash to generate a random string;
	// Sum256 will give a string with length equals 64. But the name of a service must be no more than 63 characters.
	// Also need to add "cluster-proxy-" as prefix to prevent content starts with a number.
	content := sha256.Sum256([]byte(fmt.Sprintf("%s %s %s", cluster, namespace, service)))
	return fmt.Sprintf("cluster-proxy-%x", content)[:63]
}

// isProxyService determines whether a request is meant to proxy to a target service.
// An example service URL: https://<route location cluster-proxy>/<managed_cluster_name>/api/v1/namespaces/<namespace_name>/services/<[https:]service_name[:port_name]>/proxy-service/<service_path>
func isProxyService(reqURI string) bool {
	urlparams := strings.Split(reqURI, "/")
	if urlparams[2] == "api" && urlparams[3] == "v1" && urlparams[4] == "namespaces" && urlparams[6] == "services" && urlparams[8] == "proxy-service" {
		return true
	}
	return false
}
