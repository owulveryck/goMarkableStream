package main

import (
	"context"
	"net"

	"golang.ngrok.com/ngrok"
	"golang.ngrok.com/ngrok/config"
	"tailscale.com/tsnet"
)

func setupListener(ctx context.Context, s *configuration) (net.Listener, error) {
	switch s.BindAddr {
	case "tailscale":
		srv := new(tsnet.Server)
		srv.Hostname = "gomarkablestream"
		if !s.DevMode {
			srv.Logf = func(string, ...any) {}
		}
		// Disable logs
		return srv.Listen("tcp", ":2001")
	case "ngrok":
		l, err := ngrok.Listen(ctx,
			config.HTTPEndpoint(),
			ngrok.WithAuthtokenFromEnv(),
		)
		s.BindAddr = l.Addr().String()
		c.TLS = false
		return l, err
	default:
		return net.Listen("tcp", s.BindAddr)
	}
}
