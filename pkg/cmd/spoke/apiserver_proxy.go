package spoke

import (
	"crypto/tls"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"time"

	"github.com/open-cluster-management/cluster-proxy-addon/pkg/config"
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
)

const (
	KUBE_APISERVER_ADDRESS = "https://kubernetes.default.svc"
)

func proxyHandler(wr http.ResponseWriter, req *http.Request) {
	apiserverURL, err := url.Parse(KUBE_APISERVER_ADDRESS)
	if err != nil {
		klog.Errorf("KUBE_APISERVER_ADDRESS parse error: %s", err.Error())
		http.Error(wr, err.Error(), http.StatusBadRequest)
		return
	}

	dump, err := httputil.DumpRequest(req, true)
	if err != nil {
		klog.Errorf("KUBE_APISERVER_ADDRESS parse error: %s", err.Error())
		http.Error(wr, err.Error(), http.StatusBadRequest)
		return
	}
	klog.V(4).Infof("request:\n %s", string(dump))

	// change the proto from http to https
	req.Proto = "https"

	proxy := httputil.NewSingleHostReverseProxy(apiserverURL)
	proxy.Transport = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		// skip server-auth of kube-apiserver
		// TODO use server-auth
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		// golang http pkg automaticly upgrade http connection to http2 connection, but http2 can not upgrade to SPDY which used in "kubectl exec".
		// set ForceAttemptHTTP2 = false to prevent auto http2 upgration
		ForceAttemptHTTP2: false,
	}

	proxy.ServeHTTP(wr, req)
}

func NewAPIServerProxy() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apiserver-proxy",
		Short: "Start a apiserver-proxy",
		Run: func(cmd *cobra.Command, args []string) {
			http.HandleFunc("/", proxyHandler)
			if err := http.ListenAndServe(":"+strconv.Itoa(config.APISERVER_PROXY_PORT), nil); err != nil {
				klog.Errorf("listen to http err: %s", err.Error())
			}
		},
	}
	return cmd
}
