package prometheus

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

// stable defaults.
const (
	maxOpenConnections = 3
	readHeaderTimeout  = 10 * time.Second
)

// PrometheusServer is a prometheus server that serves metrics registered in the DefaultRegisterer.
// It is a wrapper around the promhttp.Handler() handler. The server will be started in a go-routine,
// and is gracefully stopped on close.
type PrometheusServer struct { //nolint
	srv    *http.Server
	done   chan struct{}
	logger *zap.Logger
}

// NewPrometheusServer creates a prometheus server if the metrics are enabled and
// address is set, and valid. Notice, this method does not start the server.
func NewPrometheusServer(prometheusAddress string, logger *zap.Logger) (*PrometheusServer, error) {
	// get the prometheus server address
	if !isValidAddress(prometheusAddress) {
		return nil, fmt.Errorf("invalid prometheus server address: %s", prometheusAddress)
	}
	srv := &http.Server{
		Addr: prometheusAddress,
		Handler: promhttp.InstrumentMetricHandler(
			prometheus.DefaultRegisterer, promhttp.HandlerFor(
				prometheus.DefaultGatherer,
				promhttp.HandlerOpts{MaxRequestsInFlight: maxOpenConnections},
			),
		),
		ReadHeaderTimeout: readHeaderTimeout,
	}

	logger = logger.With(zap.String("server", "prometheus"))
	ps := &PrometheusServer{
		srv:    srv,
		done:   make(chan struct{}),
		logger: logger,
	}

	return ps, nil
}

// Start will spawn a http server that will handle requests to /metrics
// and serves the metrics registered in the DefaultRegisterer.
func (ps *PrometheusServer) Start() {
	if err := ps.srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		ps.logger.Info("prometheus server error", zap.Error(err))
	} else {
		ps.logger.Info("prometheus server closed")
	}

	// close the done channel
	close(ps.done)
}

// Close gracefully closes the server.
func (ps *PrometheusServer) Close() {
	if ps == nil || ps.srv == nil {
		return
	}

	if err := ps.srv.Close(); err != nil {
		ps.logger.Info("prometheus server close error", zap.Error(err))
	}
}

// Done exposes a channel that is closed when the server stops.
func (ps *PrometheusServer) Done() <-chan struct{} {
	if ps == nil {
		ch := make(chan struct{})
		close(ch)
		return ch
	}
	return ps.done
}

func isValidAddress(addr string) bool {
	if addr == "" {
		return false
	}

	host, port, err := net.SplitHostPort(addr)
	if err != nil || port == "" {
		return false
	}

	if host == "" {
		return false
	}

	return true
}
