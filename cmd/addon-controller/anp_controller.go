package main

import (
	"context"
	"github.com/openshift/library-go/pkg/controller/controllercmd"
	"github.com/openshift/library-go/pkg/operator/events"
	"github.com/spf13/cobra"
	certificatesv1 "k8s.io/api/certificates/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilrand "k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	"open-cluster-management.io/addon-framework/pkg/addonmanager"
	"open-cluster-management.io/addon-framework/pkg/agent"
	"open-cluster-management.io/api/addon/v1alpha1"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
	"xuezhaojun/api-proxy-network-addon/pkg/certificate"
	"xuezhaojun/api-proxy-network-addon/pkg/version"
)

var (
	genericScheme = runtime.NewScheme()
	genericCodecs = serializer.NewCodecFactory(genericScheme)
	genericCodec  = genericCodecs.UniversalDeserializer()
)

const (
	SIGNER_NAME = "open-cluster-managment.io/anp-addon"
	ADDON_NAME  = "anp-addon"
)

func init() {
	scheme.AddToScheme(genericScheme)
}

// controller running on hub
func newControllerCommand() *cobra.Command {
	cmd := controllercmd.
		NewControllerCommandConfig("addon-controller", version.Get(), runController).
		NewCommand()
	cmd.Use = "controller"
	cmd.Short = "Start the addon controller"

	return cmd
}

func runController(ctx context.Context, controllerContext *controllercmd.ControllerContext) error {
	// anp-agent
	mgr, err := addonmanager.New(controllerContext.KubeConfig)
	if err != nil {
		return err
	}
	agentRegistration := &anpAddonAgent{
		kubeConfig: controllerContext.KubeConfig,
		recorder:   controllerContext.EventRecorder,
		agentName:  utilrand.String(6),
	}
	mgr.AddAgent(agentRegistration)
	mgr.Start(ctx)

	// anp-cert-controller
	hubKubeClient, err := kubernetes.NewForConfig(controllerContext.KubeConfig)
	if err != nil {
		return nil
	}
	c := certificate.NewCertController(certificate.ANP, hubKubeClient, controllerContext.EventRecorder)
	go c.Run(ctx, 1)

	<-ctx.Done()
	return nil
}

type anpAddonAgent struct {
	kubeConfig *rest.Config
	recorder   events.Recorder
	agentName  string
}

func (a *anpAddonAgent) Manifests(cluster *clusterv1.ManagedCluster, addon *v1alpha1.ManagedClusterAddOn) ([]runtime.Object, error) {
	objects := []runtime.Object{}
	// TODO Currnetly no manifest work need to deploy
	return objects, nil
}

func (a *anpAddonAgent) GetAgentAddonOptions() agent.AgentAddonOptions {
	return agent.AgentAddonOptions{
		AddonName: "anp-addon",
		Registration: &agent.RegistrationOption{
			CSRConfigurations: a.csrConfigurations,
			CSRApproveCheck: func(cluster *clusterv1.ManagedCluster, addon *v1alpha1.ManagedClusterAddOn, csr *certificatesv1.CertificateSigningRequest) bool {
				if strings.HasPrefix(csr.Spec.Username, "system:open-cluster-management:"+cluster.Name) {
					klog.Info("CSR approved")
					return true
				} else {
					klog.Info("CSR not approved due to illegal requester", "requester", csr.Spec.Username)
					return false
				}
			},
			PermissionConfig: func(cluster *clusterv1.ManagedCluster, addon *v1alpha1.ManagedClusterAddOn) error {
				return nil
			},
			CSRSign: func(csr *certificatesv1.CertificateSigningRequest) []byte {
				c, err := client.New(a.kubeConfig, client.Options{})
				if err != nil {
					klog.ErrorS(err, "create kube client failed when sign csr")
					return nil
				}
				caCert, caKey, _, err := certificate.GetCertFromSecret(context.TODO(), c, certificate.ANP_CA, certificate.ANP_NAMESPACE)
				if err != nil {
					klog.ErrorS(err, "get cert from secret failed when sign csr")
					return nil
				}
				return certificate.SignCSR(csr, caCert, caKey)
			},
		},
	}
}

func (a *anpAddonAgent) csrConfigurations(cluster *clusterv1.ManagedCluster) []v1alpha1.RegistrationConfig {
	return []v1alpha1.RegistrationConfig{
		{
			SignerName: SIGNER_NAME,
			Subject: v1alpha1.Subject{
				User:   agent.DefaultUser(cluster.Name, ADDON_NAME, a.agentName),
				Groups: agent.DefaultGroups(cluster.Name, ADDON_NAME),
			},
		},
	}
}
