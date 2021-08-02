package main

import (
	"crypto/tls"
	goflag "flag"
	"fmt"
	"github.com/open-cluster-management/cluster-proxy-addon/pkg/config"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	utilflag "k8s.io/component-base/cli/flag"
	"k8s.io/component-base/logs"
	"k8s.io/klog/v2"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
)

const (
	KUBE_APISERVER_ADDRESS = "https://kubernetes.default.svc"
)

func proxyHandler(wr http.ResponseWriter, req *http.Request) {
	apiserverURL, err := url.Parse(KUBE_APISERVER_ADDRESS)
	if err != nil {
		klog.Errorf("KUBE_APISERVER_ADDRESS parse error: %s", err.Error())
		return
	}

	klog.V(4).InfoS("requestURL", req.RequestURI)

	if klog.V(4).Enabled() {
		for k, v := range req.Header {
			klog.InfoS("Header:", k, v)
		}
	}

	// change the proto from http to https
	req.Proto = "https"

	// skip insecure verify
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	http.DefaultTransport.(*http.Transport).ForceAttemptHTTP2 = false
	if !http.DefaultTransport.(*http.Transport).ForceAttemptHTTP2 && http.DefaultTransport.(*http.Transport).TLSClientConfig != nil {
		klog.V(4).InfoS("not upgrade to http2")
	}

	proxy := httputil.NewSingleHostReverseProxy(apiserverURL)

	if req.Header.Get("Connection") == "Upgrade" && req.Header.Get("Upgrade") == "SPDY/3.1" {
		klog.V(4).InfoS("upgrade to spdy/3.1")
	}

	proxy.ServeHTTP(wr, req)
}

func main() {
	pflag.CommandLine.SetNormalizeFunc(utilflag.WordSepNormalizeFunc)
	pflag.CommandLine.AddGoFlagSet(goflag.CommandLine)

	logs.InitLogs()
	defer logs.FlushLogs()

	cmd := &cobra.Command{
		Use:   "apiserver-proxy",
		Short: "apiserver-proxy",
		Run: func(cmd *cobra.Command, args []string) {
			http.HandleFunc("/", proxyHandler)
			if err := http.ListenAndServe(":"+strconv.Itoa(config.APISERVER_PROXY_PORT), nil); err != nil {
				klog.Errorf("listen to http err: %s", err.Error())
			}
		},
	}

	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
