-- ============================================================================
-- Migration: 008_richer_expenses_uptime_locations
-- Commentary: Expands ledger entries, monitor checks, and location metadata
--              without breaking existing records.
-- ============================================================================

CREATE TABLE IF NOT EXISTS expense_entries (
    id TEXT PRIMARY KEY NOT NULL,
    title TEXT NOT NULL,
    category TEXT NOT NULL DEFAULT 'manual',
    amount REAL NOT NULL DEFAULT 0 CHECK (amount >= 0),
    currency TEXT NOT NULL DEFAULT 'MXN',
    occurred_on DATE NOT NULL,
    notes TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE expense_entries ADD COLUMN entry_type TEXT NOT NULL DEFAULT 'expense';
ALTER TABLE expense_entries ADD COLUMN account_name TEXT NOT NULL DEFAULT 'cash';
ALTER TABLE expense_entries ADD COLUMN counterparty TEXT;
ALTER TABLE expense_entries ADD COLUMN scope TEXT NOT NULL DEFAULT 'infrastructure';
ALTER TABLE expense_entries ADD COLUMN due_on DATE;
ALTER TABLE expense_entries ADD COLUMN paid_on DATE;
ALTER TABLE expense_entries ADD COLUMN payment_method TEXT;

UPDATE expense_entries
SET
    entry_type = COALESCE(NULLIF(entry_type, ''), 'expense'),
    account_name = COALESCE(NULLIF(account_name, ''), 'cash'),
    scope = COALESCE(NULLIF(scope, ''), 'infrastructure');

CREATE INDEX IF NOT EXISTS idx_expense_entries_entry_type ON expense_entries(entry_type);
CREATE INDEX IF NOT EXISTS idx_expense_entries_scope ON expense_entries(scope);
CREATE INDEX IF NOT EXISTS idx_expense_entries_account_name ON expense_entries(account_name);
CREATE INDEX IF NOT EXISTS idx_expense_entries_due_on ON expense_entries(due_on);

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

ALTER TABLE uptime_monitors ADD COLUMN http_method TEXT NOT NULL DEFAULT 'GET';
ALTER TABLE uptime_monitors ADD COLUMN expected_status_min INTEGER NOT NULL DEFAULT 200;
ALTER TABLE uptime_monitors ADD COLUMN expected_status_max INTEGER NOT NULL DEFAULT 200;
ALTER TABLE uptime_monitors ADD COLUMN tls_mode TEXT NOT NULL DEFAULT 'skip';
ALTER TABLE uptime_monitors ADD COLUMN request_headers TEXT;
ALTER TABLE uptime_monitors ADD COLUMN expected_body_substring TEXT;

UPDATE uptime_monitors
SET
    http_method = COALESCE(NULLIF(http_method, ''), 'GET'),
    expected_status_min = COALESCE(expected_status_min, expected_status, 200),
    expected_status_max = COALESCE(expected_status_max, expected_status, 200),
    tls_mode = COALESCE(NULLIF(tls_mode, ''), 'skip');

CREATE INDEX IF NOT EXISTS idx_uptime_monitors_http_method ON uptime_monitors(http_method);
CREATE INDEX IF NOT EXISTS idx_uptime_monitors_tls_mode ON uptime_monitors(tls_mode);

CREATE TABLE IF NOT EXISTS locations (
    id TEXT PRIMARY KEY NOT NULL,
    name TEXT NOT NULL UNIQUE,
    city TEXT,
    country TEXT,
    notes TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE locations ADD COLUMN latitude REAL;
ALTER TABLE locations ADD COLUMN longitude REAL;

CREATE INDEX IF NOT EXISTS idx_locations_coordinates ON locations(latitude, longitude);
