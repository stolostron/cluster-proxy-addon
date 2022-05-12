package hub

import (
	"github.com/spf13/cobra"
	"github.com/stolostron/cluster-proxy-addon/pkg/hub"
)

func NewUserServer() *cobra.Command {
	kubectlUI := hub.NewHTTPUserServer()

	cmd := &cobra.Command{
		Use:   "user-server",
		Short: "user-server",
		Long:  `A http proxy server, receives http requests from users and forwards to the ANP proxy-server.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return kubectlUI.Run(cmd.Context())
		},
	}

	kubectlUI.AddFlags(cmd)
	return cmd
}
