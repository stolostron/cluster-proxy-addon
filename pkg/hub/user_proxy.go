package hub

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/openshift/library-go/pkg/controller/controllercmd"
	"github.com/spf13/cobra"
	"github.com/stolostron/cluster-proxy-addon/pkg/config"
	"github.com/stolostron/cluster-proxy-addon/pkg/helpers"
	"k8s.io/apimachinery/pkg/api/meta"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	addonv1alpha1client "open-cluster-management.io/api/client/addon/clientset/versioned"
)

type UserServerOptions struct {
	proxyUdsName string
	serverCert   string
	serverKey    string
	serverPort   int
	addonClient  addonv1alpha1client.Interface
}

func NewUserServerOptions() *UserServerOptions {
	return &UserServerOptions{}
}

func (u *UserServerOptions) AddFlags(cmd *cobra.Command) {
	flags := cmd.Flags()
	flags.StringVar(&u.proxyUdsName, "proxy-uds", u.proxyUdsName, "the UDS name to connect to")
	flags.StringVar(&u.serverCert, "server-cert", u.serverCert, "Secure communication with this cert")
	flags.StringVar(&u.serverKey, "server-key", u.serverKey, "Secure communication with this key")
	flags.IntVar(&u.serverPort, "server-port", u.serverPort, "handle user request using this port")
}

func (u *UserServerOptions) Validate() error {
	if u.proxyUdsName == "" {
		return fmt.Errorf("The proxy-uds is required")
	}

	if u.serverCert == "" {
		return fmt.Errorf("The server-cert is required")
	}

	if u.serverKey == "" {
		return fmt.Errorf("The server-key is required")
	}

	if u.serverPort == 0 {
		return fmt.Errorf("The server-port is required")
	}

	return nil
}

func (u *UserServerOptions) Handler(wr http.ResponseWriter, req *http.Request) {
	if u.addonClient == nil {
		http.Error(wr, fmt.Sprintf("addon client is nil"), http.StatusInternalServerError)
		return
	}

	if klog.V(4).Enabled() {
		dump, err := httputil.DumpRequest(req, true)
		if err != nil {
			http.Error(wr, err.Error(), http.StatusBadRequest)
			return
		}
		klog.V(4).Infof("request:\n%s", string(dump))
	}

	// parse clusterID from current requestURL
	clusterID, kubeAPIPath, err := helpers.ParseRequestURL(req.RequestURI)
	if err != nil {
		http.Error(wr, err.Error(), http.StatusBadRequest)
		return
	}

	// check if cluster-proxy addon installed in target managed cluster
	clusterProxyAddon, err := u.addonClient.AddonV1alpha1().ManagedClusterAddOns(clusterID).Get(context.TODO(), config.ADDON, v1.GetOptions{})
	if err != nil {
		http.Error(wr, err.Error(), http.StatusBadRequest)
		return
	}

	if meta.IsStatusConditionFalse(clusterProxyAddon.Status.Conditions, "ManifestApplied") {
		http.Error(wr, "manifestwork are not applied to agent yet", http.StatusInternalServerError)
		return
	}

	// restruct new apiserverURL
	target := fmt.Sprintf("http://%s:%d", clusterID, config.APISERVER_PROXY_PORT)
	apiserverURL, err := url.Parse(target)
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

	// TODO reuse connection
	proxy := httputil.NewSingleHostReverseProxy(apiserverURL)
	proxy.Transport = &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		// golang http pkg automaticly upgrade http connection to http2 connection, but http2 can not upgrade to SPDY which used in "kubectl exec".
		// set ForceAttemptHTTP2 = false to prevent auto http2 upgration
		ForceAttemptHTTP2: false,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			var err error
			proxyConn, err = net.Dial("unix", u.proxyUdsName)
			if err != nil {
				return nil, fmt.Errorf("dialing proxy %q failed: %v", u.proxyUdsName, err)
			}

			requestAddress := fmt.Sprintf("%s:%d", clusterID, config.APISERVER_PROXY_PORT)
			fmt.Fprintf(proxyConn, "CONNECT %s HTTP/1.1\r\nHost: 127.0.0.1\r\nUser-Agent: user-agent\r\n\r\n", requestAddress)
			br := bufio.NewReader(proxyConn)
			res, err := http.ReadResponse(br, nil)
			if err != nil {
				return nil, fmt.Errorf("reading HTTP response from CONNECT to %s via uds proxy %s failed: %v",
					requestAddress, u.proxyUdsName, err)
			}
			if res.StatusCode != 200 {
				return nil, fmt.Errorf("proxy error from %s while dialing %s: %v", u.proxyUdsName, requestAddress, res.Status)
			}

			// It's safe to discard the bufio.Reader here and return the
			// original TCP conn directly because we only use this for
			// TLS, and in TLS the client speaks first, so we know there's
			// no unbuffered data. But we can double-check.
			if br.Buffered() > 0 {
				return nil, fmt.Errorf("unexpected %d bytes of buffered data from CONNECT uds proxy %q",
					br.Buffered(), u.proxyUdsName)
			}

			return proxyConn, err
		},
	}

	proxy.ErrorHandler = func(rw http.ResponseWriter, r *http.Request, e error) {
		rw.Write([]byte(fmt.Sprintf("proxy to anp-proxy-server failed because %v", err)))
	}

	// update request URL path
	req.URL.Path = kubeAPIPath
	// update proto
	req.Proto = "http"
	klog.V(4).Infof("request scheme:%s; rawQuery:%s; path:%s", req.URL.Scheme, req.URL.RawQuery, req.URL.Path)

	proxy.ServeHTTP(wr, req)
}

func (u *UserServerOptions) Run(ctx context.Context, controllerContext *controllercmd.ControllerContext) error {
	var err error
	u.addonClient, err = addonv1alpha1client.NewForConfig(controllerContext.KubeConfig)
	if err != nil {
		klog.Fatal(err)
	}

	if err = u.Validate(); err != nil {
		klog.Fatal(err)
	}

	klog.Infof("start https server on %d", u.serverPort)
	http.HandleFunc("/", u.Handler)

	err = http.ListenAndServeTLS(fmt.Sprintf(":%d", u.serverPort), u.serverCert, u.serverKey, nil)
	if err != nil {
		klog.Fatalf("failed to start user proxy server: %v", err)
	}

	return nil
}
