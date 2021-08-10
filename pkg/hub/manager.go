package hub

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/open-cluster-management/cluster-proxy-addon/pkg/helpers"
	"github.com/open-cluster-management/cluster-proxy-addon/pkg/hub/addon"
	"github.com/open-cluster-management/cluster-proxy-addon/pkg/hub/controllers"
	"open-cluster-management.io/addon-framework/pkg/addonmanager"

	"github.com/openshift/library-go/pkg/controller/controllercmd"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
)

const (
	containerName    = "cluster-proxy"
	defaultNamespace = "open-cluster-management"
)

type AddOnControllerOptions struct {
	AgentImage string
	Namespace  string
}

func NewAddOnControllerOptions() *AddOnControllerOptions {
	return &AddOnControllerOptions{}
}

func (o *AddOnControllerOptions) AddFlags(cmd *cobra.Command) {
	flags := cmd.Flags()
	//TODO if downstream building supports to set downstream image, we could use this flag
	// to set agent image on building phase
	flags.StringVar(&o.AgentImage, "agent-image", o.AgentImage, "The image of addon agent.")
}

func (o *AddOnControllerOptions) Complete(kubeClient kubernetes.Interface) error {
	if len(o.AgentImage) != 0 {
		return nil
	}

	o.Namespace = helpers.GetCurrentNamespace(defaultNamespace)
	podName := os.Getenv("POD_NAME")
	if len(podName) == 0 {
		return fmt.Errorf("The pod enviroment POD_NAME is required")
	}

	pod, err := kubeClient.CoreV1().Pods(o.Namespace).Get(context.TODO(), podName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	for _, container := range pod.Spec.Containers {
		if container.Name == containerName {
			o.AgentImage = pod.Spec.Containers[0].Image
			return nil
		}
	}
	return fmt.Errorf("The agent image cannot be found from the container %q of the pod %q", containerName, podName)
}

// RunControllerManager starts the controllers on hub to manage submariner deployment.
func (o *AddOnControllerOptions) RunControllerManager(ctx context.Context, controllerContext *controllercmd.ControllerContext) error {
	kubeClient, err := kubernetes.NewForConfig(controllerContext.KubeConfig)
	if err != nil {
		return err
	}

	if err := o.Complete(kubeClient); err != nil {
		return err
	}

	kubeInformer := informers.NewSharedInformerFactoryWithOptions(kubeClient, 5*time.Minute, informers.WithNamespace(o.Namespace))

	certRotationController := controllers.NewCertRotationController(
		o.Namespace,
		kubeClient,
		kubeInformer.Core().V1().Secrets(),
		kubeInformer.Core().V1().ConfigMaps(),
		controllerContext.EventRecorder,
	)

	go kubeInformer.Start(ctx.Done())
	go certRotationController.Run(ctx, 1)

	mgr, err := addonmanager.New(controllerContext.KubeConfig)
	if err != nil {
		return err
	}

	err = mgr.AddAgent(addon.NewClusterProxyAddOnAgent(kubeClient, controllerContext.EventRecorder, o.AgentImage))
	if err != nil {
		return err
	}

	go mgr.Start(ctx)

	<-ctx.Done()
	return nil
}
