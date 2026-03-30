package uptime

import (
	"context"
	"crypto/tls"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	uptimemonitor "codeberg.org/urutau-ltd/gavia/internal/models/uptime_monitor"
)

type Service struct {
	logger       *slog.Logger
	repo         *uptimemonitor.Repository
	client       *http.Client
	secureClient *http.Client
	interval     time.Duration
}

func NewService(
	logger *slog.Logger,
	repo *uptimemonitor.Repository,
	client *http.Client,
	interval time.Duration,
) *Service {
	if client == nil {
		client = newDefaultHTTPClient(true)
	}
	if interval <= 0 {
		interval = 30 * time.Second
	}

	secureClient := cloneHTTPClient(client, false)
	return &Service{
		logger:       logger,
		repo:         repo,
		client:       client,
		secureClient: secureClient,
		interval:     interval,
	}
}

func (s *Service) Start(ctx context.Context) {
	if err := s.RunDueChecks(ctx); err != nil && s.logger != nil {
		s.logger.Warn("Initial uptime checks failed", "err", err)
	}

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.RunDueChecks(ctx); err != nil && s.logger != nil {
				s.logger.Warn("Scheduled uptime checks failed", "err", err)
			}
		}
	}
}

func (s *Service) RunDueChecks(ctx context.Context) error {
	statuses, err := s.repo.GetAll(ctx, 0)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	for _, status := range statuses {
		if status == nil || !status.Enabled {
			continue
		}

		if status.LastCheckedAt != nil {
			nextRunAt := status.LastCheckedAt.Add(time.Duration(status.CheckIntervalSeconds) * time.Second)
			if now.Before(nextRunAt) {
				continue
			}
		}

		result := s.checkMonitor(ctx, &status.Monitor)
		if err := s.repo.CreateResult(ctx, result); err != nil {
			if s.logger != nil {
				s.logger.Warn("Could not persist uptime result", "monitor_id", status.ID, "err", err)
			}
		}
	}

	return nil
}

func (s *Service) checkMonitor(ctx context.Context, monitor *uptimemonitor.Monitor) *uptimemonitor.Result {
	result := &uptimemonitor.Result{
		MonitorID: monitor.ID,
		CheckedAt: time.Now().UTC(),
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(monitor.TimeoutMS)*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(timeoutCtx, monitor.HTTPMethodValue(), monitor.TargetURL, nil)
	if err != nil {
		message := err.Error()
		result.ErrorText = &message
		return result
	}

	for key, values := range parseRequestHeaders(monitor.RequestHeadersValue()) {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	start := time.Now()
	resp, err := s.clientForMonitor(monitor).Do(req)
	latency := int(time.Since(start).Milliseconds())
	result.LatencyMS = &latency
	if err != nil {
		message := err.Error()
		result.ErrorText = &message
		return result
	}
	defer resp.Body.Close()

	statusCode := resp.StatusCode
	result.StatusCode = &statusCode
	minimum, maximum := monitorStatusRange(monitor)
	result.OK = statusCode >= minimum && statusCode <= maximum
	if !result.OK {
		message := "expected HTTP " + monitor.StatusRangeDisplay() + ", got " + resp.Status
		result.ErrorText = &message
		return result
	}

	expectedBody := strings.TrimSpace(monitor.ExpectedBodySubstringValue())
	if expectedBody == "" {
		return result
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		message := err.Error()
		result.ErrorText = &message
		result.OK = false
		return result
	}
	if !strings.Contains(string(body), expectedBody) {
		message := "response body did not contain the expected text"
		result.ErrorText = &message
		result.OK = false
	}

	return result
}

func (s *Service) RunMonitorNow(ctx context.Context, id string) error {
	monitor, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if monitor == nil {
		return nil
	}

	return s.repo.CreateResult(ctx, s.checkMonitor(ctx, monitor))
}

func (s *Service) clientForMonitor(monitor *uptimemonitor.Monitor) *http.Client {
	if monitor != nil && monitor.TLSModeValue() == "verify" && s.secureClient != nil {
		return s.secureClient
	}
	return s.client
}

func newDefaultHTTPClient(skipTLSVerify bool) *http.Client {
	return &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: skipTLSVerify,
			},
		},
	}
}

func cloneHTTPClient(base *http.Client, skipTLSVerify bool) *http.Client {
	if base == nil {
		return newDefaultHTTPClient(skipTLSVerify)
	}

	cloned := *base
	if transport, ok := base.Transport.(*http.Transport); ok && transport != nil {
		nextTransport := transport.Clone()
		if nextTransport.TLSClientConfig == nil {
			nextTransport.TLSClientConfig = &tls.Config{}
		} else {
			nextTransport.TLSClientConfig = nextTransport.TLSClientConfig.Clone()
		}
		nextTransport.TLSClientConfig.InsecureSkipVerify = skipTLSVerify
		cloned.Transport = nextTransport
	}
	return &cloned
}

func parseRequestHeaders(raw string) http.Header {
	headers := http.Header{}
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			continue
		}
		headers.Add(key, value)
	}
	return headers
}

func monitorStatusRange(monitor *uptimemonitor.Monitor) (int, int) {
	if monitor == nil {
		return 200, 200
	}

	fallback := monitor.ExpectedStatus
	if fallback <= 0 {
		fallback = 200
	}

	minimum := monitor.ExpectedStatusMin
	if minimum <= 0 {
		minimum = fallback
	}

	maximum := monitor.ExpectedStatusMax
	if maximum <= 0 {
		maximum = fallback
	}

	if maximum < minimum {
		maximum = minimum
	}

	return minimum, maximum
}
