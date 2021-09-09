package main

import (
	goflag "flag"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/open-cluster-management/cluster-proxy-addon/pkg/cmd/configchecker"
	"github.com/open-cluster-management/cluster-proxy-addon/pkg/cmd/hub"
	"github.com/open-cluster-management/cluster-proxy-addon/pkg/cmd/spoke"

	utilflag "k8s.io/component-base/cli/flag"
	"k8s.io/component-base/logs"

	"github.com/open-cluster-management/cluster-proxy-addon/pkg/version"
)

func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	pflag.CommandLine.SetNormalizeFunc(utilflag.WordSepNormalizeFunc)
	pflag.CommandLine.AddGoFlagSet(goflag.CommandLine)

	logs.InitLogs()
	defer logs.FlushLogs()

	command := newClusterProxyCommand()
	if err := command.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func newClusterProxyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cluster-proxy",
		Short: "cluster-proxy",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
			os.Exit(1)
		},
	}

	if v := version.Get().String(); len(v) == 0 {
		cmd.Version = "<unknown>"
	} else {
		cmd.Version = v
	}

	cmd.AddCommand(hub.NewController())
	cmd.AddCommand(hub.NewUserServer())
	cmd.AddCommand(spoke.NewAgent())
	cmd.AddCommand(spoke.NewAPIServerProxy())
	cmd.AddCommand(configchecker.NewConfigCheckerServer())

	return cmd
}
