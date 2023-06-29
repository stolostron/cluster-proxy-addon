package utils

import (
	"crypto/sha256"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/stolostron/cluster-proxy-addon/pkg/constant"
	utilnet "k8s.io/apimachinery/pkg/util/net"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
)

const (
	HEADERSERVICECA   = "Service-Root-Ca"
	HEADERSERVICECERT = "Service-Client-Cert"
	HEADERSERVICEKEY  = "Service-Client-Key"
)

type TargetServiceConfig struct {
	Cluster   string
	Proto     string
	Service   string
	Namespace string
	Port      string
	Path      string
}

func GetTargetServiceConfigForKubeAPIServer(requestURL string) (ts TargetServiceConfig, err error) {
	paths := strings.Split(requestURL, "/")
	if len(paths) <= 2 {
		err = fmt.Errorf("requestURL format not correct, path more than 2: %s", requestURL)
		return
	}
	kubeAPIPath := strings.Join(paths[2:], "/")      // api/pods?timeout=32s
	kubeAPIPath = strings.Split(kubeAPIPath, "?")[0] // api/pods note: we only need path here, the proxy pkg would add params back
	return TargetServiceConfig{
		Cluster:   paths[1],
		Proto:     "https",
		Service:   "kubernetes",
		Namespace: "default",
		Port:      "443",
		Path:      kubeAPIPath,
	}, nil
}

func GetTargetServiceConfig(requestURL string) (ts TargetServiceConfig, err error) {
	urlparams := strings.Split(requestURL, "/")
	if len(urlparams) < 9 {
		err = fmt.Errorf("requestURL format not correct, path less than 9: %s", requestURL)
		return
	}

	// get targetHost
	namespace := urlparams[5]

	proto, service, port, valid := utilnet.SplitSchemeNamePort(urlparams[7])
	if !valid {
		return TargetServiceConfig{}, fmt.Errorf("invalid service name %q", urlparams[7])
	}
	if proto == "" {
		proto = "https" // set a default to https
	} else if proto != "https" {
		return TargetServiceConfig{}, fmt.Errorf("for security reasons, only support https yet, invaild proto: %s", proto)
	}

	// get servicePath
	servicePath := strings.Join(urlparams[9:], "/")
	servicePath = strings.Split(servicePath, "?")[0] //we only need path here, the proxy pkg would add params back

	return TargetServiceConfig{
		Cluster:   urlparams[1],
		Proto:     proto,
		Service:   service,
		Namespace: namespace,
		Port:      port,
		Path:      servicePath,
	}, nil
}

func (t TargetServiceConfig) UpdateRequest(req *http.Request) *http.Request {
	// update request URL path
	req.URL.Path = t.Path

	// populate proto, namespace, service, and port to request headers
	req.Header.Set("Cluster-Proxy-Proto", t.Proto)
	req.Header.Set("Cluster-Proxy-Namespace", t.Namespace)
	req.Header.Set("Cluster-Proxy-Service", t.Service)
	req.Header.Set("Cluster-Proxy-Port", t.Port)

	return req
}

func GetTargetServiceURLFromRequest(req *http.Request) (*url.URL, error) {
	// get proto, namespace, service, and port from request headers
	proto := req.Header.Get("Cluster-Proxy-Proto")
	namespace := req.Header.Get("Cluster-Proxy-Namespace")
	service := req.Header.Get("Cluster-Proxy-Service")
	port := req.Header.Get("Cluster-Proxy-Port")

	// validate proto, namespace, service, and port
	if proto == "" || namespace == "" || service == "" || port == "" {
		return nil, fmt.Errorf("invalid request headers")
	}

	var targetServiceURL string
	// check if the request is meant to proxy to kube-apiserver
	if proto == "https" && service == "kubernetes" && namespace == "default" && port == "443" {
		targetServiceURL = "https://kubernetes.default.svc"
	} else {
		targetServiceURL = fmt.Sprintf("%s://%s.%s.svc:%s", proto, service, namespace, port)
	}

	url, err := url.Parse(targetServiceURL)
	if err != nil {
		return nil, err
	}

	return url, nil
}

// TODO: replace with the util pkg provided by cluster-proxy later.
func GenerateServiceProxyURL(cluster, namespace, service string) string {
	// Using hash to generate a random string;
	// Sum256 will give a string with length equals 64. But the name of a service must be no more than 63 characters.
	// Also need to add "cluster-proxy-" as prefix to prevent content starts with a number.
	host := fmt.Sprintf("cluster-proxy-%x", sha256.Sum256([]byte(fmt.Sprintf("%s %s %s", cluster, namespace, service))))[:63]
	return fmt.Sprintf("https://%s:%d", host, constant.ServiceProxyPort)
}

// IsProxyService determines whether a request is meant to proxy to a target service.
// An example service URL: https://<route location cluster-proxy>/<managed_cluster_name>/api/v1/namespaces/<namespace_name>/services/<[https:]service_name[:port_name]>/proxy-service/<service_path>
func IsProxyService(reqURI string) bool {
	urlparams := strings.Split(reqURI, "/")
	if len(urlparams) > 9 && urlparams[8] == "proxy-service" {
		return true
	}
	return false
}

// ServeHealthProbes serves health probes and configchecker.
func ServeHealthProbes(healthProbeBindAddress string, customChecks ...healthz.Checker) error {
	mux := http.NewServeMux()

	checks := map[string]healthz.Checker{
		"healthz-ping": healthz.Ping,
	}

	for i, check := range customChecks {
		checks[fmt.Sprintf("custom-healthz-checker-%d", i)] = check
	}

	mux.Handle("/healthz", http.StripPrefix("/healthz", &healthz.Handler{Checks: checks}))
	server := http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		Addr:              healthProbeBindAddress,
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	}
	klog.Infof("heath probes server is running...")
	return server.ListenAndServe()
}
