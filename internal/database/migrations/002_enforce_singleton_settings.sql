-- ============================================================================
-- Migration: 002_enforce_singleton_settings
-- Commentary: Rebuilds settings tables so they are true singleton resources.
-- Notes:
--   1. A unique index on a constant expression validates there is at most one
--      row before any rewrite happens.
--   2. If an existing database already contains multiple rows in either table,
--      this migration fails explicitly instead of dropping data silently.
-- ============================================================================

CREATE UNIQUE INDEX IF NOT EXISTS idx_account_settings_singleton_guard ON account_settings((1));
CREATE UNIQUE INDEX IF NOT EXISTS idx_app_settings_singleton_guard ON app_settings((1));

ALTER TABLE account_settings RENAME TO account_settings_old;

CREATE TABLE account_settings (
    id TEXT PRIMARY KEY NOT NULL DEFAULT 'account' CHECK (id = 'account'),
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE,
    settings TEXT NOT NULL DEFAULT '{}',
    notes TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO account_settings (
    id,
    username,
    password_hash,
    email,
    settings,
    notes,
    created_at,
    updated_at
)
SELECT
    'account',
    username,
    password_hash,
    email,
    COALESCE(settings, '{}'),
    notes,
    created_at,
    updated_at
FROM account_settings_old
LIMIT 1;

DROP TABLE account_settings_old;
CREATE UNIQUE INDEX idx_account_settings_singleton ON account_settings((1));

ALTER TABLE app_settings RENAME TO app_settings_old;

CREATE TABLE app_settings (
    id TEXT PRIMARY KEY NOT NULL DEFAULT 'app' CHECK (id = 'app'),
    show_version_footer BOOLEAN NOT NULL DEFAULT 1 CHECK (show_version_footer IN (0, 1)),
    default_server_os TEXT NOT NULL DEFAULT 'Linux',
    default_currency TEXT NOT NULL DEFAULT 'USD',
    due_soon_amount INTEGER NOT NULL DEFAULT 5 CHECK (due_soon_amount >= 0),
    recent_add_amount INTEGER NOT NULL DEFAULT 5 CHECK (recent_add_amount >= 0),
    description TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO app_settings (
    id,
    show_version_footer,
    default_server_os,
    default_currency,
    due_soon_amount,
    recent_add_amount,
    description,
    created_at,
    updated_at
)
SELECT
    'app',
    COALESCE(show_version_footer, 1),
    COALESCE(default_server_os, 'Linux'),
    COALESCE(default_curency, 'USD'),
    COALESCE(due_soon_amount, 5),
    COALESCE(recent_add_amount, 5),
    description,
    created_at,
    updated_at
FROM app_settings_old
LIMIT 1;

DROP TABLE app_settings_old;
CREATE UNIQUE INDEX idx_app_settings_singleton ON app_settings((1));
