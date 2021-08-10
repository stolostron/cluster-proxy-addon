package spoke

import (
	"context"
	"errors"

	"github.com/spf13/cobra"

	"github.com/open-cluster-management/cluster-proxy-addon/pkg/helpers"
	"open-cluster-management.io/addon-framework/pkg/lease"

	"github.com/openshift/library-go/pkg/controller/controllercmd"

	"k8s.io/client-go/kubernetes"
)

const (
	addOnName                    = "cluster-proxy"
	defaultInstallationNamespace = "open-cluster-management-agent-addon"
)

type AgentOptions struct {
	InstallationNamespace string
	HubKubeconfigFile     string
	ClusterName           string
}

func NewAgentOptions() *AgentOptions {
	return &AgentOptions{}
}

func (o *AgentOptions) AddFlags(cmd *cobra.Command) {
	flags := cmd.Flags()
	flags.StringVar(&o.HubKubeconfigFile, "hub-kubeconfig", o.HubKubeconfigFile, "Location of kubeconfig file to connect to hub cluster.")
	flags.StringVar(&o.ClusterName, "cluster-name", o.ClusterName, "Name of managed cluster.")
}

func (o *AgentOptions) Complete() {
	o.InstallationNamespace = helpers.GetCurrentNamespace(defaultInstallationNamespace)
}

func (o *AgentOptions) Validate() error {
	if o.HubKubeconfigFile == "" {
		return errors.New("hub-kubeconfig is required")
	}

	if o.ClusterName == "" {
		return errors.New("cluster name is empty")
	}

	return nil
}

func (o *AgentOptions) RunAgent(ctx context.Context, controllerContext *controllercmd.ControllerContext) error {
	o.Complete()

	if err := o.Validate(); err != nil {
		return err
	}

	// start lease updater
	spokeKubeClient, err := kubernetes.NewForConfig(controllerContext.KubeConfig)
	if err != nil {
		return err
	}

	leaseUpdater := lease.NewLeaseUpdater(
		spokeKubeClient,
		addOnName,
		o.InstallationNamespace,
	)
	go leaseUpdater.Start(ctx)

	<-ctx.Done()
	return nil
}
