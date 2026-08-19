package userserver

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/spf13/cobra"
	"github.com/stolostron/cluster-proxy-addon/pkg/constant"
	"github.com/stolostron/cluster-proxy-addon/pkg/utils"
	"google.golang.org/grpc"
	grpccredentials "google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	addonutils "open-cluster-management.io/addon-framework/pkg/utils"
	konnectivity "sigs.k8s.io/apiserver-network-proxy/konnectivity-client/pkg/client"
	"sigs.k8s.io/apiserver-network-proxy/pkg/util"
	ctrlconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
)

func NewUserServerCommand() *cobra.Command {
	userServer := newUserServer()

	cmd := &cobra.Command{
		Use:   "user-server",
		Short: "user-server",
		Long:  `A http proxy server, receives http requests from users and forwards to the ANP proxy-server.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return userServer.Run(cmd.Context())
		},
	}

	userServer.AddFlags(cmd)
	return cmd
}

var (
	serviceProxyRootCA *x509.CertPool
)

type userServer struct {
	// TODO: make it a controller and reuse tunnel for each cluster to improve performance.
	getTunnel       func(context.Context) (konnectivity.Tunnel, error)
	proxyServerHost string
	proxyServerPort int

	proxyCACertPath, proxyCertPath, proxyKeyPath string

	serverCert, serverKey string
	serverPort            int

	serviceProxyCACertPath string
	agentInstallNamespace  string

	// exposedServicesConfigMap is the name of the ConfigMap (in the pod's own
	// namespace) that lists permitted service proxy targets. Defaults to the
	// well-known constant ExposedServicesConfigMapName.
	exposedServicesConfigMap string
	// serviceAllowlist is populated at startup from the ConfigMap and kept
	// up to date by an informer.
	serviceAllowlist *ServiceAllowlist
}

func (k *userServer) AddFlags(cmd *cobra.Command) {
	flags := cmd.Flags()

	flags.StringVar(&k.proxyServerHost, "host", k.proxyServerHost, "The host of the ANP proxy-server")
	flags.IntVar(&k.proxyServerPort, "port", k.proxyServerPort, "The port of the ANP proxy-server")

	flags.StringVar(&k.proxyCACertPath, "proxy-ca-cert", k.proxyCACertPath, "The path to the CA certificate of the ANP proxy-server")
	flags.StringVar(&k.proxyCertPath, "proxy-cert", k.proxyCertPath, "The path to the certificate of the ANP proxy-server")
	flags.StringVar(&k.proxyKeyPath, "proxy-key", k.proxyKeyPath, "The path to the key of the ANP proxy-server")

	flags.StringVar(&k.serverCert, "server-cert", k.serverCert, "Secure communication with this cert")
	flags.StringVar(&k.serverKey, "server-key", k.serverKey, "Secure communication with this key")
	flags.IntVar(&k.serverPort, "server-port", k.serverPort, "handle user request using this port")

	flags.StringVar(&k.serviceProxyCACertPath, "service-proxy-ca-cert", k.serviceProxyCACertPath, "The path to the CA certificate of the service proxy server")

	flags.StringVar(&k.agentInstallNamespace, "agent-install-namespace", k.agentInstallNamespace, "The namespace of the agent install")

	flags.StringVar(&k.exposedServicesConfigMap, "exposed-services-configmap", constant.ExposedServicesConfigMapName,
		"Name of the ConfigMap (in the pod's namespace) that lists which services are reachable via the service proxy path")
}

func (k *userServer) Validate() error {
	if k.serverCert == "" {
		return fmt.Errorf("The server-cert is required")
	}

	if k.serverKey == "" {
		return fmt.Errorf("The server-key is required")
	}

	if k.serverPort == 0 {
		return fmt.Errorf("The server-port is required")
	}

	if k.serviceProxyCACertPath == "" {
		return fmt.Errorf("The serviceproxy-ca-cert is required")
	}

	return nil
}

func newUserServer() *userServer {
	return &userServer{}
}

func (k *userServer) init(ctx context.Context, kubeClient kubernetes.Interface, podNamespace string) error {
	proxyTLSCfg, err := util.GetClientTLSConfig(k.proxyCACertPath, k.proxyCertPath, k.proxyKeyPath, k.proxyServerHost, nil)
	if err != nil {
		return err
	}

	// prepare ca for sevice proxy server
	serviceProxyCaCert, err := ioutil.ReadFile(k.serviceProxyCACertPath)
	if err != nil {
		return err
	}
	serviceProxyRootCA = x509.NewCertPool()
	if ok := serviceProxyRootCA.AppendCertsFromPEM(serviceProxyCaCert); !ok {
		return fmt.Errorf("failed to parse service proxy ca cert")
	}

	k.getTunnel = func(tunnelCtx context.Context) (konnectivity.Tunnel, error) {
		// instantiate a gprc proxy dialer
		tunnel, err := konnectivity.CreateSingleUseGrpcTunnelWithContext(
			ctx,
			tunnelCtx,
			net.JoinHostPort(k.proxyServerHost, strconv.Itoa(k.proxyServerPort)),
			grpc.WithTransportCredentials(grpccredentials.NewTLS(proxyTLSCfg)),
			grpc.WithKeepaliveParams(keepalive.ClientParameters{
				Time: time.Minute * 10,
			}),
		)
		if err != nil {
			return nil, err
		}
		return tunnel, nil
	}

	// Start the service allowlist watcher. The watcher enforces default-deny:
	// only services listed in the ConfigMap are reachable via the service proxy
	// path. Kube-apiserver proxy requests are not subject to this check.
	k.serviceAllowlist, err = startServiceAllowlistWatcher(ctx, kubeClient, podNamespace, k.exposedServicesConfigMap)
	if err != nil {
		return fmt.Errorf("failed to start service allowlist watcher: %w", err)
	}
	klog.Infof("service allowlist active: %d entries loaded from ConfigMap %s/%s",
		k.serviceAllowlist.Len(), podNamespace, k.exposedServicesConfigMap)

	return nil
}

func (k *userServer) ServeHTTP(wr http.ResponseWriter, req *http.Request) {
	if klog.V(4).Enabled() {
		dump, err := httputil.DumpRequest(req, true)
		if err != nil {
			http.Error(wr, err.Error(), http.StatusBadRequest)
			return
		}
		klog.V(4).Infof("request:\n%s", string(dump))
	}

	var tsc utils.TargetServiceConfig
	var err error

	switch utils.GetProxyType(req.RequestURI) {
	case utils.ProxyTypeService:
		tsc, err = utils.GetTargetServiceConfig(req.RequestURI)
		if err != nil {
			http.Error(wr, err.Error(), http.StatusBadRequest)
			return
		}
		if !k.serviceAllowlist.IsAllowed(tsc) {
			klog.V(4).Infof("service proxy request denied: %s/%s is not in the exposed services allowlist",
				tsc.Namespace, tsc.Service)
			http.Error(wr,
				fmt.Sprintf("service %s/%s is not in the exposed services allowlist", tsc.Namespace, tsc.Service),
				http.StatusForbidden)
			return
		}
	case utils.ProxyTypeKubeAPIServer:
		tsc, err = utils.GetTargetServiceConfigForKubeAPIServer(req.RequestURI)
		if err != nil {
			http.Error(wr, err.Error(), http.StatusBadRequest)
			return
		}
	}

	// get service proxy host for current managed cluster
	targetURL, err := url.Parse(serviceProxyURL(tsc.Cluster))
	if err != nil {
		http.Error(wr, err.Error(), http.StatusBadRequest)
		return
	}

	tunnel, err := k.getTunnel(req.Context())
	if err != nil {
		http.Error(wr, err.Error(), http.StatusBadRequest)
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	proxy.Transport = &http.Transport{
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig: &tls.Config{
			RootCAs:    serviceProxyRootCA,
			MinVersion: tls.VersionTLS12,
		},
		// golang http pkg automaticly upgrade http connection to http2 connection, but http2 can not upgrade to SPDY which used in "kubectl exec".
		// set ForceAttemptHTTP2 = false to prevent auto http2 upgration
		ForceAttemptHTTP2: false,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			klog.V(4).Infof("proxy dial to %s", addr)
			// TODO: may find a way to cache the proxyConn.
			return tunnel.DialContext(ctx, network, addr)
		},
	}

	proxy.ErrorHandler = func(rw http.ResponseWriter, r *http.Request, e error) {
		http.Error(rw, fmt.Sprintf("proxy to anp-proxy-server failed because %v", e), http.StatusBadGateway)
		klog.Errorf("proxy to anp-proxy-server failed because %v", e)
	}

	klog.V(4).Infof("request scheme:%s; rawQuery:%s; path:%s", req.URL.Scheme, req.URL.RawQuery, req.URL.Path)

	proxy.ServeHTTP(wr, utils.UpdateRequest(tsc, req))
}

func (k *userServer) Run(ctx context.Context) error {
	var err error

	klog.Info("begin to run user server")

	if err = k.Validate(); err != nil {
		klog.Fatal(err)
	}

	podNamespace := os.Getenv("POD_NAMESPACE")
	if len(podNamespace) == 0 {
		klog.Fatalf("Pod namespace is empty, please set the ENV for POD_NAMESPACE")
	}

	// Create the kube client for the service allowlist watcher.
	kubeClient, err := kubernetes.NewForConfig(ctrlconfig.GetConfigOrDie())
	if err != nil {
		klog.Fatalf("failed to create kube client: %v", err)
	}

	if err = k.init(ctx, kubeClient, podNamespace); err != nil {
		klog.Fatal(err)
	}

	cc, err := addonutils.NewConfigChecker("user-server", k.proxyCACertPath, k.proxyCertPath, k.proxyKeyPath, k.serverCert, k.serverKey, k.serviceProxyCACertPath)
	if err != nil {
		klog.Fatal(err)
	}

	go func() {
		if err = utils.ServeHealthProbes(":8000", cc.Check); err != nil {
			klog.Fatal(err)
		}
	}()

	klog.Infof("start https server on %d", k.serverPort)

	s := &http.Server{
		Addr:      fmt.Sprintf(":%d", k.serverPort),
		TLSConfig: &tls.Config{MinVersion: tls.VersionTLS12},
		Handler:   k,
	}

	err = s.ListenAndServeTLS(k.serverCert, k.serverKey)
	if err != nil {
		klog.Fatalf("failed to start user proxy server: %v", err)
	}

	return nil
}

// here use the same logic as in the cluster-proxy repo:
// https://github.com/stolostron/cluster-proxy/blob/304b2ded6c1a651be9ba0f15af4edf1f65ac29df/pkg/proxyagent/agent/agent.go#L297
func serviceProxyURL(clusterName string) string {
	serviceProxyHost := fmt.Sprintf("cluster-%x", sha256.Sum256([]byte(clusterName)))[:64-len("cluster-")] + ".open-cluster-management.proxy"
	return fmt.Sprintf("https://%s:%d", serviceProxyHost, constant.ServiceProxyPort)
}
