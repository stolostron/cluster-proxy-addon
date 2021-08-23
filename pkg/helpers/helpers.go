package helpers

import (
	"fmt"
	"io/ioutil"
	"strings"
)

// GetCurrentNamespace returns the current namesapce from file system,
// if the namespace is not found, it returns the defaultNamespace
func GetCurrentNamespace(defaultNamespace string) string {
	nsBytes, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		return defaultNamespace
	}
	return string(nsBytes)
}

// ParseRequestURL
// Example Input: <service-ip>:8080/<clusterID>/api/pods?timeout=32s
// Example Output:
// 	clusterID: <clusterID>
// 	kubeAPIPath: api/pods
func ParseRequestURL(requestURL string) (clusterID string, kubeAPIPath string, err error) {
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
