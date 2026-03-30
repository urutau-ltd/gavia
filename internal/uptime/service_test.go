package uptime

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	uptimemonitor "codeberg.org/urutau-ltd/gavia/internal/models/uptime_monitor"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return fn(r)
}

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

	secureTransport, ok := service.secureClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("expected secure client transport to be *http.Transport, got %T", service.secureClient.Transport)
	}

	if secureTransport.TLSClientConfig == nil || secureTransport.TLSClientConfig.InsecureSkipVerify {
		t.Fatal("expected secure uptime client to verify TLS certificates")
	}
}

func TestCheckMonitorSupportsHeadersBodyMatchAndStatusRanges(t *testing.T) {
	client := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST request, got %s", r.Method)
			}

			if got := r.Header.Get("X-Env"); got != "prod" {
				t.Fatalf("expected X-Env header to be forwarded, got %q", got)
			}

			return &http.Response{
				StatusCode: http.StatusAccepted,
				Status:     "202 Accepted",
				Body:       io.NopCloser(strings.NewReader("healthy=ok")),
				Header:     make(http.Header),
				Request:    r,
			}, nil
		}),
	}

	service := NewService(nil, nil, client, time.Second)
	monitor := &uptimemonitor.Monitor{
		ID:                    "monitor-1",
		TargetURL:             "https://example.test/health",
		HTTPMethod:            http.MethodPost,
		ExpectedStatusMin:     200,
		ExpectedStatusMax:     299,
		RequestHeaders:        pointer("X-Env: prod"),
		ExpectedBodySubstring: pointer("healthy=ok"),
		TimeoutMS:             2000,
		TLSMode:               "skip",
	}

	result := service.checkMonitor(context.Background(), monitor)
	if !result.OK {
		t.Fatalf("expected monitor check to succeed, got %+v", result)
	}

	if result.StatusCode == nil || *result.StatusCode != http.StatusAccepted {
		t.Fatalf("expected recorded status code 202, got %+v", result.StatusCode)
	}

	if result.LatencyMS == nil {
		t.Fatal("expected monitor check to record latency")
	}
}

func pointer(value string) *string {
	return &value
}
