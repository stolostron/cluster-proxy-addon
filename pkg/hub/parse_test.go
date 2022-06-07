package hub

import (
	"fmt"
	"testing"
)

func TestParseRequestURL(t *testing.T) {
	testcases := []struct {
		requestURL  string
		clusterID   string
		kubeAPIPath string
		err         error
	}{
		{
			requestURL:  "route-domain/cluster1/api/pods?timeout=32s",
			clusterID:   "cluster1",
			kubeAPIPath: "api/pods",
		},
		{
			requestURL:  "route-domain/cluster1",
			clusterID:   "cluster1",
			kubeAPIPath: "api/pods",
			err:         fmt.Errorf("requestURL format not correct, path more than 2: route-domain/cluster1"),
		},
	}
	for _, tc := range testcases {
		clusterID, kubeAPIPath, err := parseRequestURL(tc.requestURL)
		if err != nil {
			if tc.err == nil {
				t.Fatalf("except no err, but got %v", err)
			}
			continue
		}
		if clusterID != tc.clusterID {
			t.Errorf("expected clusterID: %v, got: %v", tc.clusterID, clusterID)
		}
		if kubeAPIPath != tc.kubeAPIPath {
			t.Errorf("expected kubeAPIPath: %v, got: %v", tc.kubeAPIPath, kubeAPIPath)
		}
	}
}
