-- === PROVIDERS ===
CREATE TABLE IF NOT EXISTS providers (
       id INTEGER PRIMARY KEY AUTOINCREMENT,
       name TEXT NOT NULL UNIQUE,
       website TEXT,
       notes TEXT,
       created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
       updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_providers_name ON providers(name);

-- === LOCATIONS ===
CREATE TABLE IF NOT EXISTS locations (
       id INTEGER PRIMARY KEY AUTOINCREMENT,
       name TEXT NOT NULL UNIQUE,
       city TEXT,
       country TEXT,
       notes TEXT,
       created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
       updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_locations_name ON locations(name);
CREATE INDEX IF NOT EXISTS idx_locations_country ON locations(name);

-- === OS ===
CREATE TABLE IF NOT EXISTS operating_systems (
       id INTEGER PRIMARY KEY AUTOINCREMENT,
       name TEXT NOT NULL UNIQUE,
       notes TEXT,
       created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
       updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- === IPs ===
CREATE TABLE IF NOT EXISTS ips (
       id INTEGER PRIMARY KEY AUTOINCREMENT,
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


CREATE INDEX IF NOT EXISTS idx_ips_address ON ips(address);
CREATE INDEX IF NOT EXISTS idx_ips_type ON ips(type);
CREATE INDEX IF NOT EXISTS idx_ips_country ON ips(country);

CREATE TABLE IF NOT EXISTS dns_records (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    type TEXT NOT NULL CHECK(type IN ('A', 'AAAA', 'CNAME', 'MX', 'TXT', 'NS', 'SOA', 'SRV')),
    hostname TEXT NOT NULL,
    address TEXT NOT NULL,
    notes TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_dns_hostname ON dns_records(hostname);
CREATE INDEX IF NOT EXISTS idx_dns_type ON dns_records(type);

CREATE TABLE IF NOT EXISTS labels (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    notes TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_labels_name ON labels(name);

CREATE TABLE IF NOT EXISTS domains (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    domain TEXT NOT NULL UNIQUE,
    provider_id INTEGER,
    due_date DATE,
    price REAL,
    notes TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (provider_id) REFERENCES providers(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_domains_domain ON domains(domain);
CREATE INDEX IF NOT EXISTS idx_domains_due_date ON domains(due_date);
CREATE INDEX IF NOT EXISTS idx_domains_provider ON domains(provider_id);

CREATE TABLE IF NOT EXISTS hostings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    location_id INTEGER,
    provider_id INTEGER,
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

CREATE TABLE IF NOT EXISTS servers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    hostname TEXT NOT NULL UNIQUE,
    type TEXT NOT NULL,
    os_id INTEGER,
    cpu_cores INTEGER,
    memory_gb INTEGER,
    disk_gb INTEGER,
    location_id INTEGER,
    provider_id INTEGER,
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

CREATE INDEX IF NOT EXISTS idx_servers_hostname ON servers(hostname);
CREATE INDEX IF NOT EXISTS idx_servers_type ON servers(type);
CREATE INDEX IF NOT EXISTS idx_servers_provider ON servers(provider_id);
CREATE INDEX IF NOT EXISTS idx_servers_location ON servers(location_id);
CREATE INDEX IF NOT EXISTS idx_servers_due_date ON servers(due_date);

CREATE TABLE IF NOT EXISTS server_ips (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    server_id INTEGER NOT NULL,
    ip_id INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(server_id, ip_id),
    FOREIGN KEY (server_id) REFERENCES servers(id) ON DELETE CASCADE,
    FOREIGN KEY (ip_id) REFERENCES ips(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_server_ips_server ON server_ips(server_id);
CREATE INDEX IF NOT EXISTS idx_server_ips_ip ON server_ips(ip_id);

CREATE TABLE IF NOT EXISTS server_labels (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    server_id INTEGER NOT NULL,
    label_id INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(server_id, label_id),
    FOREIGN KEY (server_id) REFERENCES servers(id) ON DELETE CASCADE,
    FOREIGN KEY (label_id) REFERENCES labels(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_server_labels_server ON server_labels(server_id);
CREATE INDEX IF NOT EXISTS idx_server_labels_label ON server_labels(label_id);

CREATE TABLE IF NOT EXISTS subscriptions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
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

CREATE INDEX IF NOT EXISTS idx_subscriptions_name ON subscriptions(name);
CREATE INDEX IF NOT EXISTS idx_subscriptions_due_date ON subscriptions(due_date);

CREATE TABLE IF NOT EXISTS account_settings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE,
    settings TEXT DEFAULT '{}',
    notes TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_account_username ON account_settings(username);

CREATE TABLE IF NOT EXISTS app_settings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    show_version_footer BOOLEAN DEFAULT true,
    default_server_os TEXT NOT NULL,
    default_curency TEXT NOT NULL,
    due_soon_amount INT NOT NULL DEFAULT 5,
    recent_add_amount INT NOT NULL DEFAULT 5, 
    description TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Do I really need indexes here?
