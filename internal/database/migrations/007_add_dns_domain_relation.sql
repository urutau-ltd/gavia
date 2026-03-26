-- ============================================================================
-- Migration: 007_add_dns_domain_relation
-- Commentary: Rebuilds dns_records so records can optionally link back to a
--              tracked domain with UUID-compatible TEXT references.
-- ============================================================================

CREATE TABLE IF NOT EXISTS dns_records (
    id TEXT PRIMARY KEY NOT NULL,
    type TEXT NOT NULL CHECK(type IN ('A', 'AAAA', 'CNAME', 'MX', 'TXT', 'NS', 'SOA', 'SRV')),
    hostname TEXT NOT NULL,
    address TEXT NOT NULL,
    notes TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE dns_records_new (
    id TEXT PRIMARY KEY NOT NULL,
    type TEXT NOT NULL CHECK(type IN ('A', 'AAAA', 'CNAME', 'MX', 'TXT', 'NS', 'SOA', 'SRV')),
    hostname TEXT NOT NULL,
    domain_id TEXT,
    address TEXT NOT NULL,
    notes TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (domain_id) REFERENCES domains(id) ON DELETE SET NULL
);

INSERT INTO dns_records_new (
    id,
    type,
    hostname,
    domain_id,
    address,
    notes,
    created_at,
    updated_at
)
SELECT
    id,
    type,
    hostname,
    NULL,
    address,
    notes,
    created_at,
    updated_at
FROM dns_records;

DROP TABLE dns_records;

ALTER TABLE dns_records_new RENAME TO dns_records;

CREATE INDEX IF NOT EXISTS idx_dns_hostname ON dns_records(hostname);
CREATE INDEX IF NOT EXISTS idx_dns_type ON dns_records(type);
CREATE INDEX IF NOT EXISTS idx_dns_domain ON dns_records(domain_id);
