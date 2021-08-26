package e2e

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	ginkgo "github.com/onsi/ginkgo"
	gomega "github.com/onsi/gomega"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/retry"

	certificatesv1 "k8s.io/api/certificates/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	addonv1alpha1 "open-cluster-management.io/api/addon/v1alpha1"
	addonclient "open-cluster-management.io/api/client/addon/clientset/versioned"
	clusterclient "open-cluster-management.io/api/client/cluster/clientset/versioned"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
)

func TestE2E(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "E2E suite")
}

var (
	managedClusterName                                          string
	userServerHost                                              string
	kubeClient, clusterProxyKubeClient, clusterProxyWrongClient kubernetes.Interface
	hubAddOnClient                                              addonclient.Interface
	hubClusterClient                                            clusterclient.Interface
	clusterCfg                                                  *rest.Config
	serviceAccountToken                                         string
)

const (
	eventuallyTimeout              = 300 // seconds
	eventuallyInterval             = 6   // seconds
	hubInstallNamespace            = "open-cluster-namespace"
	managedClusterInstallNamespace = "open-cluster-namespace-agent-addon"
	addonName                      = "cluster-proxy"
)

// This suite is sensitive to the following environment variables:
//
// - MANAGED_CLUSTER_NAME sets the name of the cluster
// - KUBECONFIG is the location of the kubeconfig file to use
//
// Note: in this test, hub and managedcluster should be one same host
var _ = ginkgo.BeforeSuite(func() {
	ginkgo.By("Approve CSR from managed cluster")
	kubeconfig := os.Getenv("KUBECONFIG")
	managedClusterName = os.Getenv("MANAGED_CLUSTER_NAME")
	if managedClusterName == "" {
		managedClusterName = "cluster1"
	}
	userServerHost = os.Getenv("USER_SERVER_HOST")
	gomega.Expect(userServerHost).ToNot(gomega.Equal(""))

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
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

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
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
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

			csr.Status.Conditions = append(csr.Status.Conditions, certificatesv1.CertificateSigningRequestCondition{
				Type:    certificatesv1.CertificateApproved,
				Status:  corev1.ConditionTrue,
				Reason:  "Approved by E2E",
				Message: "Approved as part of Loopback e2e",
			})
			_, err := kubeClient.CertificatesV1().CertificateSigningRequests().UpdateApproval(context.TODO(), csr.Name, csr, metav1.UpdateOptions{})
			return err
		})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
	}

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
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
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
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	// Except hub and managedcluster kubeclients are working
	ginkgo.By("Except hub and managedcluster kubeclients are working")
	gomega.Eventually(func() error {
		_, err := hubClusterClient.ClusterV1().ManagedClusters().Get(context.Background(), managedClusterName, metav1.GetOptions{})
		if err != nil {
			return err
		}

		_, err = kubeClient.CoreV1().Namespaces().Get(context.Background(), managedClusterName, metav1.GetOptions{})
		if err != nil {
			return err
		}
		return nil
	}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

	// Create a serviceaccount on managedcluster
	ginkgo.By("Create a serviceaccount on managedcluster")
	sa, err := kubeClient.CoreV1().ServiceAccounts(hubInstallNamespace).Create(context.Background(), &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "podServiceAccount",
			Namespace: hubInstallNamespace,
		},
	}, metav1.CreateOptions{})
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	// Get SecretToken for serviceAccount
	ginkgo.By("Get SecretToken for serviceAccount")
	secret, err := kubeClient.CoreV1().Secrets(sa.Secrets[0].Namespace).Get(context.Background(), sa.Secrets[0].Name, metav1.GetOptions{})
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	tokenEncoded, ok := secret.Data["token"]
	gomega.Expect(ok).To(gomega.Equal(true))
	token, err := base64.StdEncoding.DecodeString(string(tokenEncoded))
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	serviceAccountToken = string(token)

	// Create a role only can access pods under ns open-cluster-management
	ginkgo.By("Create a role only can access pods under ns open-cluster-management")
	_, err = kubeClient.RbacV1().Roles(hubInstallNamespace).Create(context.Background(), &v1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "podRole",
			Namespace: hubInstallNamespace,
		},
		Rules: []v1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"pods"},
				Verbs:     []string{"get", "list", "watch", "create", "delete", "update", "patch"},
			},
		},
	}, metav1.CreateOptions{})
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	// Create a rolebinding for this service account, the serviceaccount can only
	ginkgo.By("Create a rolebinding for this service account, the serviceaccount can only")
	_, err = kubeClient.RbacV1().RoleBindings(hubInstallNamespace).Create(context.Background(), &v1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "podRoleBinding",
			Namespace: hubInstallNamespace,
		},
		RoleRef: v1.RoleRef{},
		Subjects: []v1.Subject{
			{
				Kind: v1.ServiceAccountKind,
			},
		},
	}, metav1.CreateOptions{})
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	// Create managedcluster namespace for install addon agent
	ginkgo.By("Create namespace on managed cluster")
	_, err = kubeClient.CoreV1().Namespaces().Create(context.Background(), &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: managedClusterName,
		},
	}, metav1.CreateOptions{})
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	// Create an addon on hub
	ginkgo.By("Create addon on hub")
	addon := &addonv1alpha1.ManagedClusterAddOn{
		ObjectMeta: metav1.ObjectMeta{
			Name:      addonName,
			Namespace: managedClusterName,
		},
		Spec: addonv1alpha1.ManagedClusterAddOnSpec{
			InstallNamespace: managedClusterInstallNamespace,
		},
	}
	_, err = hubAddOnClient.AddonV1alpha1().ManagedClusterAddOns(managedClusterName).Create(context.Background(), addon, metav1.CreateOptions{})
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	// Check all resources(configmaps, secrets, deployment) are deployed and working
	ginkgo.By("Check all resources are working")
	gomega.Eventually(func() error {
		// deployment on hub is running
		anpServer, err := kubeClient.AppsV1().Deployments(hubInstallNamespace).Get(context.Background(), "cluster-proxy-addon-anp-server", metav1.GetOptions{})
		if err != nil {
			return err
		}

		if anpServer.Status.AvailableReplicas != 1 {
			return fmt.Errorf("available replicas for %s should be 1", "anp-server")
		}

		controller, err := kubeClient.AppsV1().Deployments(hubInstallNamespace).Get(context.Background(), "cluster-proxy-addon-controller", metav1.GetOptions{})
		if err != nil {
			return err
		}
		if controller.Status.AvailableReplicas != 1 {
			return fmt.Errorf("available replicas for %s should be 1", "controller")
		}

		// service on hub exist
		_, err = kubeClient.CoreV1().Services(hubInstallNamespace).Get(context.Background(), "cluster-proxy-addon-user", metav1.GetOptions{})
		if err != nil {
			return err
		}

		// deployment on managedcluster is running
		anpAgent, err := kubeClient.AppsV1().Deployments(managedClusterName).Get(context.Background(), "anp-agent", metav1.GetOptions{})
		if err != nil {
			return err
		}
		if anpAgent.Status.AvailableReplicas != 1 {
			return fmt.Errorf("available replicas for %s should be 1", "anp-agent")
		}

		return nil
	}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

	// Create cluster-proxy
	// the proxy-config should defined before test
	err = func() error {
		clusterCfg, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return err
		}

		// delete certificate and add token
		clusterCfg.TLSClientConfig.CertData = nil
		clusterCfg.TLSClientConfig.KeyData = nil
		clusterCfg.BearerToken = serviceAccountToken

		// change Host to the right host
		clusterCfg.Host = fmt.Sprintf("https://%s/%s", userServerHost, managedClusterName)

		// create cluster-proxy client
		clusterProxyKubeClient, err = kubernetes.NewForConfig(clusterCfg)
		if err != nil {
			return err
		}

		// change Host to the wrong host
		clusterCfg.Host = fmt.Sprintf("https://%s/%s", userServerHost, "wrongclust")

		// create wrong cluster-proxy client
		clusterProxyWrongClient, err = kubernetes.NewForConfig(clusterCfg)
		if err != nil {
			return err
		}
		return nil
	}()
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
})
