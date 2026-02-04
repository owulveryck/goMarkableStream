package main

import (
	"context"
	"fmt"
	"net"
)

// ListenerResult encapsulates listeners with their cleanup function and TLS state
type ListenerResult struct {
	Listeners        []net.Listener
	Cleanup          func() error
	UseTLS           bool // Whether caller should apply TLS
	TailscaleManager *TailscaleManager
}

func setupListener(ctx context.Context, s *configuration) (*ListenerResult, error) {
	var listeners []net.Listener
	var tm *TailscaleManager
	var cleanup func() error

	// Always create local listener using BindAddr
	localListener, err := net.Listen("tcp", s.BindAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to create local listener on %s: %w", s.BindAddr, err)
	}
	listeners = append(listeners, localListener)

	// If Tailscale is enabled, create additional Tailscale listener
	if s.TailscaleEnabled {
		tm = NewTailscaleManager(s)
		if tm == nil {
			localListener.Close()
			return nil, fmt.Errorf("Tailscale support not compiled in. Build with: go build -tags tailscale")
		}

		tsListener, err := tm.Start(ctx)
		if err != nil {
			localListener.Close()
			return nil, fmt.Errorf("failed to start Tailscale: %w", err)
		}

		// Prepend Tailscale listener (index 0 is used for Tailscale detection in main.go)
		listeners = append([]net.Listener{tsListener}, listeners...)

		cleanup = func() error {
			localListener.Close()
			return tm.Close()
		}
	} else {
		cleanup = func() error { return localListener.Close() }
	}

	return &ListenerResult{
		Listeners:        listeners,
		Cleanup:          cleanup,
		UseTLS:           s.TLS && !s.TailscaleEnabled,
		TailscaleManager: tm,
	}, nil
}
