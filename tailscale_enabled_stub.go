//go:build !tailscale

package main

// hasTailscaleSupport indicates whether this binary was built with Tailscale support.
// This constant is used by the update command to select the appropriate binary variant.
const hasTailscaleSupport = false
