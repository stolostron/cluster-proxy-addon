package configchecker

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
	"open-cluster-management.io/addon-framework/pkg/utils"
)

func NewConfigCheckerServer() *cobra.Command {
	options := struct {
		name  string
		port  int
		files []string
	}{}

	cmd := &cobra.Command{
		Use:   "config-checker",
		Short: "watch config udpate for containers",
		Run: func(cmd *cobra.Command, args []string) {
			// create a config-checker
			cc, err := utils.NewConfigChecker(options.name, options.files...)
			if err != nil {
				klog.Errorf("create config checker failed, %v", err)
			}

			// use checker in a handler
			http.HandleFunc("/"+cc.Name(), func(rw http.ResponseWriter, r *http.Request) {
				if err := cc.Check(r); err != nil {
					rw.WriteHeader(500)
					rw.Write([]byte(fmt.Sprintf("config check fail:%v", err)))
				} else {
					rw.WriteHeader(200)
					rw.Write([]byte("OK"))
				}
			})

			if err := http.ListenAndServe(":"+strconv.Itoa(options.port), nil); err != nil {
				klog.Errorf("listen to http err: %v", err)
			}
		},
	}

	cmd.Flags().StringVar(&options.name, "name", "", "other container should set liveness probe address to http://localhost:<port>/<name>")
	cmd.Flags().IntVar(&options.port, "port", 8080, "the port of config checker server")
	cmd.Flags().StringArrayVar(&options.files, "files", []string{}, "the files need to watch")

	return cmd
}
