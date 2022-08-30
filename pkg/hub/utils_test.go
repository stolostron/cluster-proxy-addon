package hub

import (
	"fmt"
	"testing"
)

func TestParseKubeAPIServerRequestURL(t *testing.T) {
	testcases := []struct {
		requestURL  string
		targetHost  string
		kubeAPIPath string
		err         error
	}{
		{
			requestURL:  "route-domain/cluster1/api/pods?timeout=32s",
			targetHost:  "https://cluster1",
			kubeAPIPath: "api/pods",
		},
		{
			requestURL:  "route-domain/cluster1",
			targetHost:  "https://cluster1",
			kubeAPIPath: "api/pods",
			err:         fmt.Errorf("requestURL format not correct, path more than 2: route-domain/cluster1"),
		},
	}
	for _, tc := range testcases {
		targetHost, kubeAPIPath, err := parseKubeAPIServerRequestURL(tc.requestURL)
		if err != nil {
			if tc.err == nil {
				t.Fatalf("except no err, but got %v", err)
			}
			continue
		}
		if targetHost != tc.targetHost {
			t.Errorf("expected clusterID: %v, got: %v", tc.targetHost, targetHost)
		}
		if kubeAPIPath != tc.kubeAPIPath {
			t.Errorf("expected kubeAPIPath: %v, got: %v", tc.kubeAPIPath, kubeAPIPath)
		}
	}
}

func TestIsProxyService(t *testing.T) {
	testcases := []struct {
		requestURL string
		isProxy    bool
	}{
		{
			requestURL: "route-domain/cluster1/api/pods?timeout=32s",
			isProxy:    false,
		},
		{
			requestURL: "route-domain/cluster1/api/v1/namespaces/default/services/https:nginx:80/proxy-service/hello",
			isProxy:    true,
		},
	}

	for _, tc := range testcases {
		isProxy := isProxyService(tc.requestURL)
		if isProxy != tc.isProxy {
			t.Errorf("expected isProxy: %v, got: %v", tc.isProxy, isProxy)
		}
	}
}

func TestParseServiceRequestURL(t *testing.T) {
	testcases := []struct {
		requestURL  string
		targetHost  string
		servicePath string
		err         error
	}{
		{
			requestURL:  "route-domain/cluster1/api/v1/namespaces/default/services/http:nginx:80/proxy-service/hello?timeout=32s",
			targetHost:  "http://cluster-proxy-61c2553c2384ff09d319698cacb5fbc664b50a6a4015d88c2:80",
			servicePath: "hello",
		},
		{
			requestURL:  "route-domain/cluster1/api/v1/namespaces/default/services/https:nginx:443/proxy-service",
			targetHost:  "https://cluster-proxy-61c2553c2384ff09d319698cacb5fbc664b50a6a4015d88c2:443",
			servicePath: "",
		},
	}

	for _, tc := range testcases {
		targetHost, serviceURL, err := parseServiceRequestURL(tc.requestURL)
		if err != nil {
			if tc.err == nil {
				t.Fatalf("except no err, but got %v", err)
			}
			continue
		}
		if targetHost != tc.targetHost {
			t.Errorf("expected clusterID: %v, got: %v", tc.targetHost, targetHost)
		}
		if serviceURL != tc.servicePath {
			t.Errorf("expected kubeAPIPath: %v, got: %v", tc.servicePath, serviceURL)
		}
	}
}
