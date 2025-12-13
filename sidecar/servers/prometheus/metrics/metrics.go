package metrics

import (
	"errors"

	"github.com/prometheus/client_golang/prometheus"
)

type FetchResult string

const (
	FetchSuccess FetchResult = "success"
	FetchFailure FetchResult = "failure"
)

type Metrics interface {
	SetDrandLatestRound(round uint64)
	AddDrandFetch(result FetchResult)
	SetDrandProcessHealthy(healthy bool)
	ObserveTimeSinceLastSuccess(seconds float64)
}

type nopMetrics struct{}

func NewNop() Metrics { return nopMetrics{} }

func (nopMetrics) SetDrandLatestRound(uint64)          {}
func (nopMetrics) AddDrandFetch(FetchResult)           {}
func (nopMetrics) SetDrandProcessHealthy(bool)         {}
func (nopMetrics) ObserveTimeSinceLastSuccess(float64) {}

func NewFromConfig(enabled bool, chainID string) (Metrics, error) {
	if !enabled {
		return NewNop(), nil
	}

	m := newPromMetrics(chainID)
	if err := m.register(); err != nil {
		return nil, err
	}
	return m, nil
}

type promMetrics struct {
	chainID string

	drandLatestRound          *prometheus.GaugeVec
	drandFetchCounter         *prometheus.CounterVec
	drandProcessHealthy       *prometheus.GaugeVec
	drandTimeSinceLastSuccess *prometheus.GaugeVec
}

func newPromMetrics(chainID string) *promMetrics {
	return &promMetrics{
		chainID: chainID,
		drandLatestRound: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "app",
			Name:      "vrf_drand_latest_round",
			Help:      "Latest successfully verified drand round served by the VRF sidecar",
		}, []string{"chain_id"}),
		drandFetchCounter: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "app",
			Name:      "vrf_drand_fetch_total",
			Help:      "Count of drand fetch attempts grouped by result",
		}, []string{"chain_id", "result"}),
		drandProcessHealthy: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "app",
			Name:      "vrf_drand_process_healthy",
			Help:      "Health flag for drand subprocess (1 healthy, 0 unhealthy)",
		}, []string{"chain_id"}),
		drandTimeSinceLastSuccess: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "app",
			Name:      "vrf_drand_time_since_last_successful_fetch_seconds",
			Help:      "Seconds since last successful drand fetch",
		}, []string{"chain_id"}),
	}
}

func (m *promMetrics) register() error {
	for _, c := range []prometheus.Collector{
		m.drandLatestRound,
		m.drandFetchCounter,
		m.drandProcessHealthy,
		m.drandTimeSinceLastSuccess,
	} {
		if err := prometheus.Register(c); err != nil {
			var already prometheus.AlreadyRegisteredError
			if errors.As(err, &already) {
				continue
			}
			return err
		}
	}
	return nil
}

func (m *promMetrics) SetDrandLatestRound(round uint64) {
	m.drandLatestRound.With(prometheus.Labels{
		"chain_id": m.chainID,
	}).Set(float64(round))
}

func (m *promMetrics) AddDrandFetch(result FetchResult) {
	m.drandFetchCounter.With(prometheus.Labels{
		"chain_id": m.chainID,
		"result":   string(result),
	}).Inc()
}

func (m *promMetrics) SetDrandProcessHealthy(healthy bool) {
	val := 0.0
	if healthy {
		val = 1.0
	}
	m.drandProcessHealthy.With(prometheus.Labels{
		"chain_id": m.chainID,
	}).Set(val)
}

func (m *promMetrics) ObserveTimeSinceLastSuccess(seconds float64) {
	m.drandTimeSinceLastSuccess.With(prometheus.Labels{
		"chain_id": m.chainID,
	}).Set(seconds)
}
