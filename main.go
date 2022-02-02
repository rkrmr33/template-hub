package main

import (
	"context"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/rkrmr33/pkg/log"
	"github.com/rkrmr33/template-hub/common"
	"github.com/rkrmr33/template-hub/server"
	"github.com/rkrmr33/template-hub/util"
)

func main() {
	var (
		conf *server.Config
		ctx  = context.Background()
	)

	ctx = util.ContextWithCancelOnSignals(ctx, syscall.SIGINT, syscall.SIGTERM)

	cmd := &cobra.Command{
		Use:   common.Bin,
		Short: "Run template registry server",
		RunE: func(cmd *cobra.Command, args []string) error {
			srv, err := conf.Build()
			if err != nil {
				return err
			}

			return srv.Start(cmd.Context())
		},
	}

	conf = server.AddFlags(cmd.Flags())

	if err := cmd.ExecuteContext(ctx); err != nil {
		log.G().Fatal(err)
	}
}
