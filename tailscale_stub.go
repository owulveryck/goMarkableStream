//go:build !tailscale

package main

import (
	"context"
	"net"
)

// TailscaleManager stub when Tailscale is not compiled in
type TailscaleManager struct{}

// NewTailscaleManager returns nil when Tailscale support is not compiled in
func NewTailscaleManager(cfg *configuration) *TailscaleManager {
	return nil
}

// Start is a stub that returns nil when Tailscale is not compiled in
func (tm *TailscaleManager) Start(ctx context.Context) (net.Listener, error) {
	return nil, nil
}

// Close is a stub that returns nil when Tailscale is not compiled in
func (tm *TailscaleManager) Close() error {
	return nil
}

// GetFunnelInfo is a stub that returns empty values when Tailscale is not compiled in
func (tm *TailscaleManager) GetFunnelInfo() (bool, string, error) {
	return false, "", nil
}

// ToggleFunnel is a stub that returns nil when Tailscale is not compiled in
func (tm *TailscaleManager) ToggleFunnel(enable bool) (net.Listener, error) {
	return nil, nil
}

// GetListener is a stub that returns nil when Tailscale is not compiled in
func (tm *TailscaleManager) GetListener() net.Listener {
	return nil
}

// StartAsync is a stub when Tailscale is not compiled in
func (tm *TailscaleManager) StartAsync(ctx context.Context) {}

// Ready is a stub that returns nil when Tailscale is not compiled in
func (tm *TailscaleManager) Ready() <-chan struct{} {
	return nil
}

// IsReady is a stub that returns false when Tailscale is not compiled in
func (tm *TailscaleManager) IsReady() bool {
	return false
}
