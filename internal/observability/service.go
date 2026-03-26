package observability

import (
	"context"
	"database/sql"
	"log/slog"
	"runtime"
	"time"

	runtimesample "codeberg.org/urutau-ltd/gavia/internal/models/runtime_sample"
)

type Service struct {
	logger   *slog.Logger
	db       *sql.DB
	repo     *runtimesample.Repository
	interval time.Duration
}

func NewService(
	logger *slog.Logger,
	db *sql.DB,
	repo *runtimesample.Repository,
	interval time.Duration,
) *Service {
	if interval <= 0 {
		interval = 30 * time.Second
	}

	return &Service{
		logger:   logger,
		db:       db,
		repo:     repo,
		interval: interval,
	}
}

func (s *Service) Start(ctx context.Context) {
	if err := s.Collect(ctx); err != nil && s.logger != nil {
		s.logger.Warn("Initial runtime sample failed", "err", err)
	}

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.Collect(ctx); err != nil && s.logger != nil {
				s.logger.Warn("Scheduled runtime sample failed", "err", err)
			}
		}
	}
}

func (s *Service) Collect(ctx context.Context) error {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	dbStats := s.db.Stats()

	return s.repo.Create(ctx, &runtimesample.Sample{
		ObservedAt:        time.Now().UTC(),
		Goroutines:        runtime.NumGoroutine(),
		HeapAllocBytes:    mem.HeapAlloc,
		HeapInuseBytes:    mem.HeapInuse,
		HeapSysBytes:      mem.HeapSys,
		TotalAllocBytes:   mem.TotalAlloc,
		SysBytes:          mem.Sys,
		NextGCBytes:       mem.NextGC,
		DBOpenConnections: dbStats.OpenConnections,
		DBInUse:           dbStats.InUse,
		DBIdle:            dbStats.Idle,
		DBWaitCount:       dbStats.WaitCount,
		DBWaitDurationMS:  dbStats.WaitDuration.Milliseconds(),
		CPUCount:          runtime.NumCPU(),
	})
}
