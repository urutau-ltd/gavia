/*
===============================================================================  
File Name: 001_create_tables.sql  
Commentary: Defines the main data structures used by the backend.  
Copyright: 2026 - Urutaú Limited. (Some) rights reserved.
License: AGPL-3.0+
===============================================================================  
*/

-- ============================================================================
-- Table: providers
-- Commentary: Stores service providers. These are related to many other
--             tables in the database.
-- Main fields: id (PK), name (UNIQUE), website, notes
-- ============================================================================
CREATE TABLE IF NOT EXISTS providers (
       id TEXT PRIMARY KEY NOT NULL,
       name TEXT NOT NULL UNIQUE,
       website TEXT,
       notes TEXT,
       created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
       updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

/*
    Trigger: tr_providers_at
    Commentary: Automatically updates the updated_at field when
                a provider registry is changed in the database.
    Event: AFTER UPDATE on the providers table
    Notes: Will run on each update. Haven't tested it's limits or if
           rate-limiting from the Go side is needed to avoid locking
           the database under stress.
*/
CREATE TRIGGER IF NOT EXISTS tr_providers_at
AFTER UPDATE ON providers
BEGIN
    UPDATE providers SET updated_at = CURRENT_TIMESTAMP WHERE id = OLD.id;
END;

-- ============================================================================
-- Table: locations
-- Commentary: Stores location registries used to track or record
--             location details for other Gavia items.
-- Main fields: id (PK), name (UNIQUE), city, country, notes
-- ============================================================================
CREATE TABLE IF NOT EXISTS locations (
       id TEXT PRIMARY KEY NOT NULL,
       name TEXT NOT NULL UNIQUE,
       city TEXT,
       country TEXT,
       notes TEXT,
       created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
       updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

/*  
    Index: idx_locations_country  
    Commentary: Optimiza búsquedas por país en la tabla locations  
    Tabla: locations  
    Columna(s): name
    Tipo: BTREE (por defecto en SQLite)
    Caso de uso: Consultas frecuentes que filtran por nombre de país
    Impacto: Mejora velocidad de búsquedas, requiere espacio adicional  
    Nota: Creado con IF NOT EXISTS para evitar errores en migraciones
*/
CREATE INDEX IF NOT EXISTS idx_locations_country ON locations(name);

-- ============================================================================
-- Table: operating_systems
-- Commentary: Stores OS labels that can be assigned to a server.
-- Main fields: id (PK), name (UNIQUE), notes
-- ============================================================================
CREATE TABLE IF NOT EXISTS operating_systems (
       id TEXT PRIMARY KEY NOT NULL,
       name TEXT NOT NULL UNIQUE,
       notes TEXT,
       created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
       updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================================
-- Table: ips
-- Commentary: Stores IP Addresses
-- Main fields: id (PK), 
--              address (UNIQUE),
--              type ('ipv4'|'ipv6'),
--              city,
--              country,
--              org,
--              asn,
--              isp,
--              notes
-- ============================================================================
CREATE TABLE IF NOT EXISTS ips (
       id TEXT PRIMARY KEY NOT NULL,
       address TEXT NOT NULL UNIQUE,
       type TEXT NOT NULL CHECK(type IN ('ipv4', 'ipv6')),
       city TEXT,
       country TEXT,
       org TEXT,
       asn TEXT,
       isp TEXT,
       notes TEXT,
       created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
       updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);


CREATE INDEX IF NOT EXISTS idx_ips_type ON ips(type);
CREATE INDEX IF NOT EXISTS idx_ips_country ON ips(country);


-- ============================================================================
-- Table: dns_records
-- Commentary: Stores DNS resource records for domain management.
-- Main fields: id (PK), type (A, CNAME, etc), hostname, address
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

CREATE INDEX IF NOT EXISTS idx_dns_hostname ON dns_records(hostname);
CREATE INDEX IF NOT EXISTS idx_dns_type ON dns_records(type);

-- ============================================================================
-- Table: labels
-- Commentary: Generic tagging system for categorizing assets.
-- Main fields: id (PK), name (UNIQUE), notes
-- ============================================================================
CREATE TABLE IF NOT EXISTS labels (
    id TEXT PRIMARY KEY NOT NULL,
    name TEXT NOT NULL UNIQUE,
    notes TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================================
-- Table: domains
-- Commentary: Tracks domain name registrations, expirations and pricing.
-- Main fields: id (PK), domain (UNIQUE), due_date, provider_id
-- ============================================================================
CREATE TABLE IF NOT EXISTS domains (
    id TEXT PRIMARY KEY NOT NULL,
    domain TEXT NOT NULL UNIQUE,
    provider_id TEXT,
    due_date DATE,
    price REAL,
    notes TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (provider_id) REFERENCES providers(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_domains_due_date ON domains(due_date);
CREATE INDEX IF NOT EXISTS idx_domains_provider ON domains(provider_id);

-- ============================================================================
-- Table: hostings
-- Commentary: Stores shared or managed hosting plan information.
-- Main fields: id (PK), name, type, disk_gb, due_date
-- ============================================================================
CREATE TABLE IF NOT EXISTS hostings (
    id TEXT PRIMARY KEY NOT NULL,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    location_id INTEGER,
    provider_id TEXT,
    disk_gb INTEGER,
    domain_id INTEGER,
    price REAL,
    due_date DATE,
    since_date DATE,
    notes TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (location_id) REFERENCES locations(id) ON DELETE SET NULL,
    FOREIGN KEY (provider_id) REFERENCES providers(id) ON DELETE SET NULL,
    FOREIGN KEY (domain_id) REFERENCES domains(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_hostings_name ON hostings(name);
CREATE INDEX IF NOT EXISTS idx_hostings_provider ON hostings(provider_id);
CREATE INDEX IF NOT EXISTS idx_hostings_location ON hostings(location_id);
CREATE INDEX IF NOT EXISTS idx_hostings_due_date ON hostings(due_date);

-- ============================================================================
-- Table: servers
-- Commentary: Defines VPS or Dedicated server specifications and billing.
-- Main fields: id (PK), hostname, cpu_cores, memory_gb, os_id
-- ============================================================================
CREATE TABLE IF NOT EXISTS servers (
    id TEXT PRIMARY KEY NOT NULL,
    hostname TEXT NOT NULL UNIQUE,
    type TEXT NOT NULL,
    os_id INTEGER,
    cpu_cores INTEGER,
    memory_gb INTEGER,
    disk_gb INTEGER,
    location_id INTEGER,
    provider_id TEXT,
    due_date DATE,
    price REAL,
    since_date DATE,
    notes TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (os_id) REFERENCES operating_systems(id) ON DELETE SET NULL,
    FOREIGN KEY (location_id) REFERENCES locations(id) ON DELETE SET NULL,
    FOREIGN KEY (provider_id) REFERENCES providers(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_servers_type ON servers(type);
CREATE INDEX IF NOT EXISTS idx_servers_provider ON servers(provider_id);
CREATE INDEX IF NOT EXISTS idx_servers_location ON servers(location_id);
CREATE INDEX IF NOT EXISTS idx_servers_due_date ON servers(due_date);

-- ============================================================================
-- Table: server_ips
-- Commentary: Pivot table linking servers with their assigned IP addresses.
-- Relationships: Many-to-Many (Servers <-> IPs)
-- ============================================================================
CREATE TABLE IF NOT EXISTS server_ips (
    id TEXT PRIMARY KEY NOT NULL,
    server_id INTEGER NOT NULL,
    ip_id INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(server_id, ip_id),
    FOREIGN KEY (server_id) REFERENCES servers(id) ON DELETE CASCADE,
    FOREIGN KEY (ip_id) REFERENCES ips(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_server_ips_server ON server_ips(server_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_server_ips_ip ON server_ips(ip_id);

-- ============================================================================
-- Table: server_labels
-- Commentary: Pivot table linking servers with classification labels.
-- Relationships: Many-to-Many (Servers <-> Labels)
-- ============================================================================
CREATE TABLE IF NOT EXISTS server_labels (
    id TEXT PRIMARY KEY NOT NULL,
    server_id INTEGER NOT NULL,
    label_id INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(server_id, label_id),
    FOREIGN KEY (server_id) REFERENCES servers(id) ON DELETE CASCADE,
    FOREIGN KEY (label_id) REFERENCES labels(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_server_labels_server ON server_labels(server_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_server_labels_label ON server_labels(label_id);

-- ============================================================================
-- Table: subscriptions
-- Commentary: Tracks recurring SaaS or service subscriptions.
-- Main fields: id (PK), name, renewal_period, due_date
-- ============================================================================
CREATE TABLE IF NOT EXISTS subscriptions (
    id TEXT PRIMARY KEY NOT NULL,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    price REAL,
    due_date DATE,
    since_date DATE,
    renewal_period TEXT,
    notes TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_subscriptions_name ON subscriptions(name);
CREATE UNIQUE INDEX IF NOT EXISTS idx_subscriptions_due_date ON subscriptions(due_date);

-- ============================================================================
-- Table: account_settings
-- Commentary: User-specific configuration and authentication data.
-- Main fields: id (PK), username, password_hash, email
-- ============================================================================
CREATE TABLE IF NOT EXISTS account_settings (
    id TEXT PRIMARY KEY NOT NULL,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE,
    settings TEXT DEFAULT '{}',
    notes TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================================
-- Table: app_settings
-- Commentary: Global application configurations and dashboard preferences.
-- Main fields: id (PK), default_currency, due_soon_amount
-- ============================================================================
CREATE TABLE IF NOT EXISTS app_settings (
    id TEXT PRIMARY KEY NOT NULL,
    show_version_footer BOOLEAN DEFAULT true,
    default_server_os TEXT NOT NULL,
    default_curency TEXT NOT NULL,
    due_soon_amount INT NOT NULL DEFAULT 5,
    recent_add_amount INT NOT NULL DEFAULT 5, 
    description TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
