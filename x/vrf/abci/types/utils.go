package types

import (
	"time"

	"github.com/cosmos/cosmos-sdk/telemetry"
	metrics "github.com/hashicorp/go-metrics"

	vrftypes "github.com/vexxvakan/vrf/x/vrf/types"
)

// RecordLatencyAndStatus is used by the ABCI handlers to record their e2e latency, and the status of the request
// to their corresponding metrics objects.
func RecordLatencyAndStatus(
	latency time.Duration,
	err error,
	method ABCIMethod,
) {
	if !telemetry.IsTelemetryEnabled() {
		return
	}

	telemetry.SetGaugeWithLabels(
		[]string{vrftypes.ModuleName, "abci", "latency_seconds"},
		float32(latency.Seconds()),
		[]metrics.Label{
			telemetry.NewLabel("method", string(method)),
		},
	)

	status := "success"
	if err != nil {
		status = "error"
	}

	telemetry.IncrCounterWithLabels(
		[]string{vrftypes.ModuleName, "abci", "requests_total"},
		1,
		[]metrics.Label{
			telemetry.NewLabel("method", string(method)),
			telemetry.NewLabel("status", status),
		},
	)
}

func RecordMessageSize(msgType MessageType, size int) {
	if !telemetry.IsTelemetryEnabled() {
		return
	}

	telemetry.SetGaugeWithLabels(
		[]string{vrftypes.ModuleName, "abci", "message_size_bytes"},
		float32(size),
		[]metrics.Label{
			telemetry.NewLabel("type", string(msgType)),
		},
	)

	telemetry.IncrCounterWithLabels(
		[]string{vrftypes.ModuleName, "abci", "message_bytes_total"},
		float32(size),
		[]metrics.Label{
			telemetry.NewLabel("type", string(msgType)),
		},
	)
}
