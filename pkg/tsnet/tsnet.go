package tsnet

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"tailscale.com/tsnet"
)

// TsnetManager manages Tailscale tsnet servers for proxy usage
type TsnetManager struct {
	mu         sync.RWMutex
	servers    map[string]*tsnet.Server
	authKey    string
	dataDir    string
	controlURL string
	ephemeral  bool
}

// NewTsnetManager creates a new Tailscale tsnet manager
func NewTsnetManager() *TsnetManager {
	return &TsnetManager{
		servers: make(map[string]*tsnet.Server),
	}
}

// NewTsnetManagerWithConfig creates a new Tailscale tsnet manager with configuration
func NewTsnetManagerWithConfig(authKey, dataDir, controlURL string, ephemeral bool) *TsnetManager {
	return &TsnetManager{
		servers:    make(map[string]*tsnet.Server),
		authKey:    authKey,
		dataDir:    dataDir,
		controlURL: controlURL,
		ephemeral:  ephemeral,
	}
}

// GetOrCreateServer gets an existing tsnet server or creates a new one
func (tm *TsnetManager) GetOrCreateServer(hostname string) (*tsnet.Server, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if server, exists := tm.servers[hostname]; exists {
		return server, nil
	}

	server := &tsnet.Server{
		Hostname: hostname,
		Logf:     func(format string, args ...interface{}) {
			// Silent logging for now, can be made configurable
		},
	}

	// Set auth key if provided
	if tm.authKey != "" {
		server.AuthKey = tm.authKey
	}

	// Set data directory if provided
	if tm.dataDir != "" {
		server.Dir = tm.dataDir
	}

	// Set control URL if provided
	if tm.controlURL != "" {
		server.ControlURL = tm.controlURL
	}

	// Set ephemeral mode if enabled
	if tm.ephemeral {
		server.Ephemeral = tm.ephemeral
	}

	tm.servers[hostname] = server
	return server, nil
}

// CreateDialer creates a dialer for a specific Tailscale node
func (tm *TsnetManager) CreateDialer(hostname string) (func(network, addr string) (net.Conn, error), error) {
	return tm.CreateDialerWithPort(hostname, "")
}

// CreateDialerWithPort creates a dialer for a specific Tailscale node with optional port override
func (tm *TsnetManager) CreateDialerWithPort(hostname, port string) (func(network, addr string) (net.Conn, error), error) {
	server, err := tm.GetOrCreateServer(hostname)
	if err != nil {
		return nil, fmt.Errorf("failed to get tsnet server: %w", err)
	}

	// Start the server if not already started
	ln, err := server.Listen("tcp", ":0")
	if err != nil {
		return nil, fmt.Errorf("failed to start tsnet server: %w", err)
	}
	ln.Close() // We just needed to start the server

	// Return a dialer function that uses the tsnet server
	return func(network, addr string) (net.Conn, error) {
		// If a specific port was provided in the tsnet URL, override the port in addr
		if port != "" {
			host, _, err := net.SplitHostPort(addr)
			if err != nil {
				// addr might not have a port, treat it as host only
				host = addr
			}
			addr = net.JoinHostPort(host, port)
		}
		return server.Dial(context.Background(), network, addr)
	}, nil
}

// CreateHTTPClient creates an HTTP client that routes through a Tailscale node
func (tm *TsnetManager) CreateHTTPClient(hostname string) (*http.Client, error) {
	dialer, err := tm.CreateDialer(hostname)
	if err != nil {
		return nil, err
	}

	transport := &http.Transport{
		Dial: dialer,
	}

	return &http.Client{
		Transport: transport,
	}, nil
}

// Shutdown closes all tsnet servers
func (tm *TsnetManager) Shutdown() error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	var lastErr error
	for hostname, server := range tm.servers {
		if err := server.Close(); err != nil {
			lastErr = fmt.Errorf("failed to close server %s: %w", hostname, err)
		}
	}

	// Clear the servers map
	tm.servers = make(map[string]*tsnet.Server)
	return lastErr
}

// IsTsnetURL checks if a URL represents a Tailscale node
func IsTsnetURL(url string) bool {
	// Tailscale URLs use the "tsnet://" scheme
	return len(url) >= 8 && url[:8] == "tsnet://"
}

// ParseTsnetURL extracts the hostname and port from a tsnet URL
// Expected format: tsnet://hostname or tsnet://hostname:port
func ParseTsnetURL(url string) (hostname string, port string, err error) {
	if !IsTsnetURL(url) {
		return "", "", fmt.Errorf("not a tsnet URL: %s", url)
	}

	hostPort := url[8:] // Remove "tsnet://" prefix
	if hostPort == "" {
		return "", "", fmt.Errorf("empty hostname in tsnet URL: %s", url)
	}

	// Check if port is specified
	if colonIndex := strings.LastIndex(hostPort, ":"); colonIndex != -1 {
		hostname = hostPort[:colonIndex]
		port = hostPort[colonIndex+1:]
		
		// Validate hostname is not empty
		if hostname == "" {
			return "", "", fmt.Errorf("empty hostname in tsnet URL: %s", url)
		}
		
		// Validate port is numeric
		if port == "" {
			return "", "", fmt.Errorf("empty port in tsnet URL: %s", url)
		}
		
		// Basic port validation (1-65535)
		if portNum, parseErr := strconv.Atoi(port); parseErr != nil || portNum < 1 || portNum > 65535 {
			return "", "", fmt.Errorf("invalid port in tsnet URL: %s", url)
		}
	} else {
		hostname = hostPort
		port = "" // No port specified
	}

	return hostname, port, nil
}
