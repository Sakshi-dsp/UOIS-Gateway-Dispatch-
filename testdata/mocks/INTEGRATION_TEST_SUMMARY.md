# Integration Test Framework Summary

## Overview

Complete end-to-end integration testing framework for UOIS Gateway, with all testing assets living inside the UOIS repository. **Dispatch is a logistics seller NP (BPP) for P2P (Point-to-Point) delivery only.**

## Directory Structure

```
testdata/mocks/
├── ondc/
│   ├── requests/              # ONDC request payloads (P2P delivery)
│   │   ├── search.json        # /search request
│   │   ├── init.json          # /init request
│   │   └── confirm.json       # /confirm request
│   └── callbacks/             # ONDC callback payloads (P2P delivery)
│       ├── on_search.json    # /on_search callback
│       ├── on_init.json       # /on_init callback
│       └── on_confirm.json    # /on_confirm callback
└── events/
    ├── published/             # Events published by UOIS Gateway
    │   ├── search_requested.json
    │   ├── init_requested.json
    │   └── confirm_requested.json
    └── consumed/             # Events consumed by UOIS Gateway
        ├── quote_computed.json
        ├── quote_created.json
        ├── quote_invalidated.json
        ├── order_confirmed.json
        ├── order_confirm_failed.json
        └── client_created.json
```

## Test Script

**Location**: `scripts/test-e2e-integration.go`

**Purpose**: Orchestrates complete ONDC → UOIS → internal services flow

**Features**:
- Loads ONDC payloads from `testdata/mocks/`
- Sends HTTP requests to UOIS Gateway endpoints
- Publishes events to Redis Streams (simulating downstream services)
- Consumes events from Redis Streams (validating UOIS Gateway consumption)
- Validates request → event → callback chaining

## Event Coverage

### Events Published by UOIS Gateway (3 events)

| Event Type | Stream | Consumer | Purpose |
|------------|--------|----------|---------|
| `SEARCH_REQUESTED` | `stream.location.search` | Location Service | Trigger serviceability computation |
| `INIT_REQUESTED` | `stream.uois.init_requested` | Order Service | Trigger quote validation and creation |
| `CONFIRM_REQUESTED` | `stream.uois.confirm_requested` | Order Service | Trigger order creation and rider assignment |

### Events Consumed by UOIS Gateway (6 event types)

| Event Type | Stream | Producer | Purpose |
|------------|--------|----------|---------|
| `QUOTE_COMPUTED` | `quote:computed` | Quote Service | Receive quote for `/search` response |
| `QUOTE_CREATED` | `stream.uois.quote_created` | Order Service | Receive validated quote for `/init` response |
| `QUOTE_INVALIDATED` | `stream.uois.quote_invalidated` | Order Service | Receive quote validation failure |
| `ORDER_CONFIRMED` | `stream.uois.order_confirmed` | Order Service | Receive order confirmation for `/confirm` response |
| `ORDER_CONFIRM_FAILED` | `stream.uois.order_confirm_failed` | Order Service | Receive order confirmation failure |
| `client.*` events | `stream:admin.client.events` | Admin Service | Sync client registry |

## P2P Delivery Characteristics

All mock payloads reflect **P2P (Point-to-Point) delivery**:

✅ **Item Descriptor Code**: `"P2P"` (not `"P2H2P"`)  
✅ **Fulfillment Type**: `"Delivery"` (direct, not hub-based)  
✅ **AWB Number**: **NOT included** (AWB only required for P2H2P)  
✅ **Direct Route**: Rider delivers directly from pickup to drop  
✅ **No Hub Routing**: Package does not route through a hub  

## Test Flow

### 1. `/search` Flow
```
ONDC Client → /search → UOIS Gateway
                ↓
         SEARCH_REQUESTED → stream.location.search
                ↓
         QUOTE_COMPUTED ← quote:computed (simulated)
                ↓
         /on_search callback → ONDC Client
```

### 2. `/init` Flow
```
ONDC Client → /init → UOIS Gateway
                ↓
         INIT_REQUESTED → stream.uois.init_requested
                ↓
         QUOTE_CREATED ← stream.uois.quote_created (simulated)
                ↓
         /on_init callback → ONDC Client
```

### 3. `/confirm` Flow
```
ONDC Client → /confirm → UOIS Gateway
                ↓
         CONFIRM_REQUESTED → stream.uois.confirm_requested
                ↓
         ORDER_CONFIRMED ← stream.uois.order_confirmed (simulated)
                ↓
         /on_confirm callback → ONDC Client
```

## Running Tests

```bash
# Basic usage
go run scripts/test-e2e-integration.go

# With custom configuration
export UOIS_BASE_URL="http://localhost:8080"
export REDIS_ADDR="localhost:6379"
go run scripts/test-e2e-integration.go
```

## Key Features

1. **Self-Contained**: All test assets in UOIS repository
2. **No External Mocks**: Uses Redis Streams for event simulation
3. **Complete Coverage**: Tests all ONDC flows and event interactions
4. **P2P Compliant**: All payloads reflect P2P delivery characteristics
5. **Event Validation**: Validates event publishing and consumption
6. **Correlation Testing**: Tests ID propagation and echo contracts

## Event Schema Compliance

All event payloads include:
- ✅ `event_id` (UUID v4) for deduplication
- ✅ `traceparent` (W3C format: `00-<trace-id>-<span-id>-<flags>`) for distributed tracing
- ✅ `timestamp` (ISO 8601 UTC) for event ordering

## Consumer Group

UOIS Gateway uses consumer group: **`uois-gateway-consumers`** (shared across all instances)

## Documentation

- [Mock Payloads README](./README.md)
- [P2P Delivery Guide](./README_P2P.md)
- [Integration Test Script README](../scripts/README_E2E_INTEGRATION.md)
- [UOIS Gateway FR](../../docs/production-docs/UOISGateway_FR.md)
- [Integration Boundary](../../docs/production-docs/INTEGRATION_BOUNDARY.md)

