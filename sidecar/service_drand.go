package sidecar

import ( //nolint:depguard
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
	"sync/atomic"
	"time"

	"github.com/drand/drand/v2/common/chain"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"

	sidecarmetrics "github.com/vexxvakan/vrf/sidecar/servers/prometheus/metrics"
	servertypes "github.com/vexxvakan/vrf/sidecar/servers/vrf/types"
)

// DrandService implements Service by talking to a local drand HTTP endpoint
// using only statically configured URLs. It can also be used alongside a
// supervised drand subprocess (see StartDrandProcess).
type DrandService struct {
	cfg     Config
	logger  *zap.Logger
	metrics sidecarmetrics.Metrics

	httpClient *http.Client

	sf       singleflight.Group
	fetchSem chan struct{}

	lastSuccessUnixNano atomic.Int64
	chainInfo           *servertypes.QueryInfoResponse
}

// NewDrandService constructs a new DrandService, checking the configured drand
// binary version and validating /info against the configured chain params.
func NewDrandService(
	ctx context.Context,
	cfg Config,
	logger *zap.Logger,
	m sidecarmetrics.Metrics,
) (*DrandService, error) {
	if logger == nil {
		logger = zap.NewNop()
	}

	if m == nil {
		m = sidecarmetrics.NewNop()
	}

	if strings.TrimSpace(cfg.DrandHTTP) == "" {
		return nil, fmt.Errorf("drand HTTP endpoint must be provided")
	}

	if err := enforceLoopbackHTTP(cfg.DrandHTTP); err != nil {
		return nil, err
	}

	if len(cfg.ChainHash) == 0 || len(cfg.PublicKey) == 0 || cfg.PeriodSeconds == 0 || cfg.GenesisUnixSec == 0 {
		return nil, fmt.Errorf("drand chain configuration is incomplete: chain hash, public key, period, and genesis are required")
	}

	if err := checkDrandBinary(cfg, logger); err != nil {
		return nil, err
	}

	s := &DrandService{
		cfg: cfg,
		logger: logger.With(
			zap.String("component", "sidecar-drand-service"),
		),
		metrics:    m,
		fetchSem:   make(chan struct{}, 1),
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}

	info, err := s.fetchChainInfo(ctx)
	if err != nil {
		return nil, err
	}

	if err := ValidateDrandChainInfo(info, cfg); err != nil {
		return nil, err
	}

	s.chainInfo = info
	return s, nil
}

// ValidateDrandChainInfo enforces that the discovered drand chain info matches
// the configured expected values (which should match on-chain VrfParams).
func ValidateDrandChainInfo(info *servertypes.QueryInfoResponse, cfg Config) error {
	if info == nil {
		return fmt.Errorf("sidecar: nil drand chain info")
	}

	if !bytes.Equal(info.ChainHash, cfg.ChainHash) {
		return fmt.Errorf("sidecar: drand chain hash mismatch: got %x, expected %x", info.ChainHash, cfg.ChainHash)
	}

	if !bytes.Equal(info.PublicKey, cfg.PublicKey) {
		return fmt.Errorf("sidecar: drand public key mismatch")
	}

	if info.PeriodSeconds != cfg.PeriodSeconds {
		return fmt.Errorf("sidecar: drand period mismatch: got %d, expected %d", info.PeriodSeconds, cfg.PeriodSeconds)
	}

	if info.GenesisUnixSec != cfg.GenesisUnixSec {
		return fmt.Errorf("sidecar: drand genesis mismatch: got %d, expected %d", info.GenesisUnixSec, cfg.GenesisUnixSec)
	}

	return nil
}

func (s *DrandService) acquireFetch() func() {
	s.fetchSem <- struct{}{}
	return func() { <-s.fetchSem }
}

func (s *DrandService) observeTimeSinceLastSuccess(now time.Time) {
	lastNanos := s.lastSuccessUnixNano.Load()
	if lastNanos == 0 {
		s.metrics.ObserveTimeSinceLastSuccess(0)
		return
	}

	last := time.Unix(0, lastNanos)
	s.metrics.ObserveTimeSinceLastSuccess(now.Sub(last).Seconds())
}

// Randomness fetches a beacon for the given round from the configured drand
// HTTP endpoint. A round of zero requests the latest beacon. Fetches are
// serialized so that at most one upstream drand HTTP request is in-flight.
func (s *DrandService) Randomness(
	ctx context.Context,
	round uint64,
) (*servertypes.QueryRandomnessResponse, error) {
	key := fmt.Sprintf("round-%d", round)

	v, err, _ := s.sf.Do(key, func() (interface{}, error) {
		release := s.acquireFetch()
		defer release()
		return s.fetchBeacon(ctx, round)
	})

	s.observeTimeSinceLastSuccess(time.Now())

	if err != nil {
		return nil, err
	}

	beacon, ok := v.(*servertypes.QueryRandomnessResponse)
	if !ok {
		return nil, ErrServiceUnavailable
	}

	return beacon, nil
}

// Info returns the drand chain information discovered from /info. This is
// expected to match the on-chain VrfParams exactly.
func (s *DrandService) Info(ctx context.Context) (*servertypes.QueryInfoResponse, error) {
	if s.chainInfo != nil {
		return s.chainInfo, nil
	}

	info, err := s.fetchChainInfo(ctx)
	if err != nil {
		return nil, err
	}

	if err := ValidateDrandChainInfo(info, s.cfg); err != nil {
		return nil, err
	}

	s.chainInfo = info
	return info, nil
}

// drandHTTPBeacon is a minimal view of the drand HTTP randomness response.
type drandHTTPBeacon struct {
	Round             uint64 `json:"round"`
	Randomness        string `json:"randomness"`
	Signature         string `json:"signature"`
	PreviousSignature string `json:"previous_signature"`
}

func (s *DrandService) fetchBeacon(ctx context.Context, round uint64) (*servertypes.QueryRandomnessResponse, error) {
	chainHashHex := fmt.Sprintf("%x", s.cfg.ChainHash)

	path := fmt.Sprintf("/%s/public/latest", chainHashHex)
	if round > 0 {
		path = fmt.Sprintf("/%s/public/%d", chainHashHex, round)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		strings.TrimRight(s.cfg.DrandHTTP, "/")+path,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("creating drand request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		s.metrics.AddDrandFetch(sidecarmetrics.FetchFailure)
		s.logger.Warn("drand fetch failed", zap.Uint64("round", round), zap.String("chain_hash", chainHashHex), zap.Error(err))
		return nil, fmt.Errorf("querying drand: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		s.metrics.AddDrandFetch(sidecarmetrics.FetchFailure)
		s.logger.Warn("drand round not yet available", zap.Uint64("round", round), zap.String("chain_hash", chainHashHex))
		return nil, ErrRoundNotAvailable
	}

	if resp.StatusCode != http.StatusOK {
		s.metrics.AddDrandFetch(sidecarmetrics.FetchFailure)
		s.logger.Warn("drand returned non-200", zap.Uint64("round", round), zap.String("chain_hash", chainHashHex), zap.String("status", resp.Status))
		return nil, fmt.Errorf("drand returned non-200: %s", resp.Status)
	}

	var hb drandHTTPBeacon
	if err := json.NewDecoder(resp.Body).Decode(&hb); err != nil {
		s.metrics.AddDrandFetch(sidecarmetrics.FetchFailure)
		return nil, fmt.Errorf("decoding drand response: %w", err)
	}

	sig, err := decodeHexBytes(hb.Signature)
	if err != nil {
		s.metrics.AddDrandFetch(sidecarmetrics.FetchFailure)
		return nil, fmt.Errorf("decoding signature: %w", err)
	}

	var prevSig []byte
	if strings.TrimSpace(hb.PreviousSignature) != "" {
		prevSig, err = decodeHexBytes(hb.PreviousSignature)
		if err != nil {
			s.metrics.AddDrandFetch(sidecarmetrics.FetchFailure)
			return nil, fmt.Errorf("decoding previous signature: %w", err)
		}
	}

	randHash := sha256.Sum256(sig)

	// If the endpoint returned randomness, verify it matches sha256(signature).
	if strings.TrimSpace(hb.Randomness) != "" {
		gotRand, err := decodeHexBytes(hb.Randomness)
		if err != nil {
			s.metrics.AddDrandFetch(sidecarmetrics.FetchFailure)
			return nil, fmt.Errorf("decoding randomness: %w", err)
		}
		if !bytes.Equal(gotRand, randHash[:]) {
			s.metrics.AddDrandFetch(sidecarmetrics.FetchFailure)
			return nil, fmt.Errorf("drand randomness mismatch: sha256(signature) != randomness")
		}
	}

	s.metrics.AddDrandFetch(sidecarmetrics.FetchSuccess)
	s.metrics.SetDrandLatestRound(hb.Round)
	s.lastSuccessUnixNano.Store(time.Now().UnixNano())

	s.logger.Info(
		"fetched drand beacon",
		zap.Uint64("round", hb.Round),
		zap.String("chain_hash", chainHashHex),
	)

	return &servertypes.QueryRandomnessResponse{
		DrandRound:        hb.Round,
		Randomness:        randHash[:],
		Signature:         sig,
		PreviousSignature: prevSig,
	}, nil
}

func (s *DrandService) fetchChainInfo(ctx context.Context) (*servertypes.QueryInfoResponse, error) {
	chainHashHex := fmt.Sprintf("%x", s.cfg.ChainHash)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		strings.TrimRight(s.cfg.DrandHTTP, "/")+fmt.Sprintf("/%s/info", chainHashHex),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("creating drand /info request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("querying drand /info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("drand /info returned non-200: %s", resp.Status)
	}

	info, err := chain.InfoFromJSON(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("decoding drand /info response: %w", err)
	}

	pubKey, err := info.PublicKey.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("encoding drand public key: %w", err)
	}

	return &servertypes.QueryInfoResponse{
		ChainHash:      info.Hash(),
		PublicKey:      pubKey,
		PeriodSeconds:  uint64(info.Period / time.Second),
		GenesisUnixSec: info.GenesisTime,
	}, nil
}

func decodeHexBytes(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "0x")
	if s == "" {
		return nil, fmt.Errorf("empty hex string")
	}
	return hex.DecodeString(s)
}

func enforceLoopbackHTTP(endpoint string) error {
	u, err := url.Parse(endpoint)
	if err != nil {
		return fmt.Errorf("invalid drand HTTP endpoint: %w", err)
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("invalid drand HTTP endpoint scheme: %q", u.Scheme)
	}

	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("drand HTTP endpoint must include host")
	}

	if strings.EqualFold(host, "localhost") {
		return nil
	}

	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return fmt.Errorf("drand HTTP endpoint must be loopback-only, got host %q", host)
	}

	return nil
}

// checkDrandBinary runs "drand version" and, when ExpectedBinaryVersion is
// non-empty, enforces that the discovered version matches exactly.
func checkDrandBinary(cfg Config, logger *zap.Logger) error {
	path := cfg.BinaryPath
	if strings.TrimSpace(path) == "" {
		path = "drand"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, path, "version") //nolint:gosec

	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("running drand version: %w", err)
	}

	version := strings.TrimSpace(string(out))
	logger.Info("detected drand binary version", zap.String("version", version))

	if cfg.ExpectedBinaryVersion != "" && version != cfg.ExpectedBinaryVersion {
		return fmt.Errorf(
			"drand version mismatch: got %q, expected %q",
			version,
			cfg.ExpectedBinaryVersion,
		)
	}

	return nil
}
