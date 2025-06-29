package mubeng

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"

	"github.com/mubeng/mubeng/pkg/helper/awsurl"
	"github.com/mubeng/mubeng/pkg/tsnet"
	"h12.io/socks"
)

// Transport to auto-switch transport between HTTP/S or SOCKS v4(A) & v5 proxies.
//
// Depending on the protocol scheme, returning value of [http.Transport] with
// [http.Transport.Dialer] or [http.Transport.Proxy]. If protocol scheme is "aws",
// it will return default [http.Transport]. If protocol scheme is "tsnet",
// it will create a transport that routes through a Tailscale node.
func Transport(p string) (*http.Transport, error) {
	var proxyURL *url.URL
	var err error

	tr := new(http.Transport)

	if awsurl.IsURL(p) {
		return tr, fmt.Errorf("%w: %w", ErrUnsupportedProxyProtocolScheme, ErrSwitchTransportAWSProtocolScheme)
	}

	// Check if it's a tsnet URL
	if tsnet.IsTsnetURL(p) {
		hostname, port, err := tsnet.ParseTsnetURL(p)
		if err != nil {
			return nil, fmt.Errorf("invalid tsnet URL: %w", err)
		}

		// Use the global tsnet manager or create a new one if not initialized
		if TsnetManager == nil {
			TsnetManager = tsnet.NewTsnetManager()
		}

		// Create a dialer that routes through Tailscale to the target hostname
		baseDialer, err := TsnetManager.CreateDialerWithPort(port)
		if err != nil {
			return nil, fmt.Errorf("failed to create tsnet dialer: %w", err)
		}

		// Wrap the dialer to always connect to the specified hostname
		tr.Dial = func(network, addr string) (net.Conn, error) {
			// Replace the host part with our target hostname
			_, originalPort, err := net.SplitHostPort(addr)
			if err != nil {
				// If no port specified, use the port from tsnet URL or default
				if port != "" {
					originalPort = port
				} else {
					originalPort = "80" // Default port
				}
			} else if port != "" {
				// Override with port from tsnet URL if specified
				originalPort = port
			}
			
			targetAddr := net.JoinHostPort(hostname, originalPort)
			return baseDialer(network, targetAddr)
		}
	} else {
		proxyURL, err = url.Parse(p)
		if err != nil {
			return nil, err
		}

		switch proxyURL.Scheme {
		case "socks4", "socks4a", "socks5":
			// TODO(dwisiswant0): deprecated, update this later.
			// nolint: staticcheck
			tr.Dial = socks.Dial(p)
		case "http", "https":
			tr.Proxy = http.ProxyURL(proxyURL)
		default:
			return nil, fmt.Errorf("%w: %s", ErrUnsupportedProxyProtocolScheme, proxyURL.Scheme)
		}
	}

	tr.DisableKeepAlives = true
	tr.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true,
		MinVersion:         tls.VersionTLS10,
		CipherSuites:       getUnsafeCipherSuites(),
	}

	return tr, nil
}
