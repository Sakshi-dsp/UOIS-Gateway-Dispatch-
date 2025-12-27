-- Create client_registry schema
CREATE SCHEMA IF NOT EXISTS client_registry;

-- Create clients table
CREATE TABLE IF NOT EXISTS client_registry.clients (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL UNIQUE,
    client_code VARCHAR(50) NOT NULL,
    client_secret_hash TEXT NOT NULL,
    allowed_ips CIDR[],
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

