package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	goflag "flag"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	utilflag "k8s.io/component-base/cli/flag"
	"k8s.io/component-base/logs"
	"k8s.io/klog/v2"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"open-cluster-management.io/cluster-proxy-addon/pkg/config"
	"os"
	"os/signal"
	"strconv"
	"strings"
)

const (
	FlagServerPort = "server-port"
	FlagProxyUds   = "proxy-uds"

	FlagServerCert = "server-cert"
	FlagServerKey  = "server-key"
)

const (
	ClusterRequestProto = "http"
	ProxyUds            = "/go/src/open-cluster-management.io/api-network-proxy-addon/socket"
)

type UserServer struct {
	proxyUdsName string
}

func NewUserServer(proxyUdsName string) (*UserServer, error) {
	return &UserServer{
		proxyUdsName: proxyUdsName,
	}, nil
}

func (u *UserServer) proxyHandler(wr http.ResponseWriter, req *http.Request) {
	// parse clusterID from current requestURL
	clusterID, kubeAPIPath, err := parseRequestURL(req.RequestURI)
	if err != nil {
		klog.ErrorS(err, "parse request URL failed")
		return
	}

	// connect with http tunnel
	o := &options{
		mode:         "http-connect",
		proxyUdsName: u.proxyUdsName,
		requestProto: ClusterRequestProto,
		requestHost:  clusterID,
		requestPort:  config.APISERVER_PROXY_PORT,
		requestPath:  kubeAPIPath,
	}

	// skip insecure verify
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	// replace dialer with tunnel dialer
	dialer, err := getUDSDialer(o)
	if err != nil {
		klog.ErrorS(err, "get dialer failed")
		return
	}
	http.DefaultTransport.(*http.Transport).DialContext = dialer
	http.DefaultTransport.(*http.Transport).ForceAttemptHTTP2 = false

	// restruct new apiserverURL
	target := fmt.Sprintf("%s://%s:%d", o.requestProto, o.requestHost, o.requestPort)
	apiserverURL, err := url.Parse(target)
	if err != nil {
		klog.ErrorS(err, "parse restructed URL")
		return
	}

	// update request URL path
	req.URL.Path = o.requestPath

	// update proti
	req.Proto = "http"

	klog.V(4).InfoS("request:", "scheme", req.URL.Scheme, "rawQuery", req.URL.RawQuery, "path", req.URL.Path)

	proxy := httputil.NewSingleHostReverseProxy(apiserverURL)
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
		err = errors.New("requestURL format not correct")
		return
	}
	clusterID = paths[1]                             // <clusterID>
	kubeAPIPath = strings.Join(paths[2:], "/")       // api/pods?timeout=32s
	kubeAPIPath = strings.Split(kubeAPIPath, "?")[0] // api/pods
	return
}

// options is copy from apiserver-network-proxy/cmd/client/main.go GrpcProxyClientOptions
type options struct {
	requestProto string
	requestPath  string
	requestHost  string
	requestPort  int
	proxyUdsName string
	mode         string
}

func getUDSDialer(o *options) (func(ctx context.Context, network, addr string) (net.Conn, error), error) {
	var proxyConn net.Conn
	var err error

	// Setup signal handler
	ch := make(chan os.Signal, 1)
	signal.Notify(ch)
	go func() {
		for {
			sig := <-ch
			if strings.Contains(sig.String(), "urgent I/O") {
				klog.V(4).InfoS("listen Urgent I/O but not close the connection")
				continue
			} else {
				klog.InfoS("Signal close connection", "sig", sig.String())
				if proxyConn == nil {
					klog.InfoS("connection already closed")
				} else if proxyConn != nil {
					err := proxyConn.Close()
					klog.ErrorS(err, "connection closed")
				}
				return
			}
		}
	}()

	requestAddress := fmt.Sprintf("%s:%d", o.requestHost, o.requestPort)

	proxyConn, err = net.Dial("unix", o.proxyUdsName)
	if err != nil {
		return nil, fmt.Errorf("dialing proxy %q failed: %v", o.proxyUdsName, err)
	}
	fmt.Fprintf(proxyConn, "CONNECT %s HTTP/1.1\r\nHost: %s\r\nUser-Agent: %s\r\n\r\n", requestAddress, "127.0.0.1")
	br := bufio.NewReader(proxyConn)
	res, err := http.ReadResponse(br, nil)
	if err != nil {
		return nil, fmt.Errorf("reading HTTP response from CONNECT to %s via uds proxy %s failed: %v",
			requestAddress, o.proxyUdsName, err)
	}
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("proxy error from %s while dialing %s: %v", o.proxyUdsName, requestAddress, res.Status)
	}

	// It's safe to discard the bufio.Reader here and return the
	// original TCP conn directly because we only use this for
	// TLS, and in TLS the client speaks first, so we know there's
	// no unbuffered data. But we can double-check.
	if br.Buffered() > 0 {
		return nil, fmt.Errorf("unexpected %d bytes of buffered data from CONNECT uds proxy %q",
			br.Buffered(), o.proxyUdsName)
	}

	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		return proxyConn, nil
	}, nil
}

func main() {
	pflag.CommandLine.SetNormalizeFunc(utilflag.WordSepNormalizeFunc)
	pflag.CommandLine.AddGoFlagSet(goflag.CommandLine)

	logs.InitLogs()
	defer logs.FlushLogs()

	cmd := &cobra.Command{
		Use:   "anp-user-server",
		Short: "anp-user-server",
		Run: func(cmd *cobra.Command, args []string) {
			serverPort, _ := cmd.Flags().GetInt(FlagServerPort)
			proxyUds, _ := cmd.Flags().GetString(FlagProxyUds)
			serverCert, _ := cmd.Flags().GetString(FlagServerCert)
			serverKey, _ := cmd.Flags().GetString(FlagServerKey)

			us, err := NewUserServer(proxyUds)
			if err != nil {
				klog.ErrorS(err, "new user server failed")
				return
			}

			http.HandleFunc("/", us.proxyHandler)
			if err := http.ListenAndServeTLS("localhost:"+strconv.Itoa(serverPort), serverCert, serverKey, nil); err != nil {
				klog.ErrorS(err, "listen to http err")
			}
		},
	}

	cmd.Flags().Int(FlagServerPort, 8080, "handle user request using this port")
	cmd.Flags().String(FlagProxyUds, ProxyUds, "the UDS name to connect to")
	cmd.Flags().String(FlagServerCert, "", "Secure communication with this cert.")
	cmd.Flags().String(FlagServerKey, "", "Secure communication with this key.")

	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
