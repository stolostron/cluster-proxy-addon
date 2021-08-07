package hub

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/open-cluster-management/cluster-proxy-addon/pkg/config"
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
)

const (
	FlagServerPort = "server-port"
	FlagProxyUds   = "proxy-uds"

	FlagServerCert = "server-cert"
	FlagServerKey  = "server-key"
)

const (
	ClusterRequestProto = "http"
	ProxyUds            = "/tmp/cluster-proxy-socket"
)

type userServer struct {
	proxyUdsName string
}

func newUserServer(proxyUdsName string) (*userServer, error) {
	return &userServer{
		proxyUdsName: proxyUdsName,
	}, nil
}

func (u *userServer) proxyHandler(wr http.ResponseWriter, req *http.Request) {
	// parse clusterID from current requestURL
	clusterID, kubeAPIPath, err := parseRequestURL(req.RequestURI)
	if err != nil {
		klog.Errorf("parse request URL failed: %v", err)
		http.Error(wr, err.Error(), http.StatusBadRequest)
		return
	}

	// restruct new apiserverURL
	target := fmt.Sprintf("%s://%s:%d", ClusterRequestProto, clusterID, config.APISERVER_PROXY_PORT)
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
			fmt.Fprintf(proxyConn, "CONNECT %s HTTP/1.1\r\nHost: %s\r\nUser-Agent: %s\r\n\r\n", requestAddress, "127.0.0.1")
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
	cmd := &cobra.Command{
		Use:   "anp-user-server",
		Short: "anp-user-server",
		Run: func(cmd *cobra.Command, args []string) {
			serverPort, err := cmd.Flags().GetInt(FlagServerPort)
			if err != nil {
				klog.Errorf("failed to read args %s: %v", FlagServerPort, err)
				return
			}
			proxyUds, err := cmd.Flags().GetString(FlagProxyUds)
			if err != nil {
				klog.Errorf("failed to read args %s: %v", FlagProxyUds, err)
				return
			}
			serverCert, err := cmd.Flags().GetString(FlagServerCert)
			if err != nil {
				klog.Errorf("failed to read args %s: %v", FlagServerCert, err)
				return
			}
			serverKey, err := cmd.Flags().GetString(FlagServerKey)
			if err != nil {
				klog.Errorf("failed to read args %s: %v", FlagServerKey, err)
				return
			}

			us, err := newUserServer(proxyUds)
			if err != nil {
				klog.Errorf("new user server failed: %v", err)
				return
			}

			http.HandleFunc("/", us.proxyHandler)
			if err := http.ListenAndServeTLS("localhost:"+strconv.Itoa(serverPort), serverCert, serverKey, nil); err != nil {
				klog.Errorf("listen to http err: %v", err)
			}
		},
	}

	cmd.Flags().Int(FlagServerPort, 8080, "handle user request using this port")
	cmd.Flags().String(FlagProxyUds, ProxyUds, "the UDS name to connect to")
	cmd.Flags().String(FlagServerCert, "", "Secure communication with this cert.")
	cmd.Flags().String(FlagServerKey, "", "Secure communication with this key.")

	return cmd
}
