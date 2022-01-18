package spoke

import (
	"github.com/spf13/cobra"

	"github.com/openshift/library-go/pkg/controller/controllercmd"

	"github.com/stolostron/cluster-proxy-addon/pkg/spoke"
	"github.com/stolostron/cluster-proxy-addon/pkg/version"
)

func NewAgent() *cobra.Command {
	agentOptions := spoke.NewAgentOptions()
	cmd := controllercmd.
		NewControllerCommandConfig("cluster-proxy-addon-agent", version.Get(), agentOptions.RunAgent).
		NewCommand()
	cmd.Use = "agent"
	cmd.Short = "Start the cluster proxy add-on agent"

	agentOptions.AddFlags(cmd)
	return cmd
}
