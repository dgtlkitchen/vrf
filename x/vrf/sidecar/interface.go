package sidecar

import (
	"context"

	"google.golang.org/grpc"

	vrftypes "github.com/vexxvakan/vrf/sidecar/servers/vrf/types"
)

type Client interface {
	Randomness(ctx context.Context, in *vrftypes.QueryRandomnessRequest, opts ...grpc.CallOption) (*vrftypes.QueryRandomnessResponse, error)
	Info(ctx context.Context, in *vrftypes.QueryInfoRequest, opts ...grpc.CallOption) (*vrftypes.QueryInfoResponse, error)

	Start(context.Context) error
	Stop() error
}

type NoOpClient struct{}

func (NoOpClient) Start(context.Context) error {
	return nil
}

func (NoOpClient) Stop() error {
	return nil
}

func (NoOpClient) Randomness(
	_ context.Context,
	_ *vrftypes.QueryRandomnessRequest,
	_ ...grpc.CallOption,
) (*vrftypes.QueryRandomnessResponse, error) {
	return nil, nil
}

func (NoOpClient) Info(
	_ context.Context,
	_ *vrftypes.QueryInfoRequest,
	_ ...grpc.CallOption,
) (*vrftypes.QueryInfoResponse, error) {
	return nil, nil
}
