package hub

import (
	"context"
	"open-cluster-management.io/addon-framework/pkg/addonmanager"
	"open-cluster-management.io/cluster-proxy-addon/pkg/config"
	"open-cluster-management.io/cluster-proxy-addon/pkg/hub/addon"
	"time"

	"github.com/spf13/cobra"

	"open-cluster-management.io/cluster-proxy-addon/pkg/helpers"
	"open-cluster-management.io/cluster-proxy-addon/pkg/hub/controllers"

	"github.com/openshift/library-go/pkg/controller/controllercmd"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
)

type AddOnControllerOptions struct {
	AgentImage string
	Namespace  string
}

func NewAddOnControllerOptions() *AddOnControllerOptions {
	return &AddOnControllerOptions{
		Namespace: helpers.GetCurrentNamespace(config.DEFAULT_NAMESPACE),
	}
}

func (o *AddOnControllerOptions) AddFlags(cmd *cobra.Command) {
	flags := cmd.Flags()
	flags.StringVar(&o.AgentImage, "agent-image", config.AGENT_IMAGE, "The image of addon agent.")
}

// RunControllerManager starts the controllers on hub to manage submariner deployment.
func (o *AddOnControllerOptions) RunControllerManager(ctx context.Context, controllerContext *controllercmd.ControllerContext) error {
	kubeClient, err := kubernetes.NewForConfig(controllerContext.KubeConfig)
	if err != nil {
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

	err = mgr.AddAgent(addon.NewClusterProxyAddOnAgent(kubeInformer.Core().V1().Secrets(), kubeClient, controllerContext.EventRecorder))
	if err != nil {
		return err
	}

	go mgr.Start(ctx)

	<-ctx.Done()
	return nil
}
