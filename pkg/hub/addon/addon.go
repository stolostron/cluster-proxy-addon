package addon

import (
	"github.com/open-cluster-management/addon-framework/pkg/agent"
	addonapiv1alpha1 "github.com/open-cluster-management/api/addon/v1alpha1"
	clusterv1 "github.com/open-cluster-management/api/cluster/v1"

	"github.com/openshift/library-go/pkg/operator/events"

	certificatesv1 "k8s.io/api/certificates/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
)

const (
	addOnName = "cluster-proxy"
	agentName = "cluster-proxy-agent"
)

type clusterProxyAddOnAgent struct {
	kubeClient kubernetes.Interface
	recorder   events.Recorder
	agentImage string
}

// NewClusterProxyAddOnAgent returns an instance of clusterProxyAddOnAgent
func NewClusterProxyAddOnAgent(kubeClient kubernetes.Interface, recorder events.Recorder, agentImage string) *clusterProxyAddOnAgent {
	return &clusterProxyAddOnAgent{
		kubeClient: kubeClient,
		recorder:   recorder,
		agentImage: agentImage,
	}
}

// Manifests generates manifestworks to deploy the clusternet-proxy-addon agent on the managed cluster
func (a *clusterProxyAddOnAgent) Manifests(
	cluster *clusterv1.ManagedCluster, addon *addonapiv1alpha1.ManagedClusterAddOn) ([]runtime.Object, error) {
	//TODO
	return nil, nil
}

// GetAgentAddonOptions returns the options of cluster-proxy-addon agent
func (a *clusterProxyAddOnAgent) GetAgentAddonOptions() agent.AgentAddonOptions {
	return agent.AgentAddonOptions{
		AddonName: addOnName,
		Registration: &agent.RegistrationOption{
			CSRConfigurations: agent.KubeClientSignerConfigurations(addOnName, agentName),
			CSRApproveCheck:   a.csrApproveCheck,
			PermissionConfig:  a.permissionConfig,
		},
	}
}

func (a *clusterProxyAddOnAgent) csrApproveCheck(
	cluster *clusterv1.ManagedCluster,
	addon *addonapiv1alpha1.ManagedClusterAddOn,
	csr *certificatesv1.CertificateSigningRequest) bool {
	//TODO
	return false
}

func (a *clusterProxyAddOnAgent) permissionConfig(
	cluster *clusterv1.ManagedCluster, addon *addonapiv1alpha1.ManagedClusterAddOn) error {
	//TODO
	return nil
}
