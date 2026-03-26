-- ============================================================================
-- Migration: 003_auth_and_settings
-- Commentary: Finalizes account/app settings for authentication and dashboard
--              preferences, and adds persistent user sessions.
-- ============================================================================

ALTER TABLE account_settings RENAME TO account_settings_legacy;

CREATE TABLE account_settings (
    id TEXT PRIMARY KEY NOT NULL DEFAULT 'account' CHECK (id = 'account'),
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    api_token_hash TEXT NOT NULL DEFAULT '',
    api_token_hint TEXT NOT NULL DEFAULT '',
    avatar_path TEXT NOT NULL DEFAULT '/static/img/avatar-1.svg',
    recovery_public_key TEXT NOT NULL DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO account_settings (
    id,
    username,
    password_hash,
    api_token_hash,
    api_token_hint,
    avatar_path,
    recovery_public_key,
    created_at,
    updated_at
)
SELECT
    'account',
    username,
    password_hash,
    '',
    '',
    '/static/img/avatar-1.svg',
    '',
    created_at,
    updated_at
FROM account_settings_legacy
LIMIT 1;

DROP TABLE account_settings_legacy;
CREATE UNIQUE INDEX idx_account_settings_singleton_v2 ON account_settings((1));

ALTER TABLE app_settings RENAME TO app_settings_legacy;

CREATE TABLE app_settings (
    id TEXT PRIMARY KEY NOT NULL DEFAULT 'app' CHECK (id = 'app'),
    show_version_footer BOOLEAN NOT NULL DEFAULT 1 CHECK (show_version_footer IN (0, 1)),
    default_server_os TEXT NOT NULL DEFAULT 'Linux',
    default_currency TEXT NOT NULL DEFAULT 'MXN',
    dashboard_currency TEXT NOT NULL DEFAULT 'MXN',
    dashboard_due_soon_amount INTEGER NOT NULL DEFAULT 5 CHECK (dashboard_due_soon_amount >= 0),
    dashboard_expense_history_json TEXT NOT NULL DEFAULT '[]',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO app_settings (
    id,
    show_version_footer,
    default_server_os,
    default_currency,
    dashboard_currency,
    dashboard_due_soon_amount,
    dashboard_expense_history_json,
    created_at,
    updated_at
)
SELECT
    'app',
    COALESCE(show_version_footer, 1),
    COALESCE(default_server_os, 'Linux'),
    COALESCE(default_currency, 'MXN'),
    COALESCE(default_currency, 'MXN'),
    COALESCE(due_soon_amount, 5),
    '[]',
    created_at,
    updated_at
FROM app_settings_legacy
LIMIT 1;

DROP TABLE app_settings_legacy;
CREATE UNIQUE INDEX idx_app_settings_singleton_v2 ON app_settings((1));

CREATE TABLE IF NOT EXISTS user_sessions (
    id TEXT PRIMARY KEY NOT NULL,
    account_id TEXT NOT NULL DEFAULT 'account' CHECK (account_id = 'account'),
    token_hash TEXT NOT NULL UNIQUE,
    expires_at DATETIME NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (account_id) REFERENCES account_settings(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_user_sessions_expires_at ON user_sessions(expires_at);
