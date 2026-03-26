-- ============================================================================
-- Migration: 005_observability_and_uptime
-- Commentary: Adds financial currencies, exchange-rate history, runtime
--              diagnostics samples, and lightweight uptime monitors.
-- ============================================================================

ALTER TABLE domains ADD COLUMN currency TEXT NOT NULL DEFAULT '';
ALTER TABLE hostings ADD COLUMN currency TEXT NOT NULL DEFAULT '';
ALTER TABLE servers ADD COLUMN currency TEXT NOT NULL DEFAULT '';
ALTER TABLE subscriptions ADD COLUMN currency TEXT NOT NULL DEFAULT '';

UPDATE domains
SET currency = COALESCE(NULLIF(currency, ''), (SELECT default_currency FROM app_settings WHERE id = 'app'), 'MXN');

UPDATE hostings
SET currency = COALESCE(NULLIF(currency, ''), (SELECT default_currency FROM app_settings WHERE id = 'app'), 'MXN');

UPDATE servers
SET currency = COALESCE(NULLIF(currency, ''), (SELECT default_currency FROM app_settings WHERE id = 'app'), 'MXN');

UPDATE subscriptions
SET currency = COALESCE(NULLIF(currency, ''), (SELECT default_currency FROM app_settings WHERE id = 'app'), 'MXN');

CREATE INDEX IF NOT EXISTS idx_domains_currency ON domains(currency);
CREATE INDEX IF NOT EXISTS idx_hostings_currency ON hostings(currency);
CREATE INDEX IF NOT EXISTS idx_servers_currency ON servers(currency);
CREATE INDEX IF NOT EXISTS idx_subscriptions_currency ON subscriptions(currency);

CREATE TABLE IF NOT EXISTS exchange_rate_samples (
    id TEXT PRIMARY KEY NOT NULL,
    base_currency TEXT NOT NULL,
    quote_currency TEXT NOT NULL,
    rate REAL NOT NULL CHECK (rate > 0),
    source TEXT NOT NULL,
    observed_at DATETIME NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_exchange_rate_pair_time
    ON exchange_rate_samples(base_currency, quote_currency, observed_at DESC);

CREATE TABLE IF NOT EXISTS runtime_samples (
    id TEXT PRIMARY KEY NOT NULL,
    observed_at DATETIME NOT NULL,
    goroutines INTEGER NOT NULL,
    heap_alloc_bytes INTEGER NOT NULL,
    heap_inuse_bytes INTEGER NOT NULL,
    heap_sys_bytes INTEGER NOT NULL,
    total_alloc_bytes INTEGER NOT NULL,
    sys_bytes INTEGER NOT NULL,
    next_gc_bytes INTEGER NOT NULL,
    db_open_connections INTEGER NOT NULL,
    db_in_use INTEGER NOT NULL,
    db_idle INTEGER NOT NULL,
    db_wait_count INTEGER NOT NULL,
    db_wait_duration_ms INTEGER NOT NULL,
    cpu_count INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_runtime_samples_observed_at
    ON runtime_samples(observed_at DESC);

CREATE TABLE IF NOT EXISTS uptime_monitors (
    id TEXT PRIMARY KEY NOT NULL,
    name TEXT NOT NULL,
    target_url TEXT NOT NULL UNIQUE,
    kind TEXT NOT NULL DEFAULT 'http' CHECK (kind IN ('http')),
    expected_status INTEGER NOT NULL DEFAULT 200 CHECK (expected_status >= 100 AND expected_status <= 599),
    check_interval_seconds INTEGER NOT NULL DEFAULT 300 CHECK (check_interval_seconds >= 30),
    timeout_ms INTEGER NOT NULL DEFAULT 5000 CHECK (timeout_ms >= 250),
    enabled BOOLEAN NOT NULL DEFAULT 1 CHECK (enabled IN (0, 1)),
    notes TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_uptime_monitors_enabled ON uptime_monitors(enabled);

CREATE TABLE IF NOT EXISTS uptime_monitor_results (
    id TEXT PRIMARY KEY NOT NULL,
    monitor_id TEXT NOT NULL,
    checked_at DATETIME NOT NULL,
    ok BOOLEAN NOT NULL CHECK (ok IN (0, 1)),
    status_code INTEGER,
    latency_ms INTEGER,
    error_text TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (monitor_id) REFERENCES uptime_monitors(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_uptime_monitor_results_monitor_time
    ON uptime_monitor_results(monitor_id, checked_at DESC);
