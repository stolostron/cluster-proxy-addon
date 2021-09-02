package e2e

import (
	"testing"

	ginkgo "github.com/onsi/ginkgo"
	gomega "github.com/onsi/gomega"
)

func TestE2E(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "E2E suite")
}

// This suite is sensitive to the following environment variables:
//
// - MANAGED_CLUSTER_NAME sets the name of the cluster
// - KUBECONFIG is the location of the kubeconfig file to use
//
// Note: in this test, hub and managedcluster should be one same host
var _ = ginkgo.BeforeSuite(func() {
	// TODO add logic
})
