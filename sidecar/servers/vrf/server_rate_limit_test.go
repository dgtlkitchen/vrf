package vrf

import (
	"context"
	"net"
	"testing"

	"golang.org/x/time/rate"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	vrftypes "github.com/vexxvakan/vrf/sidecar/servers/vrf/types"
)

func TestRandomness_PerClientRateLimit_IsolatedBetweenTCPClients(t *testing.T) {
	server := NewServer(stubService{}, nil, nil)
	server.limiter = rate.NewLimiter(rate.Inf, 1) // disable global limiting for this test
	server.perClientRate = 0
	server.perClientBurst = 1

	ctxClient1 := peer.NewContext(context.Background(), &peer.Peer{
		Addr: &net.TCPAddr{IP: net.ParseIP("192.0.2.10"), Port: 1234},
	})
	ctxClient2 := peer.NewContext(context.Background(), &peer.Peer{
		Addr: &net.TCPAddr{IP: net.ParseIP("192.0.2.11"), Port: 1234},
	})

	_, err := server.Randomness(ctxClient1, &vrftypes.QueryRandomnessRequest{Round: 1})
	if err != nil {
		t.Fatalf("client1 first request should succeed, got: %v", err)
	}

	_, err = server.Randomness(ctxClient1, &vrftypes.QueryRandomnessRequest{Round: 1})
	if status.Code(err) != codes.ResourceExhausted {
		t.Fatalf("client1 second request should be rate-limited, got: %v", err)
	}

	_, err = server.Randomness(ctxClient2, &vrftypes.QueryRandomnessRequest{Round: 1})
	if err != nil {
		t.Fatalf("client2 request should not be impacted by client1 throttling, got: %v", err)
	}
}

func TestRandomness_PerClientRateLimit_TCPClientsIdentifiedByIPNotPort(t *testing.T) {
	server := NewServer(stubService{}, nil, nil)
	server.limiter = rate.NewLimiter(rate.Inf, 1) // disable global limiting for this test
	server.perClientRate = 0
	server.perClientBurst = 1

	ctxClientA := peer.NewContext(context.Background(), &peer.Peer{
		Addr: &net.TCPAddr{IP: net.ParseIP("203.0.113.5"), Port: 1111},
	})
	ctxClientB := peer.NewContext(context.Background(), &peer.Peer{
		Addr: &net.TCPAddr{IP: net.ParseIP("203.0.113.5"), Port: 2222},
	})

	_, err := server.Randomness(ctxClientA, &vrftypes.QueryRandomnessRequest{Round: 1})
	if err != nil {
		t.Fatalf("clientA first request should succeed, got: %v", err)
	}

	_, err = server.Randomness(ctxClientB, &vrftypes.QueryRandomnessRequest{Round: 1})
	if status.Code(err) != codes.ResourceExhausted {
		t.Fatalf("clientB should share rate limiter with clientA (same IP), got: %v", err)
	}
}

func TestRandomness_PerClientRateLimit_IsolatedBetweenUDSClients(t *testing.T) {
	server := NewServer(stubService{}, nil, nil)
	server.limiter = rate.NewLimiter(rate.Inf, 1) // disable global limiting for this test
	server.perClientRate = 0
	server.perClientBurst = 1

	ctxClient1 := peer.NewContext(context.Background(), &peer.Peer{
		Addr: &net.UnixAddr{Net: "unix", Name: "pid=100"},
	})
	ctxClient2 := peer.NewContext(context.Background(), &peer.Peer{
		Addr: &net.UnixAddr{Net: "unix", Name: "pid=200"},
	})

	_, err := server.Randomness(ctxClient1, &vrftypes.QueryRandomnessRequest{Round: 1})
	if err != nil {
		t.Fatalf("client1 first request should succeed, got: %v", err)
	}

	_, err = server.Randomness(ctxClient1, &vrftypes.QueryRandomnessRequest{Round: 1})
	if status.Code(err) != codes.ResourceExhausted {
		t.Fatalf("client1 second request should be rate-limited, got: %v", err)
	}

	_, err = server.Randomness(ctxClient2, &vrftypes.QueryRandomnessRequest{Round: 1})
	if err != nil {
		t.Fatalf("client2 request should not be impacted by client1 throttling, got: %v", err)
	}
}
