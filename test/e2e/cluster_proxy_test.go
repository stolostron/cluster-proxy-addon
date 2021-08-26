package e2e

import (
	"context"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/remotecommand"
)

var _ = ginkgo.Describe("Request through Cluster-Proxy", func() {
	ginkgo.Describe("Get pods", func() {
		ginkgo.Context("URL is invalid", func() {
			ginkgo.It("shoudl return error msg", func() {
				_, err := clusterProxyWrongClient.CoreV1().Pods(hubInstallNamespace).List(context.Background(), v1.ListOptions{})
				gomega.Expect(err).ToNot(gomega.BeNil())
			})
		})

		ginkgo.Context("URL is vailid", func() {
			ginkgo.It("should return pods information", func() {
				_, err := clusterProxyKubeClient.CoreV1().Pods(hubInstallNamespace).List(context.Background(), v1.ListOptions{})
				gomega.Expect(err).ToNot(gomega.BeNil())
			})
		})

		ginkgo.Context("URL is valid, but out of namepsace open-cluster-management", func() {
			ginkgo.It("should return forbidden", func() {
				_, err := clusterProxyKubeClient.CoreV1().Pods(managedClusterInstallNamespace).List(context.Background(), v1.ListOptions{})
				gomega.Expect(err).ToNot(gomega.BeNil())
			})
		})
	})

	ginkgo.Describe("Execute", func() {
		ginkgo.It("should return helloworld", func() {
			req := clusterProxyKubeClient.CoreV1().RESTClient().Post().Resource("pods").Name("cluster-proxy-addon").Namespace(hubInstallNamespace).SubResource("exec")
			scheme := runtime.NewScheme()
			err := corev1.AddToScheme(scheme)
			gomega.Expect(err).To(gomega.BeNil())

			parameterCodec := runtime.NewParameterCodec(scheme)
			req.VersionedParams(&corev1.PodExecOptions{
				Command:   []string{"sh", "-c", "echo hello"},
				Container: "proxy-server",
				Stdin:     true,
				// Stdin:     stdin != nil,
				Stdout: true,
				Stderr: true,
				TTY:    false,
			}, parameterCodec)

			exec, err := remotecommand.NewSPDYExecutor("", "POST", req.URL()) // the config should be cluster-proxy config
			gomega.Expect(err).To(gomega.BeNil())

			err = exec.Stream(remotecommand.StreamOptions{
				Stdin:  stdin,
				Stdout: &stdout,
				Stderr: &stderr,
				Tty:    false,
			})
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(stdout.String()).To(gomega.Equal("hello"))
		})
	})

	ginkgo.Describe("Log", func() {
		ginkgo.It("shoudl return logs information", func() {
			req := clusterProxyKubeClient.CoreV1().Pods(hubInstallNamespace).GetLogs("cluster-proxy-addon", &corev1.PodLogOptions{})
			podlogs, err := req.Stream(context.Background())
			gomega.Expect(err).To(gomega.BeNil())
			podlogs.Close()
		})
	})

	ginkgo.Describe("Watch", func() {
		ginkgo.It("shoud watch", func() {
			watch, err := clusterProxyKubeClient.CoreV1().Pods(hubInstallNamespace).Watch(context.TODO(), v1.ListOptions{})
			gomega.Expect(err).To(gomega.BeNil())

			// create a pod
			_, err = clusterProxyKubeClient.CoreV1().Pods(hubInstallNamespace).Create(context.Background(), &corev1.Pod{}, v1.CreateOptions{})
			gomega.Expect(err).To(gomega.BeNil())

			// check if r is create
			select {
			case <-watch.ResultChan():
				// this chan shoud not receive any pod event before pod created
			default:
				ginkgo.Fail("Failed to received a pod create event")
			}
		})
	})
})
