-- ============================================================================
-- Migration: 004_dashboard_and_api
-- Commentary: Adds structured expense entries and migrates any legacy
--              dashboard expense history notes into the new table.
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

CREATE INDEX IF NOT EXISTS idx_expense_entries_occurred_on ON expense_entries(occurred_on DESC);
CREATE INDEX IF NOT EXISTS idx_expense_entries_currency ON expense_entries(currency);

INSERT INTO expense_entries (
    id,
    title,
    category,
    amount,
    currency,
    occurred_on,
    notes,
    created_at,
    updated_at
)
SELECT
    lower(hex(randomblob(16))),
    json_each.value,
    'legacy_note',
    0,
    COALESCE(app_settings.dashboard_currency, 'MXN'),
    DATE('now'),
    'Imported from legacy dashboard expense history',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM app_settings
JOIN json_each(
    CASE
        WHEN json_valid(app_settings.dashboard_expense_history_json)
        THEN app_settings.dashboard_expense_history_json
        ELSE '[]'
    END
)
WHERE app_settings.id = 'app'
  AND json_each.type = 'text'
  AND NOT EXISTS (
      SELECT 1
      FROM expense_entries
      WHERE category = 'legacy_note'
  );

UPDATE app_settings
SET dashboard_expense_history_json = '[]'
WHERE id = 'app'
  AND json_valid(dashboard_expense_history_json);
