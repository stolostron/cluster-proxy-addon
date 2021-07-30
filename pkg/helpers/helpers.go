package helpers

import "io/ioutil"

// GetCurrentNamespace returns the current namesapce from file system,
// if the namespace is not found, it returns the defaultNamespace
func GetCurrentNamespace(defaultNamespace string) string {
	nsBytes, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		return defaultNamespace
	}
	return string(nsBytes)
}
