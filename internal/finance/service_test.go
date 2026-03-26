package finance

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestFetchCoinGeckoPrefersUSD(t *testing.T) {
	service := NewService(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		nil,
		ServiceConfig{
			Client: &http.Client{
				Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
					if got := r.URL.Query().Get("vs_currencies"); got != "usd,usdc" {
						t.Fatalf("expected vs_currencies=usd,usdc, got %q", got)
					}

					return jsonResponse(`{"monero":{"usd":215.5,"usdc":215.4,"last_updated_at":1711459200}}`), nil
				}),
			},
			CoinGeckoBaseURL: "https://example.invalid",
		},
	)

	sample, err := service.fetchCoinGecko(context.Background())
	if err != nil {
		t.Fatalf("fetchCoinGecko returned error: %v", err)
	}

	if sample == nil {
		t.Fatal("expected sample, got nil")
	}

	if sample.Rate != 215.5 {
		t.Fatalf("expected usd rate 215.5, got %f", sample.Rate)
	}

	if sample.Source != "coingecko" {
		t.Fatalf("expected source %q, got %q", "coingecko", sample.Source)
	}

	if sample.ObservedAt != time.Unix(1711459200, 0).UTC() {
		t.Fatalf("expected observed_at from payload, got %v", sample.ObservedAt)
	}
}

func TestFetchCoinGeckoFallsBackToUSDC(t *testing.T) {
	service := NewService(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		nil,
		ServiceConfig{
			Client: &http.Client{
				Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
					return jsonResponse(`{"monero":{"usd":0,"usdc":214.9,"last_updated_at":1711459200}}`), nil
				}),
			},
			CoinGeckoBaseURL: "https://example.invalid",
		},
	)

	sample, err := service.fetchCoinGecko(context.Background())
	if err != nil {
		t.Fatalf("fetchCoinGecko returned error: %v", err)
	}

	if sample == nil {
		t.Fatal("expected sample, got nil")
	}

	if sample.Rate != 214.9 {
		t.Fatalf("expected usdc fallback rate 214.9, got %f", sample.Rate)
	}

	if sample.Source != "coingecko-usdc-fallback" {
		t.Fatalf("expected source %q, got %q", "coingecko-usdc-fallback", sample.Source)
	}

	if sample.QuoteCurrency != "USD" {
		t.Fatalf("expected quote currency to remain USD for compatibility, got %q", sample.QuoteCurrency)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return fn(r)
}

func jsonResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
