package e2e

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
)

var _ = Describe("Requests through Cluster-Proxy", func() {
	Describe("Get pods", func() {
		Context("URL is vailid", func() {
			It("should return pods information", func() {
				_, err := clusterProxyKubeClient.CoreV1().Pods(hubInstallNamespace).List(context.Background(), v1.ListOptions{})
				Expect(err).To(BeNil())
			})
		})

		Context("URL is invalid", func() {
			It("shoudl return error msg", func() {
				_, err := clusterProxyWrongClient.CoreV1().Pods(hubInstallNamespace).List(context.Background(), v1.ListOptions{})
				Expect(err).ToNot(BeNil())
			})
		})

		Context("URL is valid, but out of namepsace open-cluster-management", func() {
			It("should return forbidden", func() {
				_, err := clusterProxyKubeClient.CoreV1().Pods(managedClusterInstallNamespace).List(context.Background(), v1.ListOptions{})
				Expect(err).ToNot(BeNil())
				Expect(errors.IsForbidden(err)).To(Equal(true))
			})
		})

		Context("URL is valid, but using unauth token", func() {
			It("should return unauth", func() {
				_, err := clusterProxyUnAuthClient.CoreV1().Pods(hubInstallNamespace).List(context.Background(), v1.ListOptions{})
				Expect(err).ToNot(BeNil())
				Expect(errors.IsUnauthorized(err)).To(Equal(true))
			})
		})
	})

	Describe("Get Logs of a pod", func() {
		It("should return logs information", func() {
			req := clusterProxyKubeClient.CoreV1().Pods(hubInstallNamespace).GetLogs(podName, &corev1.PodLogOptions{})
			podlogs, err := req.Stream(context.Background())
			Expect(err).To(BeNil())
			podlogs.Close()
		})
	})

	Describe("Watch ConfigMap create", func() {
		It("shoud watch", func() {
			watch, err := clusterProxyKubeClient.CoreV1().ConfigMaps(hubInstallNamespace).Watch(context.TODO(), v1.ListOptions{})
			Expect(err).To(BeNil())

			// create a pod
			_, err = kubeClient.CoreV1().ConfigMaps(hubInstallNamespace).Create(context.Background(), &corev1.ConfigMap{
				ObjectMeta: v1.ObjectMeta{
					Name: "cluster-proxy-test",
				},
			}, v1.CreateOptions{})
			Expect(err).To(BeNil())

			// check if r is create
			select {
			case <-watch.ResultChan():
				// this chan shoud not receive any pod event before pod created
				err := kubeClient.CoreV1().ConfigMaps(hubInstallNamespace).Delete(context.Background(), "cluster-proxy-test", metav1.DeleteOptions{})
				Expect(err).To(BeNil())
			default:
				Fail("Failed to received a pod create event")
			}
		})
	})

	Describe("Execute in a pod", func() {
		It("should return hello", func() {
			req := clusterProxyKubeClient.CoreV1().RESTClient().Post().Resource("pods").Name(podName).Namespace(hubInstallNamespace).SubResource("exec").Param("container", "manager")

			req.VersionedParams(&corev1.PodExecOptions{
				Command:   []string{"/bin/sh", "-c", "echo hello"},
				Container: "manager",
				Stdin:     false,
				Stdout:    true,
				Stderr:    true,
				TTY:       false,
			}, scheme.ParameterCodec)

			exec, err := remotecommand.NewSPDYExecutor(clusterProxyCfg, "POST", req.URL())
			Expect(err).To(BeNil())

			var stdout, stderr bytes.Buffer
			err = exec.StreamWithContext(context.Background(), remotecommand.StreamOptions{
				Stdin:  nil,
				Stdout: &stdout,
				Stderr: &stderr,
				Tty:    false,
			})
			Expect(err).To(BeNil())
			Expect(strings.Contains(stdout.String(), "hello")).To(Equal(true))
		})
	})

	Describe("Access Prometheus-k8s service", func() {
		It("should return metrics with http code 200", func() {
			targetHost := fmt.Sprintf(`https://%s/%s/api/v1/namespaces/openshift-monitoring/services/prometheus-k8s:9091/proxy-service/api/v1/query?query=machine_cpu_sockets`, userServerHost, managedClusterName)
			fmt.Println("The targetHost: ", targetHost)

			req, err := http.NewRequest("GET", targetHost, nil)
			Expect(err).To(BeNil())

			// Create secret token for serviceaccount openshift-monitoring/prometheus-k8s
			_, err = kubeClient.CoreV1().Secrets("openshift-monitoring").Create(context.Background(), &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "prometheus-k8s-token",
					Annotations: map[string]string{
						"kubernetes.io/service-account.name": "prometheus-k8s",
					},
				},
				Type: "kubernetes.io/service-account-token",
			}, metav1.CreateOptions{})
			Expect(err).To(BeNil())

			var prometheusk8sToken string
			Eventually(func() error {
				tokenSecret, err := kubeClient.CoreV1().Secrets("openshift-monitoring").Get(context.Background(), "prometheus-k8s-token", metav1.GetOptions{})
				if err != nil {
					return err
				}
				token, ok := tokenSecret.Data["token"]
				if !ok {
					return fmt.Errorf("should containe token in secret %s", tokenSecret.Name)
				}
				prometheusk8sToken = string(token)
				return nil
			}, eventuallyTimeout, eventuallyInterval).ShouldNot(HaveOccurred())

			// Add token to request header
			req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", prometheusk8sToken))

			resp, err := clusterProxyHttpClient.Do(req)
			Expect(err).To(BeNil())
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			Expect(err).To(BeNil())
			fmt.Println("response:", string(body))

			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			Expect(strings.Contains(string(body), "success")).To(Equal(true))
		})
	})
})
