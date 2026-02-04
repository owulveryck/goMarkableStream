package main

import (
	"context"
	"fmt"
	"net"

	"golang.ngrok.com/ngrok"
	"golang.ngrok.com/ngrok/config"
)

// ListenerResult encapsulates listeners with their cleanup function and TLS state
type ListenerResult struct {
	Listeners        []net.Listener
	Cleanup          func() error
	UseTLS           bool // Whether caller should apply TLS
	TailscaleManager *TailscaleManager
}

func setupListener(ctx context.Context, s *configuration) (*ListenerResult, error) {
	switch s.BindAddr {
	case "ngrok":
		l, err := ngrok.Listen(ctx,
			config.HTTPEndpoint(),
			ngrok.WithAuthtokenFromEnv(),
		)
		if err != nil {
			return nil, err
		}
		s.BindAddr = l.Addr().String()
		return &ListenerResult{
			Listeners: []net.Listener{l},
			Cleanup:   func() error { return l.Close() },
			UseTLS:    false, // ngrok handles TLS
		}, nil

	case "tailscale":
		tm := NewTailscaleManager(s)
		if tm == nil {
			return nil, fmt.Errorf("Tailscale support not compiled in. Build with: go build -tags tailscale")
		}
		tsListener, err := tm.Start(ctx)
		if err != nil {
			return nil, err
		}

		// Also create local listener for LAN access (different port to avoid conflict)
		localListener, err := net.Listen("tcp", ":8443")
		if err != nil {
			tm.Close()
			return nil, err
		}

		return &ListenerResult{
			Listeners: []net.Listener{tsListener, localListener},
			Cleanup: func() error {
				localListener.Close()
				return tm.Close()
			},
			UseTLS:           false, // WireGuard encrypts Tailscale; local is plain HTTP
			TailscaleManager: tm,
		}, nil

	default:
		l, err := net.Listen("tcp", s.BindAddr)
		if err != nil {
			return nil, err
		}
		return &ListenerResult{
			Listeners: []net.Listener{l},
			Cleanup:   func() error { return l.Close() },
			UseTLS:    s.TLS,
		}, nil
	}
}
