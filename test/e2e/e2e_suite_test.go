package e2e

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	certificatesv1 "k8s.io/api/certificates/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/retry"

	addonclient "open-cluster-management.io/api/client/addon/clientset/versioned"
	clusterclient "open-cluster-management.io/api/client/cluster/clientset/versioned"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
)

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "E2E suite")
}

var (
	managedClusterName                                                                    string
	kubeClient, clusterProxyKubeClient, clusterProxyWrongClient, clusterProxyUnAuthClient kubernetes.Interface
	hubAddOnClient                                                                        addonclient.Interface
	hubClusterClient                                                                      clusterclient.Interface
	clusterCfg                                                                            *rest.Config
	clusterProxyCfg                                                                       *rest.Config
	serviceAccountToken                                                                   string
	podName                                                                               string
)

const (
	eventuallyTimeout              = 600 // seconds
	eventuallyInterval             = 30  // seconds
	hubInstallNamespace            = "open-cluster-management-addon"
	managedClusterInstallNamespace = "open-cluster-management-cluster-proxy"
	serviceAccountName             = "cluster-proxy-test"
)

// This suite is sensitive to the following environment variables:
//
// - MANAGED_CLUSTER_NAME sets the name of the cluster
// - KUBECONFIG is the location of the kubeconfig file to use
//
// Note: in this test, hub and managedcluster should be one same host
var _ = BeforeSuite(func() {
	kubeconfig := os.Getenv("KUBECONFIG")
	managedClusterName = os.Getenv("MANAGED_CLUSTER_NAME")
	if managedClusterName == "" {
		managedClusterName = "cluster1"
	}

	By("Init clients")
	err := func() error {
		var err error
		clusterCfg, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return err
		}

		kubeClient, err = kubernetes.NewForConfig(clusterCfg)
		if err != nil {
			return err
		}

		hubAddOnClient, err = addonclient.NewForConfig(clusterCfg)
		if err != nil {
			return err
		}

		hubClusterClient, err = clusterclient.NewForConfig(clusterCfg)

		return err
	}()
	Expect(err).To(BeNil())

	prepareOCM()

	prepareAddon()

	prepareTestServiceAccount()

	preparePodFortest()

	prepareClusterProxyClient()
})

func prepareOCM() {
	var err error
	By("Approve CSR from managed cluster")
	var csrs *certificatesv1.CertificateSigningRequestList
	// Waiting for the CSR for ManagedCluster to exist
	err = wait.Poll(1*time.Second, 120*time.Second, func() (bool, error) {
		var err error
		csrs, err = kubeClient.CertificatesV1().CertificateSigningRequests().List(context.TODO(), metav1.ListOptions{
			LabelSelector: fmt.Sprintf("open-cluster-management.io/cluster-name = %v", managedClusterName),
		})
		if err != nil {
			return false, err
		}

		if len(csrs.Items) >= 1 {
			return true, nil
		}

		return false, nil
	})
	Expect(err).To(BeNil())

	// Approving all pending CSRs
	for i := range csrs.Items {
		csr := &csrs.Items[i]
		if !strings.HasPrefix(csr.Name, managedClusterName) {
			continue
		}

		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			csr, err = kubeClient.CertificatesV1().CertificateSigningRequests().Get(context.TODO(), csr.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}

			for _, c := range csr.Status.Conditions {
				if c.Type == certificatesv1.CertificateApproved {
					return nil
				}
			}

			csr.Status.Conditions = append(csr.Status.Conditions, certificatesv1.CertificateSigningRequestCondition{
				Type:    certificatesv1.CertificateApproved,
				Status:  corev1.ConditionTrue,
				Reason:  "Approved by E2E",
				Message: "Approved as part of Loopback e2e",
			})
			_, err := kubeClient.CertificatesV1().CertificateSigningRequests().UpdateApproval(context.TODO(), csr.Name, csr, metav1.UpdateOptions{})
			return err
		})
		Expect(err).To(BeNil())
	}

	By("Accepting ManagedCluster")
	var managedCluster *clusterv1.ManagedCluster
	// Waiting for ManagedCluster to exist
	err = wait.Poll(1*time.Second, 120*time.Second, func() (bool, error) {
		var err error
		managedCluster, err = hubClusterClient.ClusterV1().ManagedClusters().Get(context.TODO(), managedClusterName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		return true, nil
	})
	Expect(err).To(BeNil())

	// Accepting ManagedCluster
	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var err error
		managedCluster, err = hubClusterClient.ClusterV1().ManagedClusters().Get(context.TODO(), managedCluster.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		managedCluster.Spec.HubAcceptsClient = true
		managedCluster.Spec.LeaseDurationSeconds = 5
		_, err = hubClusterClient.ClusterV1().ManagedClusters().Update(context.TODO(), managedCluster, metav1.UpdateOptions{})
		return err
	})
	Expect(err).To(BeNil())

	By("Except hub and managedcluster kubeclients are working")
	// Except hub and managedcluster kubeclients are working
	Eventually(func() error {
		_, err := hubClusterClient.ClusterV1().ManagedClusters().Get(context.Background(), managedClusterName, metav1.GetOptions{})
		if err != nil {
			return err
		}

		_, err = kubeClient.CoreV1().Namespaces().Get(context.Background(), managedClusterName, metav1.GetOptions{})
		if err != nil {
			return err
		}
		return nil
	}, eventuallyTimeout, eventuallyInterval).ShouldNot(HaveOccurred())
}

func prepareAddon() {
	var err error

	By("Check resources are running")
	Eventually(func() error {
		// deployments on hub is running
		deployments := []string{
			"cluster-proxy-addon-manager",
			"cluster-proxy-addon-user",
			"cluster-proxy",
		}
		for _, deployment := range deployments {
			d, err := kubeClient.AppsV1().Deployments(hubInstallNamespace).Get(context.Background(), deployment, metav1.GetOptions{})
			if err != nil {
				return err
			}
			if d.Status.AvailableReplicas != 1 {
				return fmt.Errorf("available replicas for %s should be 1", deployment)
			}
		}

		// service on hub exist
		_, err = kubeClient.CoreV1().Services(hubInstallNamespace).Get(context.Background(), "cluster-proxy-addon-user", metav1.GetOptions{})
		if err != nil {
			return err
		}

		// deployment on managedcluster is running
		anpAgent, err := kubeClient.AppsV1().Deployments(managedClusterInstallNamespace).Get(context.Background(), "cluster-proxy-proxy-agent", metav1.GetOptions{})
		if err != nil {
			return err
		}
		if anpAgent.Status.AvailableReplicas < 1 {
			return fmt.Errorf("available replicas for %s should be more than 1, but get %d", "anp-agent", anpAgent.Status.AvailableReplicas)
		}

		return nil
	}, eventuallyTimeout, eventuallyInterval).ShouldNot(HaveOccurred())
}

func prepareTestServiceAccount() {
	By("Create a serviceaccount on managedcluster")
	_, err := kubeClient.CoreV1().ServiceAccounts(hubInstallNamespace).Create(context.Background(), &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceAccountName,
			Namespace: hubInstallNamespace,
		},
	}, metav1.CreateOptions{})
	Expect(err).To(BeNil())

	By("Create a role")
	_, err = kubeClient.RbacV1().Roles(hubInstallNamespace).Create(context.Background(), &v1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "podrole",
			Namespace: hubInstallNamespace,
		},
		Rules: []v1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"pods", "pods/log"},
				Verbs:     []string{"get", "list"},
			}, {
				APIGroups: []string{""},
				Resources: []string{"pods/exec"},
				Verbs:     []string{"create"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"configmaps"},
				Verbs:     []string{"watch"},
			},
		},
	}, metav1.CreateOptions{})
	Expect(err).To(BeNil())

	By("Create a rolebinding")
	_, err = kubeClient.RbacV1().RoleBindings(hubInstallNamespace).Create(context.Background(), &v1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "podrolebinding",
			Namespace: hubInstallNamespace,
		},
		RoleRef: v1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     "podrole",
		},
		Subjects: []v1.Subject{
			{
				Kind: v1.ServiceAccountKind,
				Name: "cluster-proxy-test",
			},
		},
	}, metav1.CreateOptions{})
	Expect(err).To(BeNil())
}

func preparePodFortest() {
	pods, err := kubeClient.CoreV1().Pods(hubInstallNamespace).List(context.Background(), metav1.ListOptions{})
	Expect(err).To(BeNil())
	for _, pod := range pods.Items {
		if !strings.Contains(pod.Name, "cluster-proxy-addon-manager") {
			continue
		}
		podName = pod.Name
	}
}

func prepareClusterProxyClient() {
	var err error
	kubeconfig := os.Getenv("KUBECONFIG")
	baseDomain := os.Getenv("CLUSTER_BASE_DOMAIN")
	userServerHost := "cluster-proxy-user." + baseDomain

	By("Get secret token for serviceAccount")
	sa, err := kubeClient.CoreV1().ServiceAccounts(hubInstallNamespace).Get(context.Background(), serviceAccountName, metav1.GetOptions{})
	Expect(err).To(BeNil())

	for _, sec := range sa.Secrets {
		if !strings.Contains(sec.Name, "token") {
			continue
		}
		secret, err := kubeClient.CoreV1().Secrets(hubInstallNamespace).Get(context.Background(), sec.Name, metav1.GetOptions{})
		Expect(err).To(BeNil())
		token, ok := secret.Data["token"]
		Expect(ok).To(Equal(true))
		serviceAccountToken = string(token)
		break
	}

	By("Create kubeclient using cluster-proxy kubeconfig")
	err = func() error {
		var err error
		// create good client
		clusterProxyCfg, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return err
		}

		clusterProxyCfg.TLSClientConfig.CertData = nil
		clusterProxyCfg.TLSClientConfig.KeyData = nil
		clusterProxyCfg.BearerToken = serviceAccountToken

		clusterProxyCfg.Host = fmt.Sprintf("https://%s/%s", userServerHost, managedClusterName)

		clusterProxyKubeClient, err = kubernetes.NewForConfig(clusterProxyCfg)
		if err != nil {
			return err
		}

		// change Host to the wrong host
		clusterWrongProxyCfg, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return err
		}
		clusterWrongProxyCfg.Host = fmt.Sprintf("https://%s/%s", userServerHost, "wrongcluster")
		clusterWrongProxyCfg.TLSClientConfig.CertData = nil
		clusterWrongProxyCfg.TLSClientConfig.KeyData = nil
		clusterWrongProxyCfg.BearerToken = serviceAccountToken

		clusterProxyWrongClient, err = kubernetes.NewForConfig(clusterWrongProxyCfg)
		if err != nil {
			return err
		}

		// create unauth proxy client
		clusterUnAuthProxyCfg, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return err
		}

		clusterUnAuthProxyCfg.Host = fmt.Sprintf("https://%s/%s", userServerHost, managedClusterName)
		clusterUnAuthProxyCfg.TLSClientConfig.CertData = nil
		clusterUnAuthProxyCfg.TLSClientConfig.KeyData = nil
		clusterUnAuthProxyCfg.BearerToken = serviceAccountToken + "wrong token"

		clusterProxyUnAuthClient, err = kubernetes.NewForConfig(clusterUnAuthProxyCfg)
		if err != nil {
			return err
		}

		return nil
	}()
	Expect(err).To(BeNil())
}

var _ = AfterSuite(func() {
	err := kubeClient.CoreV1().ConfigMaps(hubInstallNamespace).Delete(context.Background(), "cluster-proxy-test", metav1.DeleteOptions{})
	Expect(err).To(BeNil())
})
