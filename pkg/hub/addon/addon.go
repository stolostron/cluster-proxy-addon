package addon

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/open-cluster-management/cluster-proxy-addon/pkg/config"
	"github.com/open-cluster-management/cluster-proxy-addon/pkg/helpers"
	bindata "github.com/open-cluster-management/cluster-proxy-addon/pkg/hub/addon/bindata"
	"github.com/openshift/library-go/pkg/assets"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2"
	"open-cluster-management.io/addon-framework/pkg/agent"
	"open-cluster-management.io/api/addon/v1alpha1"
	clusterv1 "open-cluster-management.io/api/cluster/v1"

	"github.com/openshift/library-go/pkg/operator/events"

	certificatesv1 "k8s.io/api/certificates/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var (
	genericScheme = runtime.NewScheme()
	genericCodecs = serializer.NewCodecFactory(genericScheme)
	genericCodec  = genericCodecs.UniversalDeserializer()
)

const (
	addOnGroup         = "system:open-cluster-management:addon:cluster-proxy"
	agentUserName      = "system:open-cluster-management:cluster:%s:addon:cluster-proxy:agent:cluster-proxy-addon-agent"
	clusterAddOnGroup  = "system:open-cluster-management:cluster:%s:addon:cluster-proxy"
	authenticatedGroup = "system:authenticated"
)

func init() {
	scheme.AddToScheme(genericScheme)
}

type clusterProxyAddOnAgent struct {
	recorder      events.Recorder
	hubKubeClient kubernetes.Interface
	agentImage    string
	anpPublicHost string
	anpPublicPort int
}

// NewClusterProxyAddOnAgent returns an instance of clusterProxyAddOnAgent
func NewClusterProxyAddOnAgent(hubKubeClient *kubernetes.Clientset, recorder events.Recorder, agentImage, anpPublicHost string, anpPublicPort int) *clusterProxyAddOnAgent {
	return &clusterProxyAddOnAgent{
		recorder:      recorder,
		hubKubeClient: hubKubeClient,
		agentImage:    agentImage,
		anpPublicHost: anpPublicHost,
		anpPublicPort: anpPublicPort,
	}
}

// Manifests generates manifestworks to deploy the clusternet-proxy-addon agent on the managed cluster
func (a *clusterProxyAddOnAgent) Manifests(cluster *clusterv1.ManagedCluster, addon *v1alpha1.ManagedClusterAddOn) ([]runtime.Object, error) {
	installNamespace := addon.Spec.InstallNamespace
	if len(installNamespace) == 0 {
		installNamespace = config.AGENT_NAMESPACE
	}

	// TODO add trigger in addon-framework later.
	caConfigMap, err := a.hubKubeClient.CoreV1().ConfigMaps(config.HUB_NAMESPACE).Get(context.TODO(), config.CaBundleConfigmap, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	caBundleCrt, ok := caConfigMap.Data[config.CaBundleConfigmapDataKey]
	if !ok {
		return nil, fmt.Errorf("no data in ca-bundle-crt configmap: %v", caConfigMap)
	}

	host, port := a.anpPublicHost, a.anpPublicPort

	mainfestConfig := struct {
		// addon
		KubeConfigSecret      string
		ClusterName           string
		AddonInstallNamespace string
		Image                 string

		// anp
		APIServerProxyPort int
		ProxyServerHost    string
		ProxyServerPort    int
		CABundleCrt        []byte
	}{
		// addon
		KubeConfigSecret:      fmt.Sprintf("%s-hub-kubeconfig", addon.Name),
		AddonInstallNamespace: installNamespace,
		ClusterName:           cluster.Name,
		Image:                 a.agentImage,

		// anp
		APIServerProxyPort: config.APISERVER_PROXY_PORT,
		ProxyServerHost:    host,
		ProxyServerPort:    port,
		CABundleCrt:        []byte(caBundleCrt),
	}

	files := config.AgentFiles
	if installNamespace != config.AGENT_NAMESPACE {
		files = append(files, config.AgentNamespaceFile)
	}

	objects := []runtime.Object{}
	for _, file := range files {
		raw := assets.MustCreateAssetFromTemplate(file, bindata.MustAsset(filepath.Join(config.AGENT_MANIFEST_FILES_DIR, file)), &mainfestConfig).Data
		object, _, err := genericCodec.Decode(raw, nil, nil)
		if err != nil {
			return nil, err
		}
		objects = append(objects, object)
	}

	return objects, nil
}

// GetAgentAddonOptions returns the options of cluster-proxy-addon agent
func (a *clusterProxyAddOnAgent) GetAgentAddonOptions() agent.AgentAddonOptions {
	return agent.AgentAddonOptions{
		AddonName: config.ADDON,
		Registration: &agent.RegistrationOption{
			CSRConfigurations: a.csrConfigurations,
			CSRApproveCheck:   a.csrApproveCheck,
			PermissionConfig: func(cluster *clusterv1.ManagedCluster, addon *v1alpha1.ManagedClusterAddOn) error {
				return nil
			},
			CSRSign: a.sign,
		},
	}
}

// To check the addon agent csr, we check
// 1. signer name in csr request is valid.
// 2. if organization field and commonName field in csr request is valid.
// 3. if user name in csr is the same as commonName field in csr request.
func (a *clusterProxyAddOnAgent) csrApproveCheck(cluster *clusterv1.ManagedCluster, addon *v1alpha1.ManagedClusterAddOn, csr *certificatesv1.CertificateSigningRequest) bool {
	// check signer
	if csr.Spec.SignerName != config.SIGNER_NAME && csr.Spec.SignerName != certificatesv1.KubeAPIServerClientSignerName {
		klog.Infof("CSR Approve Check Falied signerName not right: signerName: %s", csr.Name, csr.Spec.SignerName)
		return false
	}

	// check org field and commonName field
	block, _ := pem.Decode(csr.Spec.Request)
	if block == nil || block.Type != "CERTIFICATE REQUEST" {
		klog.Infof("CSR Approve Check Falied csr %q was not recognized: PEM block type is not CERTIFICATE REQUEST", csr.Name)
		return false
	}

	x509cr, err := x509.ParseCertificateRequest(block.Bytes)
	if err != nil {
		klog.Infof("CSR Approve Check Falied csr %q was not recognized: %v", csr.Name, err)
		return false
	}

	requestingOrgs := sets.NewString(x509cr.Subject.Organization...)
	if requestingOrgs.Len() != 3 {
		klog.Infof("CSR Approve Check Falied csr %q org is not equal to 3", csr.Name)
		return false
	}

	if !requestingOrgs.Has(authenticatedGroup) {
		klog.Infof("CSR Approve Check Falied csr requesting orgs doesn't contain %s", authenticatedGroup)
		return false
	}

	if !requestingOrgs.Has(addOnGroup) {
		klog.Infof("CSR Approve Check Falied csr requesting orgs doesn't contain %s", addOnGroup)
		return false
	}

	if !requestingOrgs.Has(fmt.Sprintf(clusterAddOnGroup, cluster.Name)) {
		klog.Infof("CSR Approve Check Falied csr requesting orgs doesn't contain %s", fmt.Sprintf(clusterAddOnGroup, cluster.Name))
		return false
	}

	// check commonName field
	if fmt.Sprintf(agentUserName, cluster.Name) != x509cr.Subject.CommonName {
		klog.Infof("CSR Approve Check Falied commonName not right; request %s get %s", x509cr.Subject.CommonName, fmt.Sprintf(agentUserName, cluster.Name))
		return false
	}

	// check user name
	if strings.HasPrefix(csr.Spec.Username, "system:open-cluster-management:"+cluster.Name) {
		klog.Info("CSR approved")
		return true
	} else {
		klog.Info("CSR not approved due to illegal requester", "requester", csr.Spec.Username)
		return false
	}
}

func (a *clusterProxyAddOnAgent) csrConfigurations(cluster *clusterv1.ManagedCluster) []v1alpha1.RegistrationConfig {
	return append(agent.KubeClientSignerConfigurations(config.ADDON, config.ADDON_AGENT_NAME)(cluster), v1alpha1.RegistrationConfig{
		SignerName: config.SIGNER_NAME,
		Subject: v1alpha1.Subject{
			User:   agent.DefaultUser(cluster.Name, config.ADDON, config.ADDON_AGENT_NAME),
			Groups: agent.DefaultGroups(cluster.Name, config.ADDON),
		},
	})
}

func (a *clusterProxyAddOnAgent) sign(csr *certificatesv1.CertificateSigningRequest) []byte {
	// We consider approved csr no need to do verify again
	klog.Infof("CSRSign, agent: %s", config.ADDON_AGENT_NAME)
	s, err := a.hubKubeClient.CoreV1().Secrets(config.HUB_NAMESPACE).Get(context.TODO(), config.SignerSecret, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("sign csr failed: %v", err)
		return []byte{}
	}
	caCert, caKey, err := helpers.GetCert(s)
	if err != nil {
		klog.Errorf("addon csr sign failed, csr name: %s; err: %v", csr.Name, err)
		return []byte{}
	}
	certificate, err := helpers.SignCSR(csr, caCert, caKey)
	if err != nil {
		klog.Errorf("sign csr failed: %v", err)
		return []byte{}
	}
	return certificate
}
