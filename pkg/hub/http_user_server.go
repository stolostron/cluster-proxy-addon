package hub

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/openshift/library-go/pkg/controller/controllercmd"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	grpccredentials "google.golang.org/grpc/credentials"
	"k8s.io/klog/v2"
	konnectivity "sigs.k8s.io/apiserver-network-proxy/konnectivity-client/pkg/client"
	"sigs.k8s.io/apiserver-network-proxy/pkg/util"
)

type HTTPUserServer struct {
	// TODO: make it a controller and reuse tunnel for each cluster to improve performance.
	getTunnel       func() (konnectivity.Tunnel, error)
	proxyServerHost string
	proxyServerPort int

	proxyCACertPath, proxyCertPath, proxyKeyPath string

	serverCert, serverKey string
	serverPort            int
}

func (k *HTTPUserServer) AddFlags(cmd *cobra.Command) {
	flags := cmd.Flags()

	flags.StringVar(&k.proxyServerHost, "host", k.proxyServerHost, "The host of the ANP proxy-server")
	flags.IntVar(&k.proxyServerPort, "port", k.proxyServerPort, "The port of the ANP proxy-server")

	flags.StringVar(&k.proxyCACertPath, "proxy-ca-cert", k.proxyCACertPath, "The path to the CA certificate of the ANP proxy-server")
	flags.StringVar(&k.proxyCertPath, "proxy-cert", k.proxyCertPath, "The path to the certificate of the ANP proxy-server")
	flags.StringVar(&k.proxyKeyPath, "proxy-key", k.proxyKeyPath, "The path to the key of the ANP proxy-server")

	flags.StringVar(&k.serverCert, "server-cert", k.serverCert, "Secure communication with this cert")
	flags.StringVar(&k.serverKey, "server-key", k.serverKey, "Secure communication with this key")
	flags.IntVar(&k.serverPort, "server-port", k.serverPort, "handle user request using this port")
}

func (k *HTTPUserServer) Validate() error {
	if k.serverCert == "" {
		return fmt.Errorf("The server-cert is required")
	}

	if k.serverKey == "" {
		return fmt.Errorf("The server-key is required")
	}

	if k.serverPort == 0 {
		return fmt.Errorf("The server-port is required")
	}

	return nil
}

func NewHTTPUserServer() *HTTPUserServer {
	return &HTTPUserServer{}
}

func (k *HTTPUserServer) init(ctx context.Context) error {
	proxyTLSCfg, err := util.GetClientTLSConfig(k.proxyCACertPath, k.proxyCertPath, k.proxyKeyPath, k.proxyServerHost, nil)
	if err != nil {
		return err
	}
	k.getTunnel = func() (konnectivity.Tunnel, error) {
		// instantiate a gprc proxy dialer
		tunnel, err := konnectivity.CreateSingleUseGrpcTunnel(
			ctx,
			net.JoinHostPort(k.proxyServerHost, strconv.Itoa(k.proxyServerPort)),
			grpc.WithTransportCredentials(grpccredentials.NewTLS(proxyTLSCfg)),
		)
		if err != nil {
			return nil, err
		}
		return tunnel, nil
	}
	return nil
}

func (k *HTTPUserServer) handler(wr http.ResponseWriter, req *http.Request) {
	if klog.V(4).Enabled() {
		dump, err := httputil.DumpRequest(req, true)
		if err != nil {
			http.Error(wr, err.Error(), http.StatusBadRequest)
			return
		}
		klog.V(4).Infof("request:\n%s", string(dump))
	}

	// parse clusterID from current requestURL
	clusterID, kubeAPIPath, err := parseRequestURL(req.RequestURI)
	if err != nil {
		http.Error(wr, err.Error(), http.StatusBadRequest)
		return
	}

	target := fmt.Sprintf("https://%s", clusterID)
	apiserverURL, err := url.Parse(target)
	if err != nil {
		http.Error(wr, err.Error(), http.StatusBadRequest)
		return
	}

	// TODO: the tunnel should be reused to improve performance.
	tunnel, err := k.getTunnel()
	if err != nil {
		http.Error(wr, err.Error(), http.StatusBadRequest)
		return
	}

	var proxyConn net.Conn
	defer func() {
		if proxyConn != nil {
			err = proxyConn.Close()
			if err != nil {
				klog.Errorf("connection closed: %v", err)
			}
		}
	}()

	proxy := httputil.NewSingleHostReverseProxy(apiserverURL)
	proxy.Transport = &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, // Skip server-auth for kube-apiserver
		},
		// golang http pkg automaticly upgrade http connection to http2 connection, but http2 can not upgrade to SPDY which used in "kubectl exec".
		// set ForceAttemptHTTP2 = false to prevent auto http2 upgration
		ForceAttemptHTTP2: false,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			// TODO: may find a way to cache the proxyConn.
			proxyConn, err = tunnel.DialContext(ctx, network, addr)
			return proxyConn, err
		},
	}

	proxy.ErrorHandler = func(rw http.ResponseWriter, r *http.Request, e error) {
		rw.Write([]byte(fmt.Sprintf("proxy to anp-proxy-server failed because %v", e)))
		klog.Errorf("proxy to anp-proxy-server failed because %v", e)
	}

	// update request URL path
	req.URL.Path = kubeAPIPath

	klog.V(4).Infof("request scheme:%s; rawQuery:%s; path:%s", req.URL.Scheme, req.URL.RawQuery, req.URL.Path)

	proxy.ServeHTTP(wr, req)
}

func parseRequestURL(requestURL string) (clusterID string, kubeAPIPath string, err error) {
	paths := strings.Split(requestURL, "/")
	if len(paths) <= 2 {
		err = fmt.Errorf("requestURL format not correct, path more than 2: %s", requestURL)
		return
	}
	clusterID = paths[1]                             // <clusterID>
	kubeAPIPath = strings.Join(paths[2:], "/")       // api/pods?timeout=32s
	kubeAPIPath = strings.Split(kubeAPIPath, "?")[0] // api/pods note: we only need path here, the proxy pkg would add params back
	return
}

func (k *HTTPUserServer) Run(ctx context.Context, controllerContext *controllercmd.ControllerContext) error {
	var err error

	if err = k.Validate(); err != nil {
		klog.Fatal(err)
	}

	if err = k.init(ctx); err != nil {
		klog.Fatal(err)
	}

	klog.Infof("start https server on %d", k.serverPort)
	http.HandleFunc("/", k.handler)

	err = http.ListenAndServeTLS(fmt.Sprintf(":%d", k.serverPort), k.serverCert, k.serverKey, nil)
	if err != nil {
		klog.Fatalf("failed to start user proxy server: %v", err)
	}

	return nil
}
