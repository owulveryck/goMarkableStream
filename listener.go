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

	// Always create local listener first - this is immediately usable
	localListener, err := net.Listen("tcp", s.BindAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to create local listener on %s: %w", s.BindAddr, err)
	}
	listeners = append(listeners, localListener)

	// If Tailscale is enabled, start it in background (non-blocking)
	if s.TailscaleEnabled {
		tm = NewTailscaleManager(s)
		if tm == nil {
			localListener.Close()
			return nil, fmt.Errorf("tailscale support not compiled in: build with 'go build -tags tailscale'")
		}
		// Start Tailscale in background - does not block
		tm.StartAsync(ctx)
	}

	cleanup := func() error {
		localListener.Close()
		if tm != nil {
			return tm.Close()
		}
		return nil
	}

	return &ListenerResult{
		Listeners:        listeners, // Only local listener initially
		Cleanup:          cleanup,
		UseTLS:           s.TLS && !s.TailscaleEnabled,
		TailscaleManager: tm,
	}, nil
}
