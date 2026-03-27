package uptime

import (
	"net/http"
	"testing"
	"time"
)

func TestNewServiceDefaultsToTLSIgnoringHTTPClient(t *testing.T) {
	service := NewService(nil, nil, nil, time.Second)

	if service.client == nil {
		t.Fatal("expected service to create a default HTTP client")
	}

	transport, ok := service.client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("expected default client transport to be *http.Transport, got %T", service.client.Transport)
	}

	if transport.TLSClientConfig == nil {
		t.Fatal("expected default transport to configure TLS settings")
	}

	if !transport.TLSClientConfig.InsecureSkipVerify {
		t.Fatal("expected default uptime client to skip TLS verification")
	}
}
