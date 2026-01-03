# End-to-End Integration Testing for UOIS Gateway

## Overview

This integration test framework provides complete ONDC → UOIS → internal services flow coverage, with all testing assets living inside the UOIS repository itself.

**Context**: Dispatch is a **logistics seller NP (BPP)** for **P2P (Point-to-Point) delivery only**. All mock payloads and tests reflect P2P delivery characteristics.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│              Integration Test Script                        │
│         (scripts/test-e2e-integration.go)                  │
└─────────────────────────────────────────────────────────────┘
                          │
        ┌─────────────────┼─────────────────┐
        │                 │                 │
        ▼                 ▼                 ▼
┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│  ONDC HTTP   │  │ Redis Streams│  │ Mock Payloads│
│   Requests   │  │   Events     │  │  (testdata/) │
└──────────────┘  └──────────────┘  └──────────────┘
        │                 │                 │
        └─────────────────┼─────────────────┘
                          │
                          ▼
              ┌───────────────────────┐
              │   UOIS Gateway        │
              │   (Test Target)       │
              └───────────────────────┘
```

## Test Coverage

### 1. Event Publishing Tests
Validates that UOIS Gateway publishes events correctly:
- `SEARCH_REQUESTED` → `stream.location.search` → Location Service
- `INIT_REQUESTED` → `stream.uois.init_requested` → Order Service
- `CONFIRM_REQUESTED` → `stream.uois.confirm_requested` → Order Service

### 2. Event Consumption Tests
Validates that UOIS Gateway consumes events correctly:
- `QUOTE_COMPUTED` ← `quote:computed` ← Quote Service
- `QUOTE_CREATED` ← `stream.uois.quote_created` ← Order Service
- `QUOTE_INVALIDATED` ← `stream.uois.quote_invalidated` ← Order Service
- `ORDER_CONFIRMED` ← `stream.uois.order_confirmed` ← Order Service
- `ORDER_CONFIRM_FAILED` ← `stream.uois.order_confirm_failed` ← Order Service
- `client.*` events ← `stream:admin.client.events` ← Admin Service

### 3. End-to-End Flow Tests

#### `/search` Flow
1. Send ONDC `/search` request to UOIS Gateway
2. UOIS Gateway publishes `SEARCH_REQUESTED` event
3. Simulate `QUOTE_COMPUTED` event from Quote Service
4. UOIS Gateway consumes `QUOTE_COMPUTED` and sends `/on_search` callback

#### `/init` Flow
1. Send ONDC `/init` request to UOIS Gateway
2. UOIS Gateway publishes `INIT_REQUESTED` event
3. Simulate `QUOTE_CREATED` event from Order Service
4. UOIS Gateway consumes `QUOTE_CREATED` and sends `/on_init` callback

#### `/confirm` Flow
1. Send ONDC `/confirm` request to UOIS Gateway
2. UOIS Gateway publishes `CONFIRM_REQUESTED` event
3. Simulate `ORDER_CONFIRMED` event from Order Service
4. UOIS Gateway consumes `ORDER_CONFIRMED` and sends `/on_confirm` callback

## Prerequisites

1. **UOIS Gateway** running on `http://localhost:8080` (or set `UOIS_BASE_URL`)
2. **Redis** running on `localhost:6379` (or set `REDIS_ADDR`)
3. **Go 1.21+** installed

## Running the Tests

### Basic Usage

```bash
cd UOIS-Gateway-Dispatch
go run scripts/test-e2e-integration.go
```

### With Custom Configuration

```bash
export UOIS_BASE_URL="http://localhost:8080"
export REDIS_ADDR="localhost:6379"
export REDIS_PASSWORD="your_password"

go run scripts/test-e2e-integration.go
```

### As a Test Binary

```bash
# Build the test binary
go build -o bin/test-e2e-integration scripts/test-e2e-integration.go

# Run the binary
./bin/test-e2e-integration
```

## Mock Payloads

All mock payloads are stored in `testdata/mocks/` directory:

### ONDC Request Payloads
- `testdata/mocks/ondc/requests/search.json` - ONDC `/search` request (P2P delivery)
- `testdata/mocks/ondc/requests/init.json` - ONDC `/init` request (P2P delivery)
- `testdata/mocks/ondc/requests/confirm.json` - ONDC `/confirm` request (P2P delivery)

### ONDC Callback Payloads
- `testdata/mocks/ondc/callbacks/on_search.json` - Expected `/on_search` callback
- `testdata/mocks/ondc/callbacks/on_init.json` - Expected `/on_init` callback
- `testdata/mocks/ondc/callbacks/on_confirm.json` - Expected `/on_confirm` callback

### Event Payloads

**Published by UOIS Gateway:**
- `testdata/mocks/events/published/search_requested.json`
- `testdata/mocks/events/published/init_requested.json`
- `testdata/mocks/events/published/confirm_requested.json`

**Consumed by UOIS Gateway:**
- `testdata/mocks/events/consumed/quote_computed.json`
- `testdata/mocks/events/consumed/quote_created.json`
- `testdata/mocks/events/consumed/quote_invalidated.json`
- `testdata/mocks/events/consumed/order_confirmed.json`
- `testdata/mocks/events/consumed/order_confirm_failed.json`
- `testdata/mocks/events/consumed/client_created.json`

## P2P Delivery Characteristics

All mock payloads reflect **P2P (Point-to-Point) delivery**:

- **Fulfillment Type**: `"Delivery"` (not hub-based)
- **Item Descriptor Code**: `"P2P"` (not `"P2H2P"`)
- **AWB Number**: Not required for P2P (only for P2H2P)
- **Direct Route**: Rider delivers directly from pickup to drop location
- **No Hub Routing**: Package does not route through a hub

## Event Schema Requirements

All event payloads include:
- `event_id` (UUID v4) for deduplication
- `traceparent` (W3C trace format: `00-<trace-id>-<span-id>-<flags>`) for distributed tracing
- `timestamp` (ISO 8601 UTC) for event ordering

## Correlation IDs

Events use business correlation IDs:
- `search_id` - Correlates `/search` and `/init` flows (generated by UOIS Gateway)
- `quote_id` - Correlates `/init` and `/confirm` flows (generated by Order Service)
- `dispatch_order_id` - Internal order identifier (post-confirmation, generated by Order Service)

**Note**: These are business IDs for event correlation, NOT WebSocket `correlation_id` (which UOIS Gateway never uses).

## Consumer Group Strategy

UOIS Gateway uses consumer group: **`uois-gateway-consumers`** (shared across all instances)

## Stream Names

- **Location Streams**: `stream.location.search`
- **Quote Streams**: `quote:computed`
- **UOIS Streams**: `stream.uois.*` (init_requested, confirm_requested, quote_created, order_confirmed, etc.)
- **Admin Client Events**: `stream:admin.client.events`

## Test Output

The test script provides detailed logging:
- ✓ Success indicators for each test step
- Event publication/consumption logs
- HTTP request/response logs
- Error details if tests fail

## Troubleshooting

### UOIS Gateway Not Responding
- Ensure UOIS Gateway is running: `make run` or `go run cmd/server/main.go`
- Check `UOIS_BASE_URL` environment variable
- Verify UOIS Gateway is listening on the expected port (default: 8080)

### Redis Connection Failed
- Ensure Redis is running: `redis-server` or Docker container
- Check `REDIS_ADDR` and `REDIS_PASSWORD` environment variables
- Verify Redis is accessible from the test environment

### Events Not Consumed
- Verify consumer groups are initialized: `uois-gateway-consumers`
- Check Redis Stream exists and has messages
- Ensure UOIS Gateway event consumers are running

### Callback Not Received
- UOIS Gateway sends callbacks asynchronously (within TTL period)
- Check callback URL construction: `{bap_uri}/on_{action}`
- Verify callback service is configured correctly

## Extending the Tests

To add new test scenarios:

1. **Add Mock Payload**: Create new JSON file in `testdata/mocks/`
2. **Add Test Method**: Create new test method in `test-e2e-integration.go`
3. **Update RunAllTests**: Add new test to the test suite

Example:
```go
func (it *IntegrationTest) TestNewFlow(ctx context.Context) error {
    // Load mock payload
    payload, err := it.LoadJSONFile("ondc/requests/new_endpoint.json")
    // Send request
    // Validate response
    // Check events
    return nil
}
```

## Integration with CI/CD

The integration test can be run in CI/CD pipelines:

```yaml
# Example GitHub Actions workflow
- name: Run E2E Integration Tests
  run: |
    docker-compose up -d redis postgres
    go run scripts/test-e2e-integration.go
  env:
    UOIS_BASE_URL: http://localhost:8080
    REDIS_ADDR: localhost:6379
```

## Related Documentation

- [UOIS Gateway Functional Requirements](../docs/production-docs/UOISGateway_FR.md)
- [Integration Boundary](../docs/production-docs/INTEGRATION_BOUNDARY.md)
- [ONDC API Contract](../docs/production-docs/ONDC%20-%20API%20Contract%20for%20Logistics%20(v1.2.0).md)

