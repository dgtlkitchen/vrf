package sidecar

import (
	"context"

	servertypes "github.com/vexxvakan/vrf/sidecar/servers/vrf/types"
)

type StubService struct{}

func NewStubService() Service {
	return StubService{}
}

func (StubService) Randomness(context.Context, uint64) (*servertypes.QueryRandomnessResponse, error) {
	return nil, ErrServiceUnavailable
}

func (StubService) Info(context.Context) (*servertypes.QueryInfoResponse, error) {
	return nil, ErrServiceUnavailable
}
