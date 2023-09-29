package main

import (
	"context"
	"net"

	"golang.ngrok.com/ngrok"
	"golang.ngrok.com/ngrok/config"
)

func setupListener(ctx context.Context, s *configuration) (net.Listener, error) {
	if s.BindAddr == "ngrok" {
		l, err := ngrok.Listen(ctx,
			config.HTTPEndpoint(),
			ngrok.WithAuthtokenFromEnv(),
		)
		s.BindAddr = l.Addr().String()
		c.TLS = false
		return l, err
	}
	return net.Listen("tcp", s.BindAddr)
}
