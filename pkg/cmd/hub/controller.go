package hub

import (
	"github.com/spf13/cobra"

	"github.com/openshift/library-go/pkg/controller/controllercmd"

	"open-cluster-management.io/cluster-proxy-addon/pkg/hub"
	"open-cluster-management.io/cluster-proxy-addon/pkg/version"
)

func NewController() *cobra.Command {
	addOnControllerOptions := hub.NewAddOnControllerOptions()
	cmd := controllercmd.
		NewControllerCommandConfig("clsuter-proxy-addon-controller", version.Get(), addOnControllerOptions.RunControllerManager).
		NewCommand()
	cmd.Use = "controller"
	cmd.Short = "Start the clsuter proxy add-on controller"

	addOnControllerOptions.AddFlags(cmd)
	return cmd
}
