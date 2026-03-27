package uptime

import (
	"context"
	"crypto/tls"
	"log/slog"
	"net/http"
	"time"

	uptimemonitor "codeberg.org/urutau-ltd/gavia/internal/models/uptime_monitor"
)

type Service struct {
	logger   *slog.Logger
	repo     *uptimemonitor.Repository
	client   *http.Client
	interval time.Duration
}

func NewService(
	logger *slog.Logger,
	repo *uptimemonitor.Repository,
	client *http.Client,
	interval time.Duration,
) *Service {
	if client == nil {
		client = &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		}
	}
	if interval <= 0 {
		interval = 30 * time.Second
	}

	return &Service{
		logger:   logger,
		repo:     repo,
		client:   client,
		interval: interval,
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

	req, err := http.NewRequestWithContext(timeoutCtx, http.MethodGet, monitor.TargetURL, nil)
	if err != nil {
		message := err.Error()
		result.ErrorText = &message
		return result
	}

	start := time.Now()
	resp, err := s.client.Do(req)
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
	result.OK = resp.StatusCode == monitor.ExpectedStatus
	if !result.OK {
		message := resp.Status
		result.ErrorText = &message
	}

	return result
}
