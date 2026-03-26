package finance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	exchangerate "codeberg.org/urutau-ltd/gavia/internal/models/exchange_rate"
)

type ServiceConfig struct {
	Client             *http.Client
	Interval           time.Duration
	FrankfurterBaseURL string
	CoinGeckoBaseURL   string
}

type Service struct {
	logger             *slog.Logger
	repo               *exchangerate.Repository
	client             *http.Client
	interval           time.Duration
	frankfurterBaseURL string
	coinGeckoBaseURL   string
}

func NewService(logger *slog.Logger, repo *exchangerate.Repository, cfg ServiceConfig) *Service {
	client := cfg.Client
	if client == nil {
		client = &http.Client{Timeout: 12 * time.Second}
	}

	interval := cfg.Interval
	if interval <= 0 {
		interval = 30 * time.Minute
	}

	frankfurterBaseURL := strings.TrimRight(cfg.FrankfurterBaseURL, "/")
	if frankfurterBaseURL == "" {
		frankfurterBaseURL = "https://api.frankfurter.dev"
	}

	coinGeckoBaseURL := strings.TrimRight(cfg.CoinGeckoBaseURL, "/")
	if coinGeckoBaseURL == "" {
		coinGeckoBaseURL = "https://api.coingecko.com"
	}

	return &Service{
		logger:             logger,
		repo:               repo,
		client:             client,
		interval:           interval,
		frankfurterBaseURL: frankfurterBaseURL,
		coinGeckoBaseURL:   coinGeckoBaseURL,
	}
}

func (s *Service) Start(ctx context.Context) {
	if err := s.Refresh(ctx); err != nil && s.logger != nil {
		s.logger.Warn("Initial exchange-rate refresh failed", "err", err)
	}

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.Refresh(ctx); err != nil && s.logger != nil {
				s.logger.Warn("Scheduled exchange-rate refresh failed", "err", err)
			}
		}
	}
}

func (s *Service) Refresh(ctx context.Context) error {
	var errs []error

	if sample, err := s.fetchFrankfurter(ctx); err != nil {
		errs = append(errs, err)
	} else if err := s.repo.Create(ctx, sample); err != nil {
		errs = append(errs, err)
	}

	if sample, err := s.fetchCoinGecko(ctx); err != nil {
		errs = append(errs, err)
	} else if err := s.repo.Create(ctx, sample); err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

func (s *Service) fetchFrankfurter(ctx context.Context) (*exchangerate.Sample, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.frankfurterBaseURL+"/v2/rate/MXN/USD", nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("frankfurter returned %s", resp.Status)
	}

	var payload struct {
		Date  string  `json:"date"`
		Base  string  `json:"base"`
		Quote string  `json:"quote"`
		Rate  float64 `json:"rate"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	observedAt, err := time.Parse(time.DateOnly, payload.Date)
	if err != nil {
		observedAt = time.Now().UTC()
	}

	return &exchangerate.Sample{
		BaseCurrency:  payload.Base,
		QuoteCurrency: payload.Quote,
		Rate:          payload.Rate,
		Source:        "frankfurter",
		ObservedAt:    observedAt.UTC(),
	}, nil
}

func (s *Service) fetchCoinGecko(ctx context.Context) (*exchangerate.Sample, error) {
	url := s.coinGeckoBaseURL + "/api/v3/simple/price?ids=monero&vs_currencies=usd,usdc&include_last_updated_at=true"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("coingecko returned %s", resp.Status)
	}

	var payload struct {
		Monero struct {
			USD           float64 `json:"usd"`
			USDC          float64 `json:"usdc"`
			LastUpdatedAt int64   `json:"last_updated_at"`
		} `json:"monero"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	observedAt := time.Now().UTC()
	if payload.Monero.LastUpdatedAt > 0 {
		observedAt = time.Unix(payload.Monero.LastUpdatedAt, 0).UTC()
	}

	rate := payload.Monero.USD
	source := "coingecko"
	if rate <= 0 && payload.Monero.USDC > 0 {
		rate = payload.Monero.USDC
		source = "coingecko-usdc-fallback"
	}

	if rate <= 0 {
		return nil, fmt.Errorf("coingecko returned no usable monero price in usd or usdc")
	}

	return &exchangerate.Sample{
		BaseCurrency:  "XMR",
		QuoteCurrency: "USD",
		Rate:          rate,
		Source:        source,
		ObservedAt:    observedAt,
	}, nil
}
