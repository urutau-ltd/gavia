-- ============================================================================
-- Migration: 006_fix_server_assignment_indexes
-- Commentary: Restores proper many-to-many cardinality for server IP and
--              server label assignments by removing accidental 1:1 indexes.
--              The tables are created if missing so legacy partial schemas
--              can still complete the migration chain.
-- ============================================================================

CREATE TABLE IF NOT EXISTS server_ips (
    id TEXT PRIMARY KEY NOT NULL,
    server_id TEXT NOT NULL,
    ip_id TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(server_id, ip_id),
    FOREIGN KEY (server_id) REFERENCES servers(id) ON DELETE CASCADE,
    FOREIGN KEY (ip_id) REFERENCES ips(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS server_labels (
    id TEXT PRIMARY KEY NOT NULL,
    server_id TEXT NOT NULL,
    label_id TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(server_id, label_id),
    FOREIGN KEY (server_id) REFERENCES servers(id) ON DELETE CASCADE,
    FOREIGN KEY (label_id) REFERENCES labels(id) ON DELETE CASCADE
);

DROP INDEX IF EXISTS idx_server_ips_server;
DROP INDEX IF EXISTS idx_server_ips_ip;
DROP INDEX IF EXISTS idx_server_labels_server;
DROP INDEX IF EXISTS idx_server_labels_label;

CREATE INDEX IF NOT EXISTS idx_server_ips_server ON server_ips(server_id);
CREATE INDEX IF NOT EXISTS idx_server_ips_ip ON server_ips(ip_id);
CREATE INDEX IF NOT EXISTS idx_server_labels_server ON server_labels(server_id);
CREATE INDEX IF NOT EXISTS idx_server_labels_label ON server_labels(label_id);
