package sidecar

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"

	sidecarmetrics "github.com/vexxvakan/vrf/sidecar/servers/prometheus/metrics"
)

// DrandProcessConfig configures the supervised drand subprocess.
type DrandProcessConfig struct {
	BinaryPath string
	DataDir    string

	PrivateListen string
	PublicListen  string
	ControlListen string

	ExtraArgs []string
}

// DrandProcess supervises a local drand daemon process.
type DrandProcess struct {
	cfg     DrandProcessConfig
	logger  *zap.Logger
	metrics sidecarmetrics.Metrics

	ctx    context.Context
	cancel context.CancelFunc

	mu  sync.Mutex
	cmd *exec.Cmd

	done chan struct{}
}

func StartDrandProcess(
	parentCtx context.Context,
	cfg DrandProcessConfig,
	logger *zap.Logger,
	m sidecarmetrics.Metrics,
) (*DrandProcess, error) {
	if logger == nil {
		logger = zap.NewNop()
	}

	if m == nil {
		m = sidecarmetrics.NewNop()
	}

	if strings.TrimSpace(cfg.DataDir) == "" {
		return nil, fmt.Errorf("drand data dir must be provided")
	}

	if strings.TrimSpace(cfg.PrivateListen) == "" {
		return nil, fmt.Errorf("drand private listen address must be provided")
	}

	if strings.TrimSpace(cfg.PublicListen) == "" {
		return nil, fmt.Errorf("drand public listen address must be provided")
	}

	if strings.TrimSpace(cfg.ControlListen) == "" {
		return nil, fmt.Errorf("drand control listen address must be provided")
	}

	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating drand data dir: %w", err)
	}

	ctx, cancel := context.WithCancel(parentCtx)
	p := &DrandProcess{
		cfg: cfg,
		logger: logger.With(
			zap.String("component", "sidecar-drand-process"),
		),
		metrics: m,
		ctx:     ctx,
		cancel:  cancel,
		done:    make(chan struct{}),
	}

	// Ensure drand is started successfully at least once.
	if err := p.startOnce(); err != nil {
		cancel()
		return nil, err
	}

	go p.supervise()
	return p, nil
}

func (p *DrandProcess) supervise() {
	defer close(p.done)

	backoff := time.Second
	for {
		err := p.waitCurrent()

		p.metrics.SetDrandProcessHealthy(false)
		if p.ctx.Err() != nil {
			return
		}

		p.logger.Warn("drand process exited; restarting", zap.Error(err))

		select {
		case <-time.After(backoff):
		case <-p.ctx.Done():
			return
		}

		if backoff < 30*time.Second {
			backoff *= 2
		}

		if err := p.startOnce(); err != nil {
			p.logger.Error("failed to restart drand; retrying", zap.Error(err))
		}
	}
}

func (p *DrandProcess) startOnce() error {
	bin := strings.TrimSpace(p.cfg.BinaryPath)
	if bin == "" {
		bin = "drand"
	}

	args := []string{
		"start",
		"--folder", p.cfg.DataDir,
		"--private-listen", p.cfg.PrivateListen,
		"--public-listen", p.cfg.PublicListen,
		"--control", p.cfg.ControlListen,
	}
	args = append(args, p.cfg.ExtraArgs...)

	cmd := exec.Command(bin, args...) //nolint:gosec

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("drand stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("drand stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting drand: %w", err)
	}

	p.mu.Lock()
	p.cmd = cmd
	p.mu.Unlock()

	p.metrics.SetDrandProcessHealthy(true)
	p.logger.Info("started drand daemon", zap.Int("pid", cmd.Process.Pid))

	go p.pipeToLogger(stdout, "stdout")
	go p.pipeToLogger(stderr, "stderr")

	// Ensure the child exits when the sidecar is shutting down.
	go func() {
		<-p.ctx.Done()
		_ = cmd.Process.Signal(syscall.SIGTERM)
	}()

	return nil
}

func (p *DrandProcess) waitCurrent() error {
	p.mu.Lock()
	cmd := p.cmd
	p.mu.Unlock()

	if cmd == nil {
		return fmt.Errorf("drand process is not started")
	}

	return cmd.Wait()
}

func (p *DrandProcess) pipeToLogger(r io.ReadCloser, stream string) {
	defer r.Close()

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		p.logger.Info("drand", zap.String("stream", stream), zap.String("line", line))
	}
}

// Stop terminates the supervised drand process and stops further restarts.
func (p *DrandProcess) Stop() {
	if p == nil {
		return
	}

	p.cancel()

	p.mu.Lock()
	cmd := p.cmd
	p.mu.Unlock()

	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Signal(syscall.SIGTERM)
	}

	select {
	case <-p.done:
	case <-time.After(10 * time.Second):
		if cmd != nil && cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		<-p.done
	}
}
