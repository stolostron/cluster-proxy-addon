package e2e

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	addonclient "open-cluster-management.io/api/client/addon/clientset/versioned"
	clusterclient "open-cluster-management.io/api/client/cluster/clientset/versioned"
)

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "E2E suite")
}

var (
	managedClusterName                                                                    string
	kubeClient, clusterProxyKubeClient, clusterProxyWrongClient, clusterProxyUnAuthClient kubernetes.Interface
	clusterProxyHttpClient                                                                *http.Client
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
	managedClusterInstallNamespace = "open-cluster-management-agent-addon"
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
		managedClusterName = "loopback"
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

	checkAddonStatus()

	prepareTestServiceAccount()

	preparePodFortest()

	prepareClusterProxyClient()
})

func checkAddonStatus() {
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
				fmt.Printf("\n[ERROR] Failed to get deployment %s: %v\n", deployment, err)
				return err
			}

			// Print detailed deployment status
			fmt.Printf("\n========== Deployment: %s ==========\n", deployment)
			fmt.Printf("Desired Replicas: %d\n", *d.Spec.Replicas)
			fmt.Printf("Current Replicas: %d\n", d.Status.Replicas)
			fmt.Printf("Updated Replicas: %d\n", d.Status.UpdatedReplicas)
			fmt.Printf("Ready Replicas: %d\n", d.Status.ReadyReplicas)
			fmt.Printf("Available Replicas: %d\n", d.Status.AvailableReplicas)
			fmt.Printf("Unavailable Replicas: %d\n", d.Status.UnavailableReplicas)

			// Print deployment conditions
			if len(d.Status.Conditions) > 0 {
				fmt.Printf("\nDeployment Conditions:\n")
				for _, cond := range d.Status.Conditions {
					fmt.Printf("  - Type: %s, Status: %s, Reason: %s, Message: %s\n",
						cond.Type, cond.Status, cond.Reason, cond.Message)
				}
			}

			// Print full deployment status as JSON for complete details
			statusJSON, _ := json.MarshalIndent(d.Status, "", "  ")
			fmt.Printf("\nFull Deployment Status JSON:\n%s\n", string(statusJSON))

			if d.Status.AvailableReplicas < 1 {
				// Get pods for this deployment
				fmt.Printf("\n========== Pods for deployment %s ==========\n", deployment)
				labelSelector := fmt.Sprintf("app=%s", deployment)
				pods, err := kubeClient.CoreV1().Pods(hubInstallNamespace).List(context.Background(), metav1.ListOptions{
					LabelSelector: labelSelector,
				})
				if err != nil {
					fmt.Printf("[ERROR] Failed to list pods with selector %s: %v\n", labelSelector, err)
				} else {
					fmt.Printf("Found %d pods with label selector: %s\n", len(pods.Items), labelSelector)
					for i, pod := range pods.Items {
						fmt.Printf("\n--- Pod %d: %s ---\n", i+1, pod.Name)
						fmt.Printf("Phase: %s\n", pod.Status.Phase)
						fmt.Printf("Node: %s\n", pod.Spec.NodeName)

						// Print container statuses
						fmt.Printf("\nContainer Statuses:\n")
						for _, containerStatus := range pod.Status.ContainerStatuses {
							fmt.Printf("  Container: %s\n", containerStatus.Name)
							fmt.Printf("    Image: %s\n", containerStatus.Image)
							fmt.Printf("    Ready: %v\n", containerStatus.Ready)
							fmt.Printf("    RestartCount: %d\n", containerStatus.RestartCount)

							if containerStatus.State.Waiting != nil {
								fmt.Printf("    State: Waiting - Reason: %s, Message: %s\n",
									containerStatus.State.Waiting.Reason, containerStatus.State.Waiting.Message)
							} else if containerStatus.State.Running != nil {
								fmt.Printf("    State: Running (started at %v)\n", containerStatus.State.Running.StartedAt)
							} else if containerStatus.State.Terminated != nil {
								fmt.Printf("    State: Terminated - Reason: %s, ExitCode: %d, Message: %s\n",
									containerStatus.State.Terminated.Reason, containerStatus.State.Terminated.ExitCode,
									containerStatus.State.Terminated.Message)
							}

							if containerStatus.LastTerminationState.Terminated != nil {
								fmt.Printf("    Last Termination: Reason: %s, ExitCode: %d, Message: %s\n",
									containerStatus.LastTerminationState.Terminated.Reason,
									containerStatus.LastTerminationState.Terminated.ExitCode,
									containerStatus.LastTerminationState.Terminated.Message)
							}
						}

						// Print pod conditions
						if len(pod.Status.Conditions) > 0 {
							fmt.Printf("\nPod Conditions:\n")
							for _, cond := range pod.Status.Conditions {
								fmt.Printf("  - Type: %s, Status: %s, Reason: %s, Message: %s\n",
									cond.Type, cond.Status, cond.Reason, cond.Message)
							}
						}

						// Print pod events
						events, err := kubeClient.CoreV1().Events(hubInstallNamespace).List(context.Background(), metav1.ListOptions{
							FieldSelector: fmt.Sprintf("involvedObject.name=%s", pod.Name),
						})
						if err == nil && len(events.Items) > 0 {
							fmt.Printf("\nRecent Events:\n")
							for _, event := range events.Items {
								fmt.Printf("  - %s: %s (Reason: %s, Count: %d)\n",
									event.Type, event.Message, event.Reason, event.Count)
							}
						}
					}
				}

				return fmt.Errorf("available replicas for %s should >= 1, but get %d", deployment, d.Status.AvailableReplicas)
			}
			fmt.Printf("========================================\n\n")
		}

		// service on hub exist
		_, err = kubeClient.CoreV1().Services(hubInstallNamespace).Get(context.Background(), "cluster-proxy-addon-user", metav1.GetOptions{})
		if err != nil {
			fmt.Printf("\n[ERROR] Failed to get service cluster-proxy-addon-user: %v\n", err)
			return err
		}

		// deployment on managedcluster is running
		anpAgent, err := kubeClient.AppsV1().Deployments(managedClusterInstallNamespace).Get(context.Background(), "cluster-proxy-proxy-agent", metav1.GetOptions{})
		if err != nil {
			fmt.Printf("\n[ERROR] Failed to get deployment cluster-proxy-proxy-agent: %v\n", err)
			return err
		}

		fmt.Printf("\n========== Deployment: cluster-proxy-proxy-agent (managed cluster) ==========\n")
		fmt.Printf("Available Replicas: %d\n", anpAgent.Status.AvailableReplicas)

		if anpAgent.Status.AvailableReplicas < 1 {
			statusJSON, _ := json.MarshalIndent(anpAgent.Status, "", "  ")
			fmt.Printf("Full Status:\n%s\n", string(statusJSON))
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
	if !apierrors.IsAlreadyExists(err) {
		Expect(err).To(BeNil())
	}

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
	if !apierrors.IsAlreadyExists(err) {
		Expect(err).To(BeNil())
	}

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
	if !apierrors.IsAlreadyExists(err) {
		Expect(err).To(BeNil())
	}
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

var (
	kubeconfig     string
	baseDomain     string
	userServerHost string
)

func prepareClusterProxyClient() {
	var err error
	kubeconfig = os.Getenv("KUBECONFIG")
	baseDomain = os.Getenv("CLUSTER_BASE_DOMAIN")
	userServerHost = "cluster-proxy-user." + baseDomain

	By("Get RootCA of the cluster")
	// get the ca is stored in configmap "kube-root-ca.crt" in the hubInstallNamespace.
	ca, err := kubeClient.CoreV1().ConfigMaps(hubInstallNamespace).Get(context.Background(), "kube-root-ca.crt", metav1.GetOptions{})
	Expect(err).To(BeNil())
	rootCA := ca.Data["ca.crt"]

	By("Creat secret token for serviceAccount")
	_, err = kubeClient.CoreV1().Secrets(hubInstallNamespace).Create(context.Background(), &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cluster-proxy-test-token",
			Namespace: hubInstallNamespace,
			Annotations: map[string]string{
				"kubernetes.io/service-account.name": serviceAccountName,
			},
		},
		Type: "kubernetes.io/service-account-token",
	}, metav1.CreateOptions{})
	Expect(err).To(BeNil())

	Eventually(func() error {
		tokenSecret, err := kubeClient.CoreV1().Secrets(hubInstallNamespace).Get(context.Background(), "cluster-proxy-test-token", metav1.GetOptions{})
		if err != nil {
			return err
		}
		token, ok := tokenSecret.Data["token"]
		if !ok {
			return fmt.Errorf("should containe token in secret %s", tokenSecret.Name)
		}
		serviceAccountToken = string(token)
		return nil
	}, eventuallyTimeout, eventuallyInterval).ShouldNot(HaveOccurred())

	By("Create kubeclient using cluster-proxy kubeconfig and http client to access specified services")
	err = func() error {
		var err error
		// create good client
		clusterProxyCfg, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return err
		}

		// Add rootCA to the clusterProxyCfg
		clusterProxyCfg.TLSClientConfig.CAData = []byte(rootCA)
		clusterProxyCfg.TLSClientConfig.CertData = nil
		clusterProxyCfg.TLSClientConfig.KeyData = nil
		clusterProxyCfg.BearerToken = serviceAccountToken

		clusterProxyCfg.Host = fmt.Sprintf("https://%s/%s", userServerHost, managedClusterName)
		fmt.Println("host:", clusterProxyCfg.Host)

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
		clusterWrongProxyCfg.TLSClientConfig.CAData = []byte(rootCA)
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
		clusterUnAuthProxyCfg.TLSClientConfig.CAData = []byte(rootCA)
		clusterUnAuthProxyCfg.TLSClientConfig.CertData = nil
		clusterUnAuthProxyCfg.TLSClientConfig.KeyData = nil
		clusterUnAuthProxyCfg.BearerToken = serviceAccountToken + "wrong token"

		clusterProxyUnAuthClient, err = kubernetes.NewForConfig(clusterUnAuthProxyCfg)
		if err != nil {
			return err
		}

		// clusterProxyHttpClient
		rootCAPool := x509.NewCertPool()
		rootCAPool.AppendCertsFromPEM([]byte(rootCA))
		clusterProxyHttpClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs: rootCAPool,
				},
			},
		}

		return nil
	}()
	Expect(err).To(BeNil())
}
