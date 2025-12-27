-- Create ondc_reference schema
CREATE SCHEMA IF NOT EXISTS ondc_reference;

-- Create order_mapping table
CREATE TABLE IF NOT EXISTS ondc_reference.order_mapping (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    search_id UUID,
    quote_id UUID,
    order_id VARCHAR(255),
    dispatch_order_id VARCHAR(255),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for lookups
CREATE INDEX IF NOT EXISTS idx_order_mapping_search_id ON ondc_reference.order_mapping(search_id);
CREATE INDEX IF NOT EXISTS idx_order_mapping_quote_id ON ondc_reference.order_mapping(quote_id);
CREATE INDEX IF NOT EXISTS idx_order_mapping_order_id ON ondc_reference.order_mapping(order_id);
CREATE INDEX IF NOT EXISTS idx_order_mapping_dispatch_order_id ON ondc_reference.order_mapping(dispatch_order_id);

