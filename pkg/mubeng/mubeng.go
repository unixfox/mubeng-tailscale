package mubeng

import (
	"net"
	"net/http"
	"net/url"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/mubeng/mubeng/pkg/helper/awsurl"
	"github.com/mubeng/mubeng/pkg/tsnet"
)

// New define HTTP client request of the [http.Request] itself.
//
// also removes Hop-by-hop headers when it is sent to backend (see http://www.w3.org/Protocols/rfc2616/rfc2616-sec13.html),
// then add X-Forwarded-For header value with the IP address value of rotator proxy IP.
func (proxy *Proxy) New(req *http.Request) (*http.Client, error) {
	client := &http.Client{
		CheckRedirect: proxy.redirectPolicy,
		Timeout:       proxy.Timeout,
		Transport:     proxy.Transport,
	}

	// http: Request.RequestURI can't be set in client requests.
	// http://golang.org/src/pkg/net/http/client.go
	req.RequestURI = ""

	for _, h := range HopHeaders {
		req.Header.Del(h)
	}

	req.Header.Set("X-Forwarded-Proto", req.URL.Scheme)

	// if the proxy address is an AWS URL, return early.
	if awsurl.IsURL(proxy.Address) {
		return client, nil
	}

	// if the proxy address is a tsnet URL, return early.
	if tsnet.IsTsnetURL(proxy.Address) {
		return client, nil
	}

	proxyURL, err := url.Parse(proxy.Address)
	if err != nil {
		return client, err
	}

	if host, _, err := net.SplitHostPort(proxyURL.Host); err == nil {
		// if prior, ok := req.Header["X-Forwarded-For"]; ok {
		// 	host = strings.Join(prior, ", ") + ", " + host
		// }
		req.Header.Set("X-Forwarded-For", host)
	}

	return client, nil
}

// redirectPolicy determines if a request should be redirected.
//
// It checks if the number of redirects has exceeded the maximum allowed by the
// proxy. If so, it returns [http.ErrUseLastResponse] to indicate that the last
// response should be used. Otherwise, it returns nil to allow the redirect to
// proceed.
func (proxy *Proxy) redirectPolicy(req *http.Request, via []*http.Request) error {
	if len(via) >= proxy.MaxRedirects {
		return http.ErrUseLastResponse
	}

	return nil
}

// ToRetryableHTTPClient converts standard [http.Client] to [retryablehttp.Client]
func ToRetryableHTTPClient(client *http.Client) *retryablehttp.Client {
	retryablehttpClient := retryablehttp.NewClient()
	retryablehttpClient.HTTPClient = client

	return retryablehttpClient
}

// InitTsnetManager initializes the global TsnetManager with configuration
func InitTsnetManager(authKey, dataDir, controlURL string, ephemeral bool) {
	if authKey != "" || dataDir != "" || controlURL != "" || ephemeral {
		TsnetManager = tsnet.NewTsnetManagerWithConfig(authKey, dataDir, controlURL, ephemeral)
	} else {
		TsnetManager = tsnet.NewTsnetManager()
	}
}
