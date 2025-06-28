package tsnet

import (
	"testing"
)

func TestIsTsnetURL(t *testing.T) {
	tests := []struct {
		url      string
		expected bool
	}{
		{"tsnet://my-tailscale-node", true},
		{"tsnet://node1", true},
		{"http://example.com", false},
		{"https://example.com", false},
		{"socks5://proxy.com:1080", false},
		{"tsnet://", true}, // Edge case - empty hostname but valid scheme
		{"", false},
	}

	for _, test := range tests {
		result := IsTsnetURL(test.url)
		if result != test.expected {
			t.Errorf("IsTsnetURL(%s) = %v, expected %v", test.url, result, test.expected)
		}
	}
}

func TestParseTsnetURL(t *testing.T) {
	tests := []struct {
		url          string
		expectedHost string
		expectedPort string
		shouldError  bool
	}{
		{"tsnet://my-tailscale-node", "my-tailscale-node", "", false},
		{"tsnet://node1", "node1", "", false},
		{"tsnet://node1:8080", "node1", "8080", false},
		{"tsnet://server:443", "server", "443", false},
		{"tsnet://proxy:3128", "proxy", "3128", false},
		{"tsnet://host:65535", "host", "65535", false},
		{"tsnet://host:1", "host", "1", false},
		{"http://example.com", "", "", true},
		{"tsnet://", "", "", true}, // Empty hostname should error
		{"tsnet://:8080", "", "", true}, // Empty hostname with port should error
		{"tsnet://host:0", "", "", true}, // Port 0 should error
		{"tsnet://host:65536", "", "", true}, // Port > 65535 should error
		{"tsnet://host:abc", "", "", true}, // Non-numeric port should error
		{"tsnet://host:", "", "", true}, // Empty port should error
		{"", "", "", true},
	}

	for _, test := range tests {
		hostname, port, err := ParseTsnetURL(test.url)
		
		if test.shouldError {
			if err == nil {
				t.Errorf("ParseTsnetURL(%s) expected error but got none", test.url)
			}
		} else {
			if err != nil {
				t.Errorf("ParseTsnetURL(%s) unexpected error: %v", test.url, err)
			}
			if hostname != test.expectedHost {
				t.Errorf("ParseTsnetURL(%s) hostname = %s, expected %s", test.url, hostname, test.expectedHost)
			}
			if port != test.expectedPort {
				t.Errorf("ParseTsnetURL(%s) port = %s, expected %s", test.url, port, test.expectedPort)
			}
		}
	}
}

func TestNewTsnetManager(t *testing.T) {
	manager := NewTsnetManager()
	if manager == nil {
		t.Error("NewTsnetManager() returned nil")
	}
	if manager.servers == nil {
		t.Error("NewTsnetManager() created manager with nil servers map")
	}
}

func TestNewTsnetManagerWithConfig(t *testing.T) {
	authKey := "tskey-test123"
	dataDir := "/tmp/tailscale"
	controlURL := "https://headscale.example.com"
	ephemeral := true
	
	manager := NewTsnetManagerWithConfig(authKey, dataDir, controlURL, ephemeral)
	if manager == nil {
		t.Error("NewTsnetManagerWithConfig() returned nil")
	}
	if manager.authKey != authKey {
		t.Errorf("NewTsnetManagerWithConfig() authKey = %s, expected %s", manager.authKey, authKey)
	}
	if manager.dataDir != dataDir {
		t.Errorf("NewTsnetManagerWithConfig() dataDir = %s, expected %s", manager.dataDir, dataDir)
	}
	if manager.controlURL != controlURL {
		t.Errorf("NewTsnetManagerWithConfig() controlURL = %s, expected %s", manager.controlURL, controlURL)
	}
	if manager.ephemeral != ephemeral {
		t.Errorf("NewTsnetManagerWithConfig() ephemeral = %v, expected %v", manager.ephemeral, ephemeral)
	}
}
