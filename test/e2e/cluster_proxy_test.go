package e2e

import (
	"bytes"
	"context"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
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
			default:
				Fail("Failed to received a pod create event")
			}
		})
	})

	Describe("Execute in a pod", func() {
		It("should return hello", func() {
			req := clusterProxyKubeClient.CoreV1().RESTClient().Post().Resource("pods").Name(podName).Namespace(hubInstallNamespace).SubResource("exec").Param("container", "cluster-proxy")

			req.VersionedParams(&corev1.PodExecOptions{
				Command:   []string{"/bin/sh", "-c", "echo hello"},
				Container: "cluster-proxy",
				Stdin:     false,
				Stdout:    true,
				Stderr:    true,
				TTY:       false,
			}, scheme.ParameterCodec)

			exec, err := remotecommand.NewSPDYExecutor(clusterProxyCfg, "POST", req.URL())
			Expect(err).To(BeNil())

			var stdout, stderr bytes.Buffer
			err = exec.Stream(remotecommand.StreamOptions{
				Stdin:  nil,
				Stdout: &stdout,
				Stderr: &stderr,
				Tty:    false,
			})
			Expect(err).To(BeNil())
			Expect(strings.Contains(stdout.String(), "hello")).To(Equal(true))
		})
	})
})
