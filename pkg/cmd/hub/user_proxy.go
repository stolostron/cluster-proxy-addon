package hub

import (
	"github.com/stolostron/cluster-proxy-addon/pkg/hub"

	"github.com/openshift/library-go/pkg/controller/controllercmd"
	"github.com/spf13/cobra"
	"github.com/stolostron/cluster-proxy-addon/pkg/version"
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
