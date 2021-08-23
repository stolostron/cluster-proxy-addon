package hub

import (
	"github.com/open-cluster-management/cluster-proxy-addon/pkg/hub"

	"github.com/open-cluster-management/cluster-proxy-addon/pkg/version"
	"github.com/openshift/library-go/pkg/controller/controllercmd"
	"github.com/spf13/cobra"
)

func NewUserServer() *cobra.Command {
	userServerOptions := hub.NewUserServerOptions()

	cmd := controllercmd.NewControllerCommandConfig(
		"cluster-proxy-user-server",
		version.Get(),
		userServerOptions.Run,
	).NewCommand()
	cmd.Use = "user-server"
	cmd.Short = "user-server"

	userServerOptions.AddFlags(cmd)
	return cmd
}
