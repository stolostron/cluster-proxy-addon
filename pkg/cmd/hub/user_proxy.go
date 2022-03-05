package hub

import (
	"github.com/openshift/library-go/pkg/controller/controllercmd"
	"github.com/spf13/cobra"
	"github.com/stolostron/cluster-proxy-addon/pkg/hub"
	"github.com/stolostron/cluster-proxy-addon/pkg/version"
)

func NewUserServer() *cobra.Command {
	kubectlUI := hub.NewHTTPUserServer()

	cmd := controllercmd.NewControllerCommandConfig(
		"cluster-proxy-user-server",
		version.Get(),
		kubectlUI.Run,
	).NewCommand()
	cmd.Use = "user-server"
	cmd.Short = "user-server"

	kubectlUI.AddFlags(cmd)
	return cmd
}
