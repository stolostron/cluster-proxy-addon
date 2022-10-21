package serviceproxy

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httputil"
	"time"

	"github.com/spf13/cobra"
	"github.com/stolostron/cluster-proxy-addon/pkg/constant"
	"github.com/stolostron/cluster-proxy-addon/pkg/utils"
	"k8s.io/klog/v2"
	addonutils "open-cluster-management.io/addon-framework/pkg/utils"
)

func NewServiceProxyCommand() *cobra.Command {
	serviceProxyServer := newServiceProxy()

	cmd := &cobra.Command{
		Use:   "service-proxy",
		Short: "service-proxy",
		Long:  `A http proxy server, receives http requests from proxy-agent and forwards to the target service.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return serviceProxyServer.Run(cmd.Context())
		},
	}

	serviceProxyServer.AddFlags(cmd)
	return cmd
}

type serviceProxy struct {
	cert, key    string
	apiserverCA  string
	ocpserviceCA string
	rootCAs      *x509.CertPool

	maxIdleConns          int
	idleConnTimeout       time.Duration
	tLSHandshakeTimeout   time.Duration
	expectContinueTimeout time.Duration
}

func newServiceProxy() *serviceProxy {
	return &serviceProxy{}
}

func (s *serviceProxy) AddFlags(cmd *cobra.Command) {
	flags := cmd.Flags()

	flags.StringVar(&s.cert, "cert", s.cert, "The path to the certificate of the service proxy server")
	flags.StringVar(&s.key, "key", s.key, "The path to the key of the service proxy server")
	flags.StringVar(&s.apiserverCA, "apiserver-ca", s.apiserverCA, "The path to the CA certificate of the apiserver")
	flags.StringVar(&s.ocpserviceCA, "ocpservice-ca", s.ocpserviceCA, "The path to the CA certificate of the ocp services")

	// proxy related flags
	flags.IntVar(&s.maxIdleConns, "max-idle-conns", 100, "The maximum number of idle (keep-alive) connections across all hosts.")
	flags.DurationVar(&s.idleConnTimeout, "idle-conn-timeout", 90*time.Second, "The maximum amount of time an idle (keep-alive) connection will remain idle before closing itself.")
	flags.DurationVar(&s.tLSHandshakeTimeout, "tls-handshake-timeout", 10*time.Second, "The maximum amount of time waiting to wait for a TLS handshake.")
	flags.DurationVar(&s.expectContinueTimeout, "expect-continue-timeout", 1*time.Second, "The amount of time to wait for a server's first response headers after fully writing the request headers if the request has an \"Expect: 100-continue\" header.")
}

func (s *serviceProxy) Run(ctx context.Context) error {
	var err error

	if err := s.validate(); err != nil {
		return err
	}

	// add configchecker into http probes
	cc, err := addonutils.NewConfigChecker("http-service-proxy", s.cert, s.key, s.apiserverCA, s.ocpserviceCA)
	if err != nil {
		klog.Fatal(err)
	}

	go func() {
		if err = utils.ServeHealthProbes(":8000", cc.Check); err != nil {
			klog.Fatal(err)
		}
	}()

	// get root CAs
	s.rootCAs, err = getRootCAs(s.apiserverCA, s.ocpserviceCA)
	if err != nil {
		klog.Errorf("failed to get root ca: %v", err)
		return err
	}

	http.HandleFunc("/", s.handler)
	return http.ListenAndServeTLS(fmt.Sprintf(":%d", constant.ServiceProxyPort), s.cert, s.key, nil)
}

func (s *serviceProxy) handler(wr http.ResponseWriter, req *http.Request) {
	if klog.V(4).Enabled() {
		dump, err := httputil.DumpRequest(req, true)
		if err != nil {
			http.Error(wr, err.Error(), http.StatusBadRequest)
			return
		}
		klog.V(4).Infof("request:\n %s", string(dump))
	}

	url, err := utils.GetTargetServiceURLFromRequest(req)
	if err != nil {
		http.Error(wr, err.Error(), http.StatusBadRequest)
		klog.Errorf("failed to get target service url from request: %v", err)
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(url)
	proxy.Transport = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          s.maxIdleConns,
		IdleConnTimeout:       s.idleConnTimeout,
		TLSHandshakeTimeout:   s.tLSHandshakeTimeout,
		ExpectContinueTimeout: s.expectContinueTimeout,
		// skip server-auth of kube-apiserver
		// TODO use server-auth
		TLSClientConfig: &tls.Config{
			RootCAs:    s.rootCAs,
			MinVersion: tls.VersionTLS12,
		},
		// golang http pkg automaticly upgrade http connection to http2 connection, but http2 can not upgrade to SPDY which used in "kubectl exec".
		// set ForceAttemptHTTP2 = false to prevent auto http2 upgration
		ForceAttemptHTTP2: false,
	}

	proxy.ServeHTTP(wr, req)
}

func (s *serviceProxy) validate() error {
	if s.cert == "" {
		return fmt.Errorf("cert is required")
	}
	if s.key == "" {
		return fmt.Errorf("key is required")
	}
	if s.apiserverCA == "" {
		return fmt.Errorf("apiserver-ca-cert is required")
	}
	if s.ocpserviceCA == "" {
		return fmt.Errorf("services-ca-cert is required")
	}
	return nil
}

func getRootCAs(apiserverCACert, servicesCACert string) (*x509.CertPool, error) {
	rootCa := x509.NewCertPool()

	pem, err := ioutil.ReadFile(apiserverCACert)
	if err != nil {
		return nil, err
	}
	rootCa.AppendCertsFromPEM(pem)

	pem, err = ioutil.ReadFile(servicesCACert)
	if err != nil {
		return nil, err
	}
	rootCa.AppendCertsFromPEM(pem)

	return rootCa, nil
}
