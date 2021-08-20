package hub

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/open-cluster-management/cluster-proxy-addon/pkg/config"
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
)

type userProxyHandler struct {
	proxyUdsName string
	serverCert   string
	serverKey    string
	serverPort   int
}

func (u *userProxyHandler) AddFlags(cmd *cobra.Command) {
	flags := cmd.Flags()
	flags.StringVar(&u.proxyUdsName, "proxy-uds", u.proxyUdsName, "the UDS name to connect to")
	flags.StringVar(&u.serverCert, "server-cert", u.serverCert, "Secure communication with this cert")
	flags.StringVar(&u.serverKey, "server-key", u.serverKey, "Secure communication with this key")
	flags.IntVar(&u.serverPort, "server-port", u.serverPort, "handle user request using this port")
}

func (u *userProxyHandler) Validate() error {
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

func (u *userProxyHandler) Handler(wr http.ResponseWriter, req *http.Request) {
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
		klog.Errorf("parse request URL failed: %v", err)
		http.Error(wr, err.Error(), http.StatusBadRequest)
		return
	}

	// restruct new apiserverURL
	target := fmt.Sprintf("http://%s:%d", clusterID, config.APISERVER_PROXY_PORT)
	apiserverURL, err := url.Parse(target)
	if err != nil {
		klog.Errorf("parse restructed URL: %v", err)
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

	// update request URL path
	req.URL.Path = kubeAPIPath
	// update proto
	req.Proto = "http"
	klog.V(4).Infof("request scheme:%s; rawQuery:%s; path:%s", req.URL.Scheme, req.URL.RawQuery, req.URL.Path)

	proxy.ServeHTTP(wr, req)
}

// parseRequestURL
// Example Input: <service-ip>:8080/<clusterID>/api/pods?timeout=32s
// Example Output:
// 	clusterID: <clusterID>
// 	kubeAPIPath: api/pods
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

func NewUserProxy() *cobra.Command {
	proxy := &userProxyHandler{
		proxyUdsName: "/tmp/cluster-proxy-socket",
		serverPort:   9092,
	}

	cmd := &cobra.Command{
		Use:   "user-server",
		Short: "user-server",
		Run: func(cmd *cobra.Command, args []string) {
			if err := proxy.Validate(); err != nil {
				klog.Fatal(err)
			}

			klog.Infof("start https server on %d", proxy.serverPort)
			http.HandleFunc("/", proxy.Handler)
			err := http.ListenAndServeTLS(fmt.Sprintf(":%d", proxy.serverPort), proxy.serverCert, proxy.serverKey, nil)
			if err != nil {
				klog.Fatalf("failed to start user proxy server: %v", err)
			}
		},
	}

	proxy.AddFlags(cmd)
	return cmd
}
