-- Create client_registry schema
CREATE SCHEMA IF NOT EXISTS client_registry;

-- Create clients table
CREATE TABLE IF NOT EXISTS client_registry.clients (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL UNIQUE,
    client_code VARCHAR(50) NOT NULL,
    client_secret_hash TEXT NOT NULL,
    api_key_hash TEXT, -- Alias for client_secret_hash (ONDC terminology)
    bap_id VARCHAR(255), -- ONDC Buyer App Provider ID
    bap_uri TEXT, -- ONDC Buyer App Provider URI for callbacks
    allowed_ips CIDR[],
    rate_limit INTEGER DEFAULT 100, -- Requests per window (default: 100)
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
    metadata JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_synced_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_client_registry_client_id ON client_registry.clients(client_id);
CREATE INDEX IF NOT EXISTS idx_client_registry_status ON client_registry.clients(status);
CREATE INDEX IF NOT EXISTS idx_client_registry_client_code ON client_registry.clients(client_code);
CREATE INDEX IF NOT EXISTS idx_client_registry_bap_id ON client_registry.clients(bap_id);

