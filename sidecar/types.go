package sidecar

import (
	"context"

	servertypes "github.com/vexxvakan/vrf/sidecar/servers/vrf/types"
)

type Service interface {
	Randomness(ctx context.Context, round uint64) (*servertypes.QueryRandomnessResponse, error)
	Info(ctx context.Context) (*servertypes.QueryInfoResponse, error)
}
