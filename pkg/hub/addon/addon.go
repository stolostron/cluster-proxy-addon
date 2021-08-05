package addon

import (
	"context"
	"fmt"
	"github.com/openshift/library-go/pkg/assets"
	"github.com/openshift/library-go/pkg/operator/resource/resourceapply"
	"github.com/pkg/errors"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/rand"
	corev1informers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"open-cluster-management.io/addon-framework/pkg/agent"
	"open-cluster-management.io/api/addon/v1alpha1"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
	bindata "open-cluster-management.io/cluster-proxy-addon/deploy/managedcluster/bindata"
	"open-cluster-management.io/cluster-proxy-addon/pkg/certificate"
	"open-cluster-management.io/cluster-proxy-addon/pkg/config"
	"os"
	"path/filepath"
	"strings"

	"github.com/openshift/library-go/pkg/operator/events"

	certificatesv1 "k8s.io/api/certificates/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var (
	genericScheme = runtime.NewScheme()
	genericCodecs = serializer.NewCodecFactory(genericScheme)
	genericCodec  = genericCodecs.UniversalDeserializer()
)

type clusterProxyAddOnAgent struct {
	secretInformer corev1informers.SecretInformer
	recorder       events.Recorder
	hubKubeClient  *kubernetes.Clientset
	agentName      string

	proxyServerAddress string
	proxyServerPort    int
}

// NewClusterProxyAddOnAgent returns an instance of clusterProxyAddOnAgent
func NewClusterProxyAddOnAgent(secretInformer corev1informers.SecretInformer, hubKubeClient *kubernetes.Clientset, recorder events.Recorder) *clusterProxyAddOnAgent {
	return &clusterProxyAddOnAgent{
		secretInformer: secretInformer,
		recorder:       recorder,
		agentName:      rand.String(5),
	}
}

// Manifests generates manifestworks to deploy the clusternet-proxy-addon agent on the managed cluster
func (a *clusterProxyAddOnAgent) Manifests(cluster *clusterv1.ManagedCluster, addon *v1alpha1.ManagedClusterAddOn) ([]runtime.Object, error) {
	installNamespace := addon.Spec.InstallNamespace
	if len(installNamespace) == 0 {
		installNamespace = "default"
	}

	image := os.Getenv("IMAGE_NAME")
	if len(image) == 0 {
		image = config.DefaultImage
	}

	host, port := a.getProxyServerAddress()

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
	}{
		// addon
		KubeConfigSecret:      fmt.Sprintf("%s-hub-kubeconfig", addon.Name),
		AddonInstallNamespace: installNamespace,
		ClusterName:           cluster.Name,
		Image:                 image,

		// anp
		APIServerProxyPort: config.APISERVER_PROXY_PORT,
		ProxyServerHost:    host,
		ProxyServerPort:    port,
	}

	files, err := bindata.AssetDir(config.AGENT_MANIFEST_FILES_DIR)
	if err != nil {
		return nil, err
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
		AddonName: config.ADDON_NAME,
		Registration: &agent.RegistrationOption{
			CSRConfigurations: a.csrConfigurations,
			CSRApproveCheck:   a.csrApproveCheck,
			PermissionConfig:  a.permissionConfig,
			CSRSign:           a.sign,
		},
	}
}

func (a *clusterProxyAddOnAgent) csrApproveCheck(
	cluster *clusterv1.ManagedCluster,
	addon *v1alpha1.ManagedClusterAddOn,
	csr *certificatesv1.CertificateSigningRequest) bool {
	if strings.HasPrefix(csr.Spec.Username, "system:open-cluster-management:"+cluster.Name) {
		klog.Info("CSR approved")
		return true
	} else {
		klog.Info("CSR not approved due to illegal requester", "requester", csr.Spec.Username)
		return false
	}
}

func (a *clusterProxyAddOnAgent) permissionConfig(cluster *clusterv1.ManagedCluster, addon *v1alpha1.ManagedClusterAddOn) error {
	// need a role with rbac role to access hub
	var err error

	roleName := fmt.Sprintf("%s:%s", config.Organization, config.ADDON)
	_, _, err = resourceapply.ApplyRole(context.TODO(), a.hubKubeClient.RbacV1(), a.recorder, &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      roleName,
			Namespace: config.DEFAULT_NAMESPACE,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{
					"",
				},
				Resources: []string{
					"configmaps",
				},
				Verbs: []string{
					"get", "list", "watch",
				},
			},
		},
	})
	if err != nil {
		return errors.Wrap(err, "create Role in permissionConfig failed")
	}

	roleBindingName := fmt.Sprintf("%s:%s", config.Organization, config.ADDON)
	_, _, err = resourceapply.ApplyRoleBinding(context.TODO(), a.hubKubeClient.RbacV1(), a.recorder, &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      roleBindingName,
			Namespace: config.DEFAULT_NAMESPACE,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:     "Group",
				APIGroup: "rbac.authorization.k8s.io",
				Name:     agent.DefaultGroups(cluster.Name, config.ADDON)[0],
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     roleName,
		},
	})
	if err != nil {
		return errors.Wrap(err, "create RoleBinding in permission failed")
	}

	return nil
}

func (a *clusterProxyAddOnAgent) csrConfigurations(cluster *clusterv1.ManagedCluster) []v1alpha1.RegistrationConfig {
	return []v1alpha1.RegistrationConfig{
		{
			SignerName: config.SIGNER_NAME,
			Subject: v1alpha1.Subject{
				User:   agent.DefaultUser(cluster.Name, config.ADDON, a.agentName),
				Groups: agent.DefaultGroups(cluster.Name, config.ADDON),
			},
		},
	}
}

func (a *clusterProxyAddOnAgent) sign(csr *certificatesv1.CertificateSigningRequest) []byte {
	caCert, caKey, err := certificate.GetCert(a.secretInformer.Lister(), config.SignerSecret, config.DEFAULT_NAMESPACE)
	if err != nil {
		klog.ErrorS(err, "addon csr sign failed", "csr name", csr.Name)
		return []byte{}
	}
	return certificate.SignCSR(csr, caCert, caKey)
}

func (a *clusterProxyAddOnAgent) getProxyServerAddress() (host string, port int) {
	// TODO the proority should be: evn_var -> route_on_hub -> service_on_hub -> default_value
	// right now for test sake, we use default_local_value
	return "host.docker.internal", 8091
}
