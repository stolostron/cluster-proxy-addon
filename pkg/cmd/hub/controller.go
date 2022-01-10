package hub

import (
	"github.com/spf13/cobra"

	"github.com/openshift/library-go/pkg/controller/controllercmd"

	"github.com/stolostron/cluster-proxy-addon/pkg/hub"
	"github.com/stolostron/cluster-proxy-addon/pkg/version"
)

func NewController() *cobra.Command {
	addOnControllerOptions := hub.NewAddOnControllerOptions()
	cmd := controllercmd.
		NewControllerCommandConfig("cluster-proxy-addon-controller", version.Get(), addOnControllerOptions.RunControllerManager).
		NewCommand()
	cmd.Use = "controller"
	cmd.Short = "Start the cluster proxy add-on controller"

	addOnControllerOptions.AddFlags(cmd)
	return cmd
}
