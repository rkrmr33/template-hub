package apiclient

import (
	"context"
	"crypto/tls"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/rkrmr33/pkg/log"
	"github.com/rkrmr33/template-hub/common"
	"github.com/rkrmr33/template-hub/pkg/api/registry"
	"github.com/rkrmr33/template-hub/pkg/api/version"
	"github.com/rkrmr33/template-hub/util"
	grpcutil "github.com/rkrmr33/template-hub/util/grpc"
)

const (
	MaxGRPCMessageSize = 1 << 20 // 100mb
)

type Client interface {
	NewVerionClient() version.VersionServiceClient
	NewRegistryClient() registry.RegistryServiceClient
}

type client struct {
	cc *grpc.ClientConn
}

// ClientOptions options for creating a template-hub api client
type ClientOptions struct {
	// Host refers to the address of the template-hub
	Host string
}

func AddFlags(cmd *cobra.Command) *ClientOptions {
	opts := &ClientOptions{}

	util.Must(viper.BindEnv("host", "TEMPLATE_HUB_HOST"))
	viper.SetDefault("host", fmt.Sprintf("localhost:%d", common.ServerPort))

	cmd.PersistentFlags().StringVar(&opts.Host, "host", viper.GetString("host"), "template-hub server host address")

	return opts
}

func (c *ClientOptions) Build(ctx context.Context) (Client, error) {
	tlsConfig := tls.Config{
		InsecureSkipVerify: true,
	}
	creds := credentials.NewTLS(&tlsConfig)

	dOpts := []grpc.DialOption{
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(MaxGRPCMessageSize), grpc.MaxCallSendMsgSize(MaxGRPCMessageSize)),
	}

	log.G().Debugf("dialing to grpc server: %s", c.Host)

	cc, err := grpcutil.BlockingDial(ctx, "tcp", c.Host, creds, dOpts...)
	if err != nil {
		return nil, err
	}

	return &client{
		cc,
	}, nil
}

func (c *ClientOptions) InitializePreCommand(client *Client) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if cmd.Parent() != nil && cmd.Parent().PersistentPreRunE != nil {
			if err := cmd.Parent().PersistentPreRunE(cmd.Parent(), args); err != nil {
				return err
			}
		}

		log.G().Debug("building template-hub client...")

		var err error
		*client, err = c.Build(cmd.Context())

		return err
	}
}

func (c *client) NewVerionClient() version.VersionServiceClient {
	return version.NewVersionServiceClient(c.cc)
}

func (c *client) NewRegistryClient() registry.RegistryServiceClient {
	return registry.NewRegistryServiceClient(c.cc)
}
