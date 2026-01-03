# Mock Payloads for Integration Testing

This directory contains mock ONDC-compliant JSON payloads and event payloads for end-to-end integration testing of UOIS Gateway.

## Directory Structure

```
testdata/mocks/
├── ondc/
│   ├── requests/          # ONDC request payloads
│   │   ├── search.json
│   │   ├── init.json
│   │   └── confirm.json
│   └── callbacks/         # ONDC callback payloads
│       ├── on_search.json
│       ├── on_init.json
│       └── on_confirm.json
└── events/
    ├── published/         # Events published by UOIS Gateway
    │   ├── search_requested.json
    │   ├── init_requested.json
    │   └── confirm_requested.json
    └── consumed/          # Events consumed by UOIS Gateway
        ├── quote_computed.json
        ├── quote_created.json
        ├── quote_invalidated.json
        ├── order_confirmed.json
        ├── order_confirm_failed.json
        └── client_created.json
```

## Usage

These mock payloads are used by the integration test script (`scripts/test-e2e-integration.go`) to:

1. **Send ONDC HTTP requests** to UOIS Gateway endpoints
2. **Publish events** to Redis Streams (simulating downstream services)
3. **Consume events** from Redis Streams (validating UOIS Gateway consumption)
4. **Validate end-to-end flow** from ONDC request → event → callback

## Payload Coverage

### ONDC Requests
- `/search` - Serviceability and quote request
- `/init` - Quote initialization request  
- `/confirm` - Order confirmation request

### ONDC Callbacks
- `/on_search` - Catalog response with fulfillment options
- `/on_init` - Quote response with TTL and cancellation terms
- `/on_confirm` - Order acceptance with rider assignment

### Events Published by UOIS Gateway
- `SEARCH_REQUESTED` → `stream.location.search` → Location Service
- `INIT_REQUESTED` → `stream.uois.init_requested` → Order Service
- `CONFIRM_REQUESTED` → `stream.uois.confirm_requested` → Order Service

### Events Consumed by UOIS Gateway
- `QUOTE_COMPUTED` ← `quote:computed` ← Quote Service
- `QUOTE_CREATED` ← `stream.uois.quote_created` ← Order Service
- `QUOTE_INVALIDATED` ← `stream.uois.quote_invalidated` ← Order Service
- `ORDER_CONFIRMED` ← `stream.uois.order_confirmed` ← Order Service
- `ORDER_CONFIRM_FAILED` ← `stream.uois.order_confirm_failed` ← Order Service
- `client.*` events ← `stream:admin.client.events` ← Admin Service

## Event Schema Requirements

All event payloads include:
- `event_id` (UUID v4) for deduplication
- `traceparent` (W3C trace format) for distributed tracing
- `timestamp` (ISO 8601 UTC) for event ordering

## Correlation IDs

Events use business correlation IDs:
- `search_id` - Correlates `/search` and `/init` flows
- `quote_id` - Correlates `/init` and `/confirm` flows
- `dispatch_order_id` - Internal order identifier (post-confirmation)

**Note**: These are business IDs for event correlation, NOT WebSocket `correlation_id` (which UOIS Gateway never uses).

