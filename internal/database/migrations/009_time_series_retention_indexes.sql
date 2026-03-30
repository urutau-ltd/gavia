-- ============================================================================
-- Migration: 009_time_series_retention_indexes
-- Commentary: Adds cleanup-friendly indexes for time-series retention jobs.
-- ============================================================================

CREATE INDEX IF NOT EXISTS idx_uptime_monitor_results_checked_at
    ON uptime_monitor_results(checked_at DESC);
