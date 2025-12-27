-- Create audit schema
CREATE SCHEMA IF NOT EXISTS audit;

-- Create request_response_logs table
CREATE TABLE IF NOT EXISTS audit.request_response_logs (
    request_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    transaction_id VARCHAR(255),
    message_id VARCHAR(255),
    action VARCHAR(50) NOT NULL,
    request_payload JSONB NOT NULL,
    ack_payload JSONB,
    callback_payload JSONB,
    trace_id VARCHAR(255),
    client_id VARCHAR(255),
    search_id UUID,
    quote_id UUID,
    order_id VARCHAR(255),
    dispatch_order_id VARCHAR(255),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for common queries
CREATE INDEX IF NOT EXISTS idx_request_response_logs_transaction_id ON audit.request_response_logs(transaction_id);
CREATE INDEX IF NOT EXISTS idx_request_response_logs_message_id ON audit.request_response_logs(message_id);
CREATE INDEX IF NOT EXISTS idx_request_response_logs_client_id ON audit.request_response_logs(client_id);
CREATE INDEX IF NOT EXISTS idx_request_response_logs_trace_id ON audit.request_response_logs(trace_id);
CREATE INDEX IF NOT EXISTS idx_request_response_logs_search_id ON audit.request_response_logs(search_id);
CREATE INDEX IF NOT EXISTS idx_request_response_logs_quote_id ON audit.request_response_logs(quote_id);
CREATE INDEX IF NOT EXISTS idx_request_response_logs_order_id ON audit.request_response_logs(order_id);
CREATE INDEX IF NOT EXISTS idx_request_response_logs_created_at ON audit.request_response_logs(created_at);

-- Create callback_delivery_logs table
CREATE TABLE IF NOT EXISTS audit.callback_delivery_logs (
    request_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    callback_url TEXT NOT NULL,
    attempt_no INTEGER NOT NULL DEFAULT 1,
    status VARCHAR(50) NOT NULL,
    error TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for callback delivery logs
CREATE INDEX IF NOT EXISTS idx_callback_delivery_logs_request_id ON audit.callback_delivery_logs(request_id);
CREATE INDEX IF NOT EXISTS idx_callback_delivery_logs_status ON audit.callback_delivery_logs(status);
CREATE INDEX IF NOT EXISTS idx_callback_delivery_logs_created_at ON audit.callback_delivery_logs(created_at);

