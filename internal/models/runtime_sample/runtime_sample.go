package runtimesample

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type Sample struct {
	ID                string    `json:"id"`
	ObservedAt        time.Time `json:"observed_at"`
	Goroutines        int       `json:"goroutines"`
	HeapAllocBytes    uint64    `json:"heap_alloc_bytes"`
	HeapInuseBytes    uint64    `json:"heap_inuse_bytes"`
	HeapSysBytes      uint64    `json:"heap_sys_bytes"`
	TotalAllocBytes   uint64    `json:"total_alloc_bytes"`
	SysBytes          uint64    `json:"sys_bytes"`
	NextGCBytes       uint64    `json:"next_gc_bytes"`
	DBOpenConnections int       `json:"db_open_connections"`
	DBInUse           int       `json:"db_in_use"`
	DBIdle            int       `json:"db_idle"`
	DBWaitCount       int64     `json:"db_wait_count"`
	DBWaitDurationMS  int64     `json:"db_wait_duration_ms"`
	CPUCount          int       `json:"cpu_count"`
	CreatedAt         time.Time `json:"created_at"`
}

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, sample *Sample) error {
	newID, err := uuid.NewV7()
	if err != nil {
		return err
	}

	if sample.ObservedAt.IsZero() {
		sample.ObservedAt = time.Now().UTC()
	}
	sample.ID = newID.String()

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO runtime_samples (
			id,
			observed_at,
			goroutines,
			heap_alloc_bytes,
			heap_inuse_bytes,
			heap_sys_bytes,
			total_alloc_bytes,
			sys_bytes,
			next_gc_bytes,
			db_open_connections,
			db_in_use,
			db_idle,
			db_wait_count,
			db_wait_duration_ms,
			cpu_count
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		sample.ID,
		sample.ObservedAt.UTC(),
		sample.Goroutines,
		sample.HeapAllocBytes,
		sample.HeapInuseBytes,
		sample.HeapSysBytes,
		sample.TotalAllocBytes,
		sample.SysBytes,
		sample.NextGCBytes,
		sample.DBOpenConnections,
		sample.DBInUse,
		sample.DBIdle,
		sample.DBWaitCount,
		sample.DBWaitDurationMS,
		sample.CPUCount,
	)
	return err
}

func (r *Repository) GetRecent(ctx context.Context, limit int) ([]*Sample, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			id,
			observed_at,
			goroutines,
			heap_alloc_bytes,
			heap_inuse_bytes,
			heap_sys_bytes,
			total_alloc_bytes,
			sys_bytes,
			next_gc_bytes,
			db_open_connections,
			db_in_use,
			db_idle,
			db_wait_count,
			db_wait_duration_ms,
			cpu_count,
			created_at
		FROM runtime_samples
		ORDER BY observed_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var samples []*Sample
	for rows.Next() {
		item := &Sample{}
		if err := rows.Scan(
			&item.ID,
			&item.ObservedAt,
			&item.Goroutines,
			&item.HeapAllocBytes,
			&item.HeapInuseBytes,
			&item.HeapSysBytes,
			&item.TotalAllocBytes,
			&item.SysBytes,
			&item.NextGCBytes,
			&item.DBOpenConnections,
			&item.DBInUse,
			&item.DBIdle,
			&item.DBWaitCount,
			&item.DBWaitDurationMS,
			&item.CPUCount,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}
		samples = append(samples, item)
	}

	return samples, rows.Err()
}

func (r *Repository) PruneOlderThan(ctx context.Context, cutoff time.Time) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM runtime_samples
		WHERE observed_at < ?
	`, cutoff.UTC())
	return err
}
