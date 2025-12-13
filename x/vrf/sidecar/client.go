package sidecar

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"cosmossdk.io/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	vrftypes "github.com/vexxvakan/vrf/sidecar/servers/vrf/types"
)

var _ Client = (*GRPCClient)(nil)

type GRPCClient struct {
	logger log.Logger
	mutex  sync.Mutex

	addr    string
	conn    *grpc.ClientConn
	client  vrftypes.VrfClient
	timeout time.Duration
}

func NewClient(
	logger log.Logger,
	addr string,
	timeout time.Duration,
) (*GRPCClient, error) {
	if logger == nil {
		return nil, fmt.Errorf("logger cannot be nil")
	}

	if timeout <= 0 {
		return nil, fmt.Errorf("timeout must be positive")
	}

	return &GRPCClient{
		logger:  logger,
		addr:    addr,
		timeout: timeout,
	}, nil
}

func (c *GRPCClient) Start(ctx context.Context) error {
	c.logger.Info("starting vrf sidecar client", "addr", c.addr)

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	if strings.HasPrefix(c.addr, "unix://") {
		path := strings.TrimPrefix(c.addr, "unix://")
		opts = append(opts, grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			var d net.Dialer
			return d.DialContext(ctx, "unix", path)
		}))
	}

	conn, err := grpc.NewClient(c.addr, opts...)
	if err != nil {
		c.logger.Error("failed to dial vrf sidecar gRPC server", "err", err)
		return fmt.Errorf("failed to dial vrf sidecar gRPC server: %w", err)
	}

	c.mutex.Lock()
	c.conn = conn
	c.client = vrftypes.NewVrfClient(conn)
	c.mutex.Unlock()

	c.logger.Info("vrf sidecar client started")

	return nil
}

func (c *GRPCClient) Stop() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.logger.Info("stopping vrf sidecar client")
	if c.conn == nil {
		return nil
	}

	err := c.conn.Close()
	c.client = nil
	c.logger.Info("vrf sidecar client stopped", "err", err)

	return err
}

func (c *GRPCClient) Randomness(
	ctx context.Context,
	req *vrftypes.QueryRandomnessRequest,
	opts ...grpc.CallOption,
) (*vrftypes.QueryRandomnessResponse, error) {
	c.mutex.Lock()
	cl := c.client
	c.mutex.Unlock()

	if cl == nil {
		return nil, fmt.Errorf("vrf sidecar client not started")
	}

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	return cl.Randomness(ctx, req, opts...)
}

func (c *GRPCClient) Info(
	ctx context.Context,
	req *vrftypes.QueryInfoRequest,
	opts ...grpc.CallOption,
) (*vrftypes.QueryInfoResponse, error) {
	c.mutex.Lock()
	cl := c.client
	c.mutex.Unlock()

	if cl == nil {
		return nil, fmt.Errorf("vrf sidecar client not started")
	}

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	return cl.Info(ctx, req, opts...)
}
