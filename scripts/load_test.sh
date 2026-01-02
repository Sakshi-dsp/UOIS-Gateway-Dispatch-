#!/bin/bash

# Load Testing Script for UOIS Gateway
# Tests throughput requirements: minimum 1000 requests/second

set -e

GATEWAY_URL="${GATEWAY_URL:-http://localhost:8080}"
CLIENT_ID="${CLIENT_ID:-test-client}"
CLIENT_SECRET="${CLIENT_SECRET:-test-secret}"
ENDPOINT="${ENDPOINT:-/ondc/status}"
RATE="${RATE:-1000}"
DURATION="${DURATION:-60s}"

echo "Starting load test..."
echo "Gateway URL: $GATEWAY_URL"
echo "Endpoint: $ENDPOINT"
echo "Rate: $RATE req/s"
echo "Duration: $DURATION"
echo ""

# Check if vegeta is installed
if ! command -v vegeta &> /dev/null; then
    echo "Error: vegeta is not installed"
    echo "Install with: go install github.com/tsenart/vegeta/v12@latest"
    exit 1
fi

# Create auth header
AUTH_HEADER=$(echo -n "$CLIENT_ID:$CLIENT_SECRET" | base64)

# Create test payload (minimal valid ONDC request)
PAYLOAD='{
  "context": {
    "domain": "nic2004:60221",
    "country": "IND",
    "city": "std:080",
    "action": "status",
    "core_version": "1.2.0",
    "bap_id": "test-bap",
    "bap_uri": "https://test-bap.com",
    "transaction_id": "test-txn-'$(date +%s)'",
    "message_id": "test-msg-'$(date +%s)'",
    "timestamp": "'$(date -u +"%Y-%m-%dT%H:%M:%S.000Z")'",
    "ttl": "PT30S"
  },
  "message": {
    "order_id": "test-order-123"
  }
}'

# Create vegeta target file
TARGET_FILE=$(mktemp)
echo "POST $GATEWAY_URL$ENDPOINT" > "$TARGET_FILE"
echo "Authorization: Basic $AUTH_HEADER" >> "$TARGET_FILE"
echo "Content-Type: application/json" >> "$TARGET_FILE"
echo "" >> "$TARGET_FILE"
echo "$PAYLOAD" >> "$TARGET_FILE"

# Run load test
echo "Running load test..."
vegeta attack -rate=$RATE -duration=$DURATION -targets="$TARGET_FILE" | \
    vegeta report -type=text

echo ""
echo "Generating detailed report..."
vegeta attack -rate=$RATE -duration=$DURATION -targets="$TARGET_FILE" | \
    vegeta report -type=json > load_test_results.json

echo "Results saved to load_test_results.json"
echo ""
echo "To view latency distribution:"
echo "  vegeta attack -rate=$RATE -duration=$DURATION -targets=\"$TARGET_FILE\" | vegeta report -type=hist[0,100ms,200ms,500ms,1s,2s]"

# Cleanup
rm "$TARGET_FILE"

