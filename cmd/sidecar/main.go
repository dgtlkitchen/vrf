package main

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/vexxvakan/vrf/sidecar"
	"github.com/vexxvakan/vrf/sidecar/servers/prometheus"
	sidecarmetrics "github.com/vexxvakan/vrf/sidecar/servers/prometheus/metrics"
	vrfserver "github.com/vexxvakan/vrf/sidecar/servers/vrf"
)

func main() {
	os.Exit(run())
}

func run() int {
	listenAddr := flag.String("listen-addr", "127.0.0.1:8090", "sidecar gRPC listen address (loopback or UDS via unix://)")
	allowPublic := flag.Bool("vrf-allow-public-bind", false, "allow sidecar to bind to non-loopback addresses (unsafe; operators must secure access)")

	metricsEnabled := flag.Bool("metrics-enabled", false, "enable Prometheus metrics")
	metricsAddr := flag.String("metrics-addr", "127.0.0.1:8091", "Prometheus metrics listen address (loopback only)")
	chainID := flag.String("chain-id", "", "chain ID label for metrics (optional)")

	drandSupervise := flag.Bool("drand-supervise", true, "start and supervise a local drand subprocess")
	drandHTTP := flag.String("drand-http", "", "drand HTTP base URL (defaults to http://<drand-public-addr>)")
	drandPublic := flag.String("drand-public-addr", "127.0.0.1:8081", "drand public listen address (also used for HTTP)")
	drandPrivate := flag.String("drand-private-addr", "0.0.0.0:4444", "drand private listen address")
	drandControl := flag.String("drand-control-addr", "127.0.0.1:8881", "drand control listen address")
	drandDataDir := flag.String("drand-data-dir", "", "drand data directory (required when --drand-supervise)")

	drandBinary := flag.String("drand-binary", "drand", "path to drand binary")
	drandVersion := flag.String("drand-expected-version", "", "expected drand version string (optional, exact match)")
	chainHashHex := flag.String("drand-chain-hash", "", "expected drand chain hash (hex)")
	publicKeyB64 := flag.String("drand-public-key", "", "expected drand group public key (base64)")
	periodSeconds := flag.Uint("drand-period-seconds", 0, "drand beacon period in seconds")
	genesisUnix := flag.Int64("drand-genesis-unix", 0, "drand genesis time (unix seconds)")
	flag.Parse()

	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer func() { _ = logger.Sync() }()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if !*allowPublic && !isLoopbackAddr(*listenAddr) {
		logger.Error("refusing to bind sidecar to non-loopback address without --vrf-allow-public-bind", zap.String("addr", *listenAddr))
		return 1
	}

	if *metricsEnabled && !*allowPublic && !isLoopbackAddr(*metricsAddr) {
		logger.Error("refusing to bind metrics to non-loopback address without --vrf-allow-public-bind", zap.String("addr", *metricsAddr))
		return 1
	}

	metrics, err := sidecarmetrics.NewFromConfig(*metricsEnabled, *chainID)
	if err != nil {
		logger.Error("failed to initialize metrics", zap.Error(err))
		return 1
	}

	if *metricsEnabled {
		ps, err := prometheus.NewPrometheusServer(*metricsAddr, logger)
		if err != nil {
			logger.Error("failed to create prometheus server", zap.Error(err))
			return 1
		}

		go ps.Start()
		defer ps.Close()
	}

	cfg := sidecar.Config{
		DrandSupervise:        *drandSupervise,
		DrandHTTP:             strings.TrimSpace(*drandHTTP),
		BinaryPath:            *drandBinary,
		ExpectedBinaryVersion: *drandVersion,
		DrandDataDir:          *drandDataDir,
		DrandPublicListen:     *drandPublic,
		DrandPrivateListen:    *drandPrivate,
		DrandControlListen:    *drandControl,
	}

	if cfg.DrandHTTP == "" {
		cfg.DrandHTTP = "http://" + cfg.DrandPublicListen
	}

	if *chainHashHex != "" {
		chainHash, decodeErr := hex.DecodeString(*chainHashHex)
		if decodeErr != nil {
			logger.Error("invalid drand chain hash; must be hex", zap.Error(decodeErr))
			return 1
		}
		cfg.ChainHash = chainHash
	}

	if *publicKeyB64 != "" {
		pubKey, decodeErr := base64.StdEncoding.DecodeString(*publicKeyB64)
		if decodeErr != nil {
			logger.Error("invalid drand public key; must be base64", zap.Error(decodeErr))
			return 1
		}
		cfg.PublicKey = pubKey
	}

	if *periodSeconds > 0 {
		cfg.PeriodSeconds = uint64(*periodSeconds)
	}

	if *genesisUnix > 0 {
		cfg.GenesisUnixSec = *genesisUnix
	}

	var proc *sidecar.DrandProcess
	if cfg.DrandSupervise {
		if strings.TrimSpace(cfg.DrandDataDir) == "" {
			logger.Error("--drand-data-dir is required when --drand-supervise=true")
			return 1
		}

		proc, err = sidecar.StartDrandProcess(ctx, sidecar.DrandProcessConfig{
			BinaryPath:    cfg.BinaryPath,
			DataDir:       cfg.DrandDataDir,
			PrivateListen: cfg.DrandPrivateListen,
			PublicListen:  cfg.DrandPublicListen,
			ControlListen: cfg.DrandControlListen,
		}, logger, metrics)
		if err != nil {
			logger.Error("failed to start drand subprocess", zap.Error(err))
			return 1
		}
		defer proc.Stop()
	}

	svc, err := newDrandServiceWithRetry(ctx, cfg, logger, metrics, 30*time.Second)
	if err != nil {
		logger.Error("failed to create drand service", zap.Error(err))
		return 1
	}

	server := vrfserver.NewServer(svc, logger)
	if err := server.Start(ctx, *listenAddr); err != nil {
		logger.Error("sidecar server exited with error", zap.Error(err))
		return 1
	}

	return 0
}

func newDrandServiceWithRetry(
	ctx context.Context,
	cfg sidecar.Config,
	logger *zap.Logger,
	metrics sidecarmetrics.Metrics,
	timeout time.Duration,
) (*sidecar.DrandService, error) {
	deadline := time.Now().Add(timeout)
	var lastErr error

	for time.Now().Before(deadline) {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		svc, err := sidecar.NewDrandService(ctx, cfg, logger, metrics)
		if err == nil {
			return svc, nil
		}
		lastErr = err

		logger.Warn("drand service not ready yet; retrying", zap.Error(err))
		select {
		case <-time.After(1 * time.Second):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	return nil, lastErr
}

func isLoopbackAddr(addr string) bool {
	if strings.HasPrefix(addr, "unix://") {
		return true
	}

	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}

	if strings.EqualFold(host, "localhost") {
		return true
	}

	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback()
	}

	return false
}
