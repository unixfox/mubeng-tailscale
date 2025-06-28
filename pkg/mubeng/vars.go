package mubeng

import "github.com/mubeng/mubeng/pkg/tsnet"

// HopHeaders are meaningful only for a single transport-level connection, and are not stored by caches or forwarded by proxies.
var HopHeaders = []string{
	"Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Proxy-Connection",
	"Te", // canonicalized version of "TE"
	"Trailers",
	"Transfer-Encoding",
	"Upgrade",
}

// Global tsnet manager for managing Tailscale connections
var TsnetManager *tsnet.TsnetManager
