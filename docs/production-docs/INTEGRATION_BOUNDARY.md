# UOIS Gateway Integration Boundary Reference

## Overview
UOIS Gateway is a middleware service that acts as a protocol translation and routing layer between external ONDC network participants and internal dispatch services. **NOT a source of truth for business logic, pricing, or order lifecycle management.**

**Communication Patterns:**
- **Event-Driven (Async)**: Redis Streams for service-to-service events
- **Synchronous (Sync)**: 
  - HTTP (REST) for ONDC client requests and callbacks
  - gRPC for internal service calls (Order Service, Admin Service)
  - HTTP (REST) for Zendesk integration

## Consumer Role (Events Consumed)

### From Quote Service
- **`QUOTE_COMPUTED`** → Stream: `quote:computed`
  - Triggers: `/on_search` callback composition for `/search` flow
  - Payload: Serviceability, pricing, ETAs, distance (Quote Service passes through Location Service fields)

### From Order Service
- **`QUOTE_CREATED`** → Stream: `stream.uois.quote_created`
  - Triggers: `/on_init` callback composition for `/init` flow
  - Payload: Quote ID, price, ETAs, TTL (PT15M validity period)

- **`QUOTE_INVALIDATED`** → Stream: `stream.uois.quote_invalidated`
  - Triggers: `/on_init` callback with error for `/init` flow
  - Payload: Quote ID, error details, requires_research flag

- **`ORDER_CONFIRMED`** → Stream: `stream.uois.order_confirmed`
  - Triggers: `/on_confirm` callback composition for `/confirm` flow
  - Payload: Dispatch order ID, quote ID, rider assignment details

- **`ORDER_CONFIRM_FAILED`** → Stream: `stream.uois.order_confirm_failed`
  - Triggers: `/on_confirm` callback with error for `/confirm` flow
  - Payload: Quote ID, failure reason, retry count

### From Admin Service
- **`client.*` events** → Stream: `stream:admin.client.events`
  - `client.created` → Upsert client into local registry
  - `client.updated` → Update client in local registry
  - `client.suspended` → Update status to SUSPENDED
  - `client.revoked` → Update status to REVOKED
  - `client.api_key_rotated` → Update client_secret_hash in local registry
  - Purpose: Sync client credentials and configuration for runtime authentication

### Event Consumer Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                    UOIS Gateway (Consumer)                       │
└─────────────────────────────────────────────────────────────────┘
                              │
                              │ Subscribe to Redis Streams
                              │
        ┌─────────────────────┼─────────────────────┐
        │                     │                     │
        ▼                     ▼                     ▼
┌───────────────┐   ┌───────────────┐   ┌───────────────┐
│ Quote Service │   │ Order Service │   │ Admin Service │
└───────────────┘   └───────────────┘   └───────────────┘
        │                     │                     │
        │ Publish             │ Publish             │ Publish
        │                     │                     │
        ▼                     ▼                     ▼
┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐
│ quote:computed   │  │ stream.uois.*     │  │ stream:admin.    │
│                  │  │                  │  │ client.events    │
│ QUOTE_COMPUTED   │  │ QUOTE_CREATED    │  │                  │
│                  │  │ QUOTE_INVALIDATED│  │ client.created   │
│                  │  │ ORDER_CONFIRMED  │  │ client.updated   │
│                  │  │ ORDER_CONFIRM_   │  │ client.suspended │
│                  │  │   FAILED         │  │ client.revoked   │
│                  │  │                  │  │ client.api_key_  │
│                  │  │                  │  │   rotated        │
└──────────────────┘  └──────────────────┘  └──────────────────┘
        │                     │                     │
        └─────────────────────┼─────────────────────┘
                              │
                              ▼
                    ┌──────────────────┐
                    │  Redis Streams   │
                    │  (Event Bus)     │
                    └──────────────────┘
                              │
                              │ Consume Events
                              │
                              ▼
                    ┌──────────────────┐
                    │  UOIS Gateway     │
                    │  Event Handlers   │
                    │                  │
                    │  • /on_search    │
                    │  • /on_init       │
                    │  • /on_confirm    │
                    │  • Client Registry│
                    │    Sync          │
                    └──────────────────┘
```

## Producer Role (Events Published)

### To Location Service
- **`SEARCH_REQUESTED`** → Stream: `stream.location.search`
  - Triggers: Serviceability computation for `/search` flow
  - Payload: search_id, pickup/drop coordinates, traceparent

### To Order Service
- **`INIT_REQUESTED`** → Stream: `stream.uois.init_requested`
  - Triggers: Quote validation and creation for `/init` flow
  - Payload: search_id, pickup/drop addresses, package info, traceparent

- **`CONFIRM_REQUESTED`** → Stream: `stream.uois.confirm_requested`
  - Triggers: Order creation and rider assignment for `/confirm` flow
  - Payload: quote_id, client_id, client_order_id, payment_info, traceparent

### Event Producer Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                    UOIS Gateway (Producer)                       │
└─────────────────────────────────────────────────────────────────┘
                              │
                              │ Publish to Redis Streams
                              │
        ┌─────────────────────┼─────────────────────┐
        │                     │                     │
        ▼                     ▼                     ▼
┌──────────────────┐  ┌──────────────────┐
│ stream.location. │  │ stream.uois.*    │
│ search           │  │                  │
│                  │  │ init_requested   │
│ SEARCH_REQUESTED │  │ confirm_requested│
│                  │  │                  │
└──────────────────┘  └──────────────────┘
        │                     │
        │                     │
        ▼                     ▼
┌───────────────┐   ┌───────────────┐
│ Location      │   │ Order Service │
│ Service       │   │               │
│               │   │               │
│ Consumes:     │   │ Consumes:     │
│ • SEARCH_     │   │ • INIT_       │
│   REQUESTED   │   │   REQUESTED   │
│               │   │ • CONFIRM_    │
│ Processes:    │   │   REQUESTED   │
│ • Serviceability│ │               │
│   computation │   │ Processes:    │
│ • Distance/ETA│   │ • Quote       │
│               │   │   validation  │
│               │   │ • Order       │
│               │   │   creation    │
│               │   │ • Rider       │
│               │   │   assignment  │
└───────────────┘   └───────────────┘
        │                     │
        │ Publish             │ Publish
        │                     │
        ▼                     ▼
┌──────────────────┐  ┌──────────────────┐
│ location:        │  │ stream.uois.*    │
│ serviceability:  │  │                  │
│ found            │  │ quote_created    │
│                  │  │ quote_invalidated│
│ SERVICEABILITY_  │  │ order_confirmed  │
│ FOUND            │  │ order_confirm_failed   │
└──────────────────┘  └──────────────────┘
        │                     │
        │ Consumed by         │ Consumed by
        │                     │
        ▼                     ▼
┌───────────────┐   ┌──────────────────┐
│ Quote Service │   │  UOIS Gateway    │
│               │   │  (Consumer)      │
│ Consumes:     │   │                  │
│ • SERVICEABILITY│ │ Consumes:        │
│   FOUND       │   │ • QUOTE_CREATED  │
│               │   │ • QUOTE_INVALIDATED│
│ Processes:    │   │ • ORDER_CONFIRMED│
│ • Price       │   │ • ORDER_CONFIRM_ │
│   computation │   │   FAILED         │
│ • Pass-through│   │                  │
│   serviceability│ │                  │
│   fields      │   │                  │
└───────────────┘   └──────────────────┘
        │
        │ Publish
        │
        ▼
┌──────────────────┐
│ quote:computed   │
│                  │
│ QUOTE_COMPUTED   │
│                  │
│ (consumed by     │
│  UOIS Gateway)  │
└──────────────────┘
        │
        │ Consumed by
        │
        ▼
┌──────────────────┐
│  UOIS Gateway    │
│  (Consumer)      │
│                  │
│ Event Handlers:  │
│ • /on_search     │
│ • /on_init       │
│ • /on_confirm    │
└──────────────────┘
```
## External Service Integrations

### HTTP Server (UOIS Gateway Exposes)
UOIS Gateway exposes **HTTP REST API** for ONDC-compliant endpoints:

- **Protocol**: HTTP/1.1, HTTPS
- **Port**: `8080` (default, configurable via `HTTP_PORT`)
- **Consumers**: ONDC network participants (Buyer NPs)

**ONDC Endpoints (7 main APIs + 2 IGM APIs):**

**Pre-order APIs:**
1. `POST /search` - Serviceability and quote request
2. `POST /init` - Quote initialization request
3. `POST /confirm` - Order confirmation request

**Post-order APIs:**
4. `POST /status` - Order status query
5. `POST /track` - Order tracking query
6. `POST /cancel` - Order cancellation request
7. `POST /update` - Order update request (weight, dimensions, RTS)
8. `POST /rto` - Return to Origin request

**Issue & Grievance Management (IGM) APIs:**
9. `POST /issue` - Issue creation from Buyer NP
10. `POST /issue_status` - Issue status query

**Callback Endpoints (Receive callbacks from Buyer NPs):**
- `POST /on_issue` - Receive issue callbacks
- `POST /on_issue_status` - Receive issue status callbacks

**Webhook Endpoints:**
- `POST /webhooks/zendesk/ticket_update` - Receive Zendesk webhooks

**Authentication:**
- `Authorization` header required (Basic or Bearer token)
- Client credentials validated against local client registry (synced from Admin Service via events)
- IP allowlisting (CIDR matching) per client configuration

**ONDC Compliance:**
- Request/response signing (ONDC signature validation)
- Protocol versioning support
- Network registry integration for public key fetching

### gRPC Clients (UOIS Gateway Consumes)

#### Order Service
- **Service**: `dispatch.order.v1.OrderService`
- **Port**: `50052` (default, configurable)
- **Protocol**: gRPC (HTTP/2)
- **Purpose**: Post-confirmation order operations

**gRPC Methods (5 RPCs):**
1. `GetOrder` - Fetch order status for `/status` flow
2. `GetOrderTracking` - Fetch tracking data for `/track` flow
3. `CancelOrder` - Cancel order for `/cancel` flow
4. `UpdateOrder` - Update order for `/update` flow (weight, dimensions, RTS)
5. `InitiateRTO` - Initiate Return to Origin for `/rto` flow

**Authentication:**
- Internal service-to-service authentication (API keys or mTLS)
- `traceparent` propagated in gRPC metadata for distributed tracing

#### Admin Service
- **Service**: `dispatch.admin.v1.AdminService`
- **Port**: Configurable
- **Protocol**: gRPC (HTTP/2)
- **Purpose**: Client configuration fetching

**gRPC Methods:**
1. `GetClientConfig` - Fetch client-specific configuration (protocol handling, callback URLs, feature flags, SLA requirements)
2. `AuthenticateClient` - Optional fallback for cache warm-up (not in hot-path)

**Authentication:**
- Internal service-to-service authentication
- Used for cache warm-up only (hot-path uses local client registry)

### HTTP Clients (UOIS Gateway Consumes)

#### Zendesk
- **Protocol**: HTTP/1.1, HTTPS
- **Purpose**: Issue & Grievance Management (IGM) integration
- **API Version**: Zendesk REST API v2

**HTTP Methods:**
1. `POST /api/v2/tickets.json` - Create ticket from ONDC issue
2. `GET /api/v2/tickets/{ticket_id}.json` - Get ticket details
3. `PUT /api/v2/tickets/{ticket_id}.json` - Update ticket status
4. `POST /api/v2/tickets/{ticket_id}/comments.json` - Add comments to ticket

**Authentication:**
- Basic authentication (email/token) or OAuth token
- API token or OAuth access token in Authorization header
- Secure credential storage

**Webhook Receiver:**
- `POST /webhooks/zendesk/ticket_update` - Receive ticket update webhooks from Zendesk
- Validates webhook signature (Zendesk webhook signature validation)

### Other Integrations
- **Database**: Postgres-E (audit database, separate instance) - Request/response logs, order mappings, client registry
- **Redis Streams**: Event-driven communication, idempotency, temporary caching
- **Redis Cache**: Client configuration cache, order mapping cache, request context

### Key Integration Principles

1. **Hybrid Communication**: 
   - **Event-Driven (Async)**: Redis Streams for service-to-service events (Location, Quote, Order Service)
   - **Synchronous (Sync)**: 
     - HTTP for ONDC client requests and callbacks
     - gRPC for internal service calls (Order Service, Admin Service)
     - HTTP for Zendesk integration
2. **Protocol Translation**: ONDC/Beckn ↔ Internal contracts (normalized format)
3. **Idempotency**: All events include `event_id` (UUID v4) for deduplication; ONDC requests use `transaction_id` + `message_id` hash
4. **Loose Coupling**: Depends on contracts, not implementations
5. **Single Source of Truth**: 
   - Order Service owns order state
   - Admin Service owns client configuration (UOIS maintains local projection)
   - Zendesk owns issue resolution (UOIS maintains sync and mappings)
6. **Request/Response Pattern**: 
   - Immediate HTTP 200 OK ACK for ONDC requests (< 1 second)
   - Asynchronous callback delivery within TTL period
   - gRPC provides immediate feedback for internal service calls
7. **Client Registry**: Local projection of Admin Service client data (synced via events, not direct DB queries)

### Event Schema Requirements

**All UOIS Gateway events MUST include:**
- `event_id`: UUID v4 for event-level deduplication
- `event_version`: Integer (starting at 1) for contract evolution
- `traceparent`: W3C trace context for distributed tracing
- `timestamp`: ISO 8601 UTC timestamp

**ONDC Request/Response Requirements:**
- `transaction_id`: ONDC transaction identifier for callback correlation
- `message_id`: ONDC message identifier for callback correlation
- `bap_uri`: Buyer NP URI for callback URL construction
- ONDC signature validation and signing

### Consumer Group Strategy

**Per-stream consumer groups** (no shared backpressure):
- `uois-quote-computed-cg`: QUOTE_COMPUTED events (for `/search` callbacks)
- `uois-quote-created-cg`: QUOTE_CREATED events (for `/init` callbacks)
- `uois-quote-invalidated-cg`: QUOTE_INVALIDATED events (for `/init` error callbacks)
- `uois-order-confirmed-cg`: ORDER_CONFIRMED events (for `/confirm` callbacks)
- `uois-order-confirm-failed-cg`: ORDER_CONFIRM_FAILED events (for `/confirm` error callbacks)
- `uois-client-events-cg`: client.* events from Admin Service (for client registry sync)

### Idempotency Contract

**Redis deduplication keys**: `uois-gateway:dedup:{event_id}` or `uois-gateway:dedup:{transaction_id}:{message_id}`
- TTL: 24-48 hours
- Write failures: Event retried (at-least-once delivery)

**ONDC Request Deduplication:**
- Track request hashes: `SHA-256(transaction_id + message_id + timestamp + payload)`
- Store in Redis: `uois-gateway:ondc:request:{hash}` (TTL: 24 hours)
- Store in Postgres-E: `audit.request_response_logs` (7-year retention)

**Order Mapping Deduplication:**
- `search_id` → `quote_id` mapping (idempotent)
- `quote_id` → `dispatch_order_id` mapping (idempotent)
- `client_order_id` → `dispatch_order_id` mapping (idempotent)

### Stream Naming Convention

- **Location Streams**: `stream.location.*` | `location:*`
- **Quote Streams**: `quote:*`
- **UOIS Streams**: `stream.uois.*`
- **Order Lifecycle**: `stream.order.*` (consumed by other services)
- **Admin Client Events**: `stream:admin.client.events`
- **DroneAI Streams**: `stream.droneai.*` (consumed by Order Service)

## Synchronous Communication

### From ONDC Clients (HTTP)
- **ONDC API Requests** → HTTP: `POST /{action}` (search, init, confirm, status, track, cancel, update, rto, issue, issue_status)
  - Port: `8080` (default)
  - Authentication: `Authorization` header (Basic or Bearer token)
  - Protocol: ONDC/Beckn compliant
  - Response: Immediate HTTP 200 OK ACK/NACK (< 1 second)
  - Callback: Asynchronous `/on_{action}` callback within TTL period

### To ONDC Clients (HTTP)
- **ONDC Callbacks** → HTTP: `POST {bap_uri}/on_{action}`
  - Callback URLs: Constructed from `context.bap_uri` in request
  - Retry Policy: Exponential backoff (1s → 2s → 4s → 8s → 16s), max 5 attempts
  - TTL Constraint: All retries must complete within request TTL period (typically PT30S)
  - Dead Letter Queue: For persistent failures after max retries

### To Order Service (gRPC)
- **Post-confirmation Operations** → gRPC: `dispatch.order.v1.OrderService`
  - Methods: `GetOrder`, `GetOrderTracking`, `CancelOrder`, `UpdateOrder`, `InitiateRTO`
  - Port: `50052` (default)
  - Authentication: Internal service-to-service (API keys or mTLS)
  - Response: Immediate gRPC response with order data

### To Admin Service (gRPC)
- **Client Configuration** → gRPC: `dispatch.admin.v1.AdminService`
  - Method: `GetClientConfig`
  - Purpose: Fetch client-specific configuration (cache warm-up, not hot-path)
  - Authentication: Internal service-to-service

### To Zendesk (HTTP)
- **IGM Operations** → HTTP: Zendesk REST API v2
  - Methods: Create ticket, get ticket, update ticket, add comments
  - Authentication: Basic auth (email/token) or OAuth token
  - Purpose: Bidirectional sync between ONDC issues and Zendesk tickets

### Communication Flow

**Pre-order Flow (`/search` → `/init` → `/confirm`):**
```
ONDC Client → UOIS Gateway HTTP → Immediate ACK
                ↓
         Publish SEARCH_REQUESTED → Location Service
                ↓
         Subscribe QUOTE_COMPUTED ← Quote Service
                ↓
         Compose /on_search callback → ONDC Client
```

**Post-order Flow (`/status`, `/track`, `/cancel`, `/update`, `/rto`):**
```
ONDC Client → UOIS Gateway HTTP → Immediate ACK
                ↓
         Resolve client_order_id → dispatch_order_id
                ↓
         Call Order Service gRPC
                ↓
         Compose /on_{action} callback → ONDC Client
```

**IGM Flow (`/issue` → `/issue_status`):**
```
ONDC Client → UOIS Gateway HTTP → Immediate ACK
                ↓
         Create Zendesk Ticket → Zendesk HTTP
                ↓
         Store issue in Redis
                ↓
         Compose /on_issue callback → ONDC Client
                ↓
         Zendesk Webhook → UOIS Gateway HTTP
                ↓
         Update issue status → Redis
                ↓
         Compose /on_issue_status callback → ONDC Client
```

**Characteristics:**
- **Synchronous**: Immediate HTTP 200 OK ACK for ONDC requests
- **Asynchronous**: Event-driven processing and callback delivery
- **Protocol Translation**: ONDC/Beckn ↔ Internal normalized format
- **Distributed Tracing**: W3C traceparent propagation in all events and calls
- **Idempotency**: All operations idempotent (event_id, transaction_id + message_id)

---

**Version:** v1.0.0 | **Source:** UOISGateway_FR.md | **Last Updated:** January 2025

