# UOIS Gateway Implementation Plan

**Version:** 1.0.0  
**Source of Truth:** `UOISGateway_FR.md`  
**Last Updated:** January 2025

---

## 1. Overview & Scope

### 1.1 Purpose

The **Universal Order Interface Service (UOIS) Gateway** is a middleware service that acts as a protocol translation and routing layer between:

- **External Clients**: ONDC network participants (Buyer NPs)
- **Internal Services**: Order Service, Location Service, Quote Service, Admin Service, DroneAI, Notification Service

### 1.2 Core Responsibilities

- Protocol translation (ONDC/Beckn ↔ Internal contracts)
- Request validation and transformation
- Client authentication and authorization
- Client configuration processing (fetch from Admin Service, cache in Redis, persist in client registry)
- Event-driven request orchestration (publish/subscribe to event stream)
- Callback relay via event consumption
- Idempotency and deduplication
- Security and non-repudiation (ONDC signing)
- Error normalization and standardization
- **Distributed tracing** (generate W3C traceparent at edge, propagate through all events and calls)
- Audit logging for dispute resolution (7-year retention to Postgres-E)
- Issue & Grievance Management (IGM) for ONDC compliance

### 1.3 Service Boundaries

**UOIS Gateway owns:**
- Protocol translation and validation
- Client authentication and rate limiting
- Request/response signing (ONDC)
- Event publishing and subscription
- Callback relay via event consumption
- Idempotency and deduplication
- Issue & Grievance Management (IGM) API endpoints
- IGM state management and Zendesk Helpdesk integration (bidirectional sync)
  - Issue storage and references (Redis)
- Client configuration processing and caching (Redis cache, Postgres-E client registry)
- Temporary caching for performance
- Audit logging to Postgres-E (audit schema)
- **Trace ID generation** (W3C traceparent) at edge for distributed tracing
- search_id (Serviceability ID) generation for `/search` requests (UOIS Gateway generates search_id, never derives from request payload)

**UOIS Gateway does NOT own:**
- Business logic (pricing, capacity, routing) → Quote Service, Location Service, DroneAI
- Order lifecycle management → Order Service
  - Issue resolution and ticket content → External Helpdesk (Zendesk) - UOIS Gateway maintains sync and references only
- Client configuration source of truth → Admin Service (UOIS Gateway fetches and processes config, but Admin Service owns the authoritative configuration)

---

## 2. Common Request Processing Contract

**Purpose**: Define the standard processing pattern that applies to ALL APIs unless explicitly overridden.

**Authoritative Rule**: Unless explicitly overridden, all sections MUST NOT restate ACK, TTL, retry, callback, audit, or tracing behavior defined here.

### 2.1 Standard Processing Flow

Applies to: `/search`, `/init`, `/confirm`, `/status`, `/track`, `/cancel`, `/update`, `/rto`

**Note**: Event consumption uses Redis Streams consumer groups (`XREADGROUP`), not pub/sub subscriptions. See Section 3.3 for detailed consumer group architecture.

```pseudo
function HandleRequest(request):
    // 1. Edge Processing
    traceparent = GenerateOrExtractTraceparent(request.headers)
    span = StartRootSpan(traceparent)
    
    client_id, err = AuthenticateClient(request)
    if err != nil:
        return NACK(401, error_code=65002)
    
    err = ValidateRequestStructure(request)
    if err != nil:
        return NACK(400, error_code=65001)
    
    // 2. Immediate Response
    return HTTP_200_OK_ACK()  // < 1 second, non-blocking
    
    // 3. Asynchronous Processing (background)
    go ProcessAsync(request, traceparent, client_id)
end

function ProcessAsync(request, traceparent, client_id):
    // Publish events (when applicable)
    if needs_event:
        event = TransformToEvent(request)
        event.traceparent = traceparent
        PublishEvent(stream, event)  // Uses XADD to Redis Streams
    
    // Consume response events from consumer group (when applicable)
    if needs_event_consumption:
        lifecycle_correlation_id = ExtractLifecycleCorrelationID(request)  // search_id or quote_id (NOT WebSocket correlation_id)
        // Uses XREADGROUP with consumer group, filters by lifecycle_correlation_id, ACKs after processing
        response_event = ConsumeFromStream(stream, "uois-gateway-consumers", lifecycle_correlation_id, timeout=request.ttl)
    
    // Call Order Service gRPC (when applicable)
    if needs_grpc:
        grpc_response = CallOrderService(method, dispatch_order_id)
    
    // 4. Callback Delivery
    callback_url = ConstructCallbackURL(request.bap_uri, action)
    callback_payload = ComposeCallback(response_event, grpc_response)
    
    retry_count = 0
    while retry_count < MAX_RETRIES:
        err = SendCallback(callback_url, callback_payload)
        if err == nil:
            break
        retry_count++
        backoff_delay = CalculateExponentialBackoff(retry_count)
        sleep(backoff_delay)
    
    if retry_count >= MAX_RETRIES:
        MoveToDLQ(callback_url, callback_payload)
    
    // 5. Audit & Observability
    PersistAuditLog(request, callback_payload, traceparent, lifecycle_ids)  // search_id, quote_id, dispatch_order_id (NOT correlation_id)
end
```

### 2.2 TTL Handling

**Request TTL**: Buyer NP sends `ttl` in request context (typically `PT30S` for most APIs, `PT15M` for quote validity)

**Response within TTL**: Send callback asynchronously within TTL period

**Quote TTL**: Quote validity period (`PT15M`) - validate before `/confirm`

**Critical Constraint**: Total retry duration (including all backoff delays) must complete within the request TTL period. If exponential backoff exceeds TTL, reduce max retries or backoff intervals.

### 2.3 Idempotency & Correlation

| Operation | Idempotency Key | Lifecycle Correlation ID | Storage Key Pattern |
|-----------|----------------|-------------------------|---------------------|
| `/search` | `search_id` (generated by UOIS) | `search_id` | `search:{search_id}` |
| `/init` | `search_id` (from `/search`) | `search_id` | `search:{search_id}` |
| `/confirm` | `quote_id` (from `/init`) | `quote_id` | `quote:{quote_id}` |
| `/status` | `order.id` (ONDC, from `message.order_id`) | `order.id` (ONDC) | `order:{order.id}` |
| `/track` | `order.id` (ONDC, from `message.order_id`) | `order.id` (ONDC) | `order:{order.id}` |
| `/cancel` | `order.id` (ONDC, from `message.order_id`) | `order.id` (ONDC) | `order:{order.id}` |
| `/update` | `(order.id + update_type)` (ONDC, from `message.order_id`) | `order.id` (ONDC) | `order:{order.id}:update:{update_type}` |
| `/rto` | `order.id` (ONDC, from `message.order_id`) | `order.id` (ONDC) | `order:{order.id}` |

**Idempotency Rules:**
- Track request hashes for replay protection (ONDC `transaction_id` + ONDC `message_id` hash)
- Use lifecycle IDs for event correlation: `search_id` (search/init), `quote_id` (init/confirm)
- Use ONDC `order.id` (seller-generated) for post-confirmation correlation
- **Note**: Lifecycle IDs (`search_id`, `quote_id`, `dispatch_order_id`) are for internal event correlation, NOT WebSocket `correlation_id` (which UOIS Gateway never uses)
- Support idempotent request replay - return existing response if duplicate detected
- Store idempotency keys in Redis with TTL (24 hours) and Postgres-E for audit (7 years)

---

## 3. Interfaces & Contracts

### 3.1 External APIs (HTTP REST)

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
- `POST /webhooks/zendesk/ticket_update` - Receive Zendesk Helpdesk webhooks

**ONDC Callback Requirements** (all 7 callbacks required for Logistics Seller NP):
- `/on_search`: Catalog with fulfillment options, pricing, and terms
- `/on_init`: Quote, cancellation terms, and transaction-level contract terms
- `/on_confirm`: Order acceptance/rejection, fulfillment slots, agent details, AWB number
- `/on_status`: Current order status, fulfillment states, proof of pickup/delivery
- `/on_cancel`: Cancellation details and updated quote
- `/on_update`: Updated order details and updated quote (if weight/dimensions changed)
- `/on_track`: Tracking information (GPS coordinates or tracking URL) - Note: As of July 2023, `callback_url` removed; use polling only

### 3.2 Events Published by UOIS Gateway

| Event Type | Stream | Consumer(s) | Purpose |
|-----------|-------|------------|---------|
| SEARCH_REQUESTED | `stream.location.search` | **Location Service** | Trigger serviceability computation for `/search` flow |
| INIT_REQUESTED | `stream.uois.init_requested` | **Order Service** | Trigger quote validation and creation for `/init` flow |
| CONFIRM_REQUESTED | `stream.uois.confirm_requested` | **Order Service** | Trigger order creation and rider assignment for `/confirm` flow |

**Event Schema Requirements:**
- All events MUST include `search_id` (for `/search` and `/init`) or `quote_id` (for `/confirm`) as correlation key
- All events MUST include `traceparent` (W3C format) for distributed tracing
- All events MUST include `event_id` (UUID v4) for event-level deduplication

**Event Publishing Pattern:**
- Use Redis Streams `XADD` command to publish events
- Stream names are shared resources (no prefixing)
- Events are published asynchronously after HTTP ACK response

### 3.3 Events Consumed by UOIS Gateway

| Event Type | Stream | Producer | Purpose |
|-----------|-------|----------|---------|
| QUOTE_COMPUTED | `quote:computed` | Quote Service | Receive quote for `/search` response to client |
| QUOTE_CREATED | `stream.uois.quote_created` | Order Service | Receive validated quote for `/init` response to client |
| QUOTE_INVALIDATED | `stream.uois.quote_invalidated` | Order Service | Receive quote validation failure for `/init` response to client |
| ORDER_CONFIRMED | `stream.uois.order_confirmed` | Order Service | Receive order confirmation for `/confirm` response to client |
| ORDER_CONFIRM_FAILED | `stream.uois.order_confirm_failed` | Order Service | Receive order confirmation failure for `/confirm` response to client |
| client.* events | `stream:admin.client.events` | Admin Service | Sync client registry (client.created, client.updated, client.suspended, client.revoked, client.api_key_rotated) |

**Event Consumption Pattern (Redis Streams Consumer Groups):**
- **Consumer Group Name**: `uois-gateway-consumers` (shared across all UOIS Gateway instances)
- **Consumer Name**: `uois-gateway-{instance-id}` (unique per instance, e.g., `uois-gateway-instance-1`)
- **Consumption Method**: Use `XREADGROUP` command with blocking reads (`BLOCK` parameter)
- **Event Filtering**: Filter consumed events by lifecycle correlation ID (`search_id`, `quote_id`) after consumption (Redis Streams does not support filtering by field)
- **Note**: These are lifecycle IDs for event correlation, NOT WebSocket `correlation_id` (which UOIS Gateway never generates or uses)
- **ACK Mechanism**: ACK messages after successful processing using `XACK` command
- **Timeout Handling**: Implement timeout logic using `BLOCK` parameter and request start time comparison
- **PEL Handling**: Handle Pending Entry List (PEL) on restart - reprocess unacked messages from previous instance failures

**Consumer Group Setup (One-time initialization):**
```redis
XGROUP CREATE quote:computed uois-gateway-consumers 0 MKSTREAM
XGROUP CREATE stream.uois.quote_created uois-gateway-consumers 0 MKSTREAM
XGROUP CREATE stream.uois.quote_invalidated uois-gateway-consumers 0 MKSTREAM
XGROUP CREATE stream.uois.order_confirmed uois-gateway-consumers 0 MKSTREAM
XGROUP CREATE stream.uois.order_confirm_failed uois-gateway-consumers 0 MKSTREAM
XGROUP CREATE stream:admin.client.events uois-gateway-consumers 0 MKSTREAM
```

**Event Consumption Implementation Pattern:**
```go
// Pseudo-code for event consumption
// lifecycleCorrelationID is search_id or quote_id (NOT WebSocket correlation_id, NOT dispatch_order_id for event filtering)
func ConsumeFromStream(stream, groupName, consumerName, lifecycleCorrelationID string, timeout time.Duration) (Event, error) {
    startTime := time.Now()
    
    for {
        // Check timeout
        if time.Since(startTime) > timeout {
            return nil, ErrTimeout
        }
        
        // Blocking read from consumer group (5 second block)
        messages, err := XREADGROUP(
            GROUP groupName consumerName,
            BLOCK 5000, // 5 second block
            STREAMS stream ">", // ">" means new messages only
            COUNT 10, // Read up to 10 messages per call
        )
        
        if err != nil {
            // Handle error (timeout, connection, etc.)
            continue
        }
        
        // Filter messages by lifecycle correlation ID (search_id or quote_id depending on stream)
        for _, msg := range messages {
            event := parseEvent(msg.Values)
            // Match by lifecycle ID (search_id or quote_id depending on stream)
            if event.SearchID == lifecycleCorrelationID || 
               event.QuoteID == lifecycleCorrelationID {
                // Found matching event - ACK and return
                XACK stream groupName msg.ID  // msg.ID is Redis Stream message_id (for ACK only)
                return event, nil
            } else {
                // Not our event - ACK and continue (or store for other requests)
                XACK stream groupName msg.ID  // msg.ID is Redis Stream message_id (for ACK only)
            }
        }
    }
}
```

### 3.4 gRPC Clients

**Order Service:**
- Service: `dispatch.order.v1.OrderService`
- Port: `50052` (default, configurable)
- Methods:
  - `GetOrder` - Fetch order status for `/status` flow
  - `GetOrderTracking` - Fetch tracking data for `/track` flow
  - `CancelOrder` - Cancel order for `/cancel` flow
  - `UpdateOrder` - Update order for `/update` flow
  - `InitiateRTO` - Initiate Return to Origin for `/rto` flow

**Admin Service:**
- Service: `dispatch.admin.v1.AdminService`
- Methods:
  - `GetClientConfig` - Fetch client-specific configuration (cache warm-up, not hot-path)
  - `AuthenticateClient` - Optional fallback for cache warm-up (not in hot-path)

### 3.5 HTTP Clients

**Zendesk Helpdesk:**
- Protocol: HTTP/1.1, HTTPS
- API Version: Zendesk REST API v2
- Methods:
  - `POST /api/v2/tickets.json` - Create ticket from ONDC issue
  - `GET /api/v2/tickets/{ticket_id}.json` - Get ticket details
  - `PUT /api/v2/tickets/{ticket_id}.json` - Update ticket status
  - `POST /api/v2/tickets/{ticket_id}/comments.json` - Add comments to ticket

---

## 4. Data & Models

### 4.1 ID Propagation & Echo Contract

**Core Principle**: UOIS Gateway manages internal processing identifiers by generating or receiving them from owning services and propagating them to Buyer NPs inside ONDC-compliant first-class ID fields. Buyer NPs are required to echo these identifiers unchanged in subsequent requests. UOIS Gateway extracts echoed identifiers only from predefined protocol locations and uses them strictly for internal correlation. UOIS Gateway never maps, derives, or infers identifiers across domains.

**Critical ONDC Protocol Constraint**: ONDC guarantees echo ONLY for specific first-class ID fields, NOT for tags or custom fields. Therefore:
- Internal IDs MUST NOT be sent in tags for logging/debugging
- If an ID must be echoed back by Buyer NP, it MUST be placed in an ONDC-defined echoable field
- Tags MUST NOT be used to carry internal identifiers for correlation
- Buyer NP is NOT expected to echo tags and is allowed to drop them

**ID Domain Isolation Law**: UOIS Gateway never maps, translates, derives, infers, or substitutes identifiers across domains. Each identifier is owned by exactly one domain and is treated as an opaque value outside that domain. Multiple identifiers may coexist on the same order record, but none replace or represent another.

**ID Generation & Ownership:**
- `search_id`: Generated by UOIS Gateway during `/search` request processing (internal-only, never sent in ONDC payloads)
- `quote_id`: Generated by Order Service during `/init` flow, received via `QUOTE_CREATED` event (canonical transactional identifier)
- `dispatch_order_id`: Generated by Order Service, received via `ORDER_CONFIRMED` event (internal-only, never echoed by Buyer NP)
- `order.id` (ONDC): **Seller-generated identifier** (generated by Seller NP in `/on_confirm` response)
  - **Who creates it**: Seller NP (UOIS Gateway / Order Service)
  - **Where it originates**: Generated during `/confirm` flow, sent in `/on_confirm` → `message.order.id`
  - **Stability**: Stable across lifecycle
  - **Semantic meaning**: ONDC network-facing order identifier (not internal dispatch_order_id)
  - **Ownership**: Seller-owned, network-facing
  - **Flow requirement**: Buyer MUST echo this `order.id` in all post-order APIs
  - **Echo contract**: Buyer echoes `order.id` in `/status`, `/track`, `/cancel`, `/update`, `/rto`
  - **Internal use-cases**:
    - Callback composition: `/on_status`, `/on_track`, `/on_cancel` MUST return `order.id` (ONDC)
    - Audit & dispute resolution: Support teams reference ONDC order.id
    - IGM/grievance flows: Issues raised against ONDC order.id
    - Customer support tooling: Search by ONDC order.id
    - Reconciliation: Matching ONDC order.id vs internal orders
    - Rider App / Ops UI: ONDC order.id shown alongside dispatch ID
  - **Internal mapping**: `order.id` (ONDC) resolves internally to `dispatch_order_id` (internal-only)

**ONDC Canonical Echo Fields:**

| ONDC Callback | First-Class Field | Seller Provides | Buyer Echoes In | Purpose |
|--------------|------------------|-----------------|-----------------|---------|
| `/on_search` | `message.catalog.bpp/providers[].id` | Stable `provider.id` (e.g., "P1") | `/init` → `message.order.provider.id` | Provider selection (opaque identifier) |
| `/on_init` | `message.order.quote.id` | `quote_id` (canonical transactional identifier) | `/confirm` → `message.order.quote.id` | Quote correlation, validation, audit |
| `/on_confirm` | `message.order.id` | `order.id` (ONDC, seller-generated) | Post-order APIs (`/status`, `/track`, etc.) | Buyer echoes seller-generated `order.id` |

**Critical Rules**:
- **`quote_id` is the canonical transactional identifier** - sent ONLY in `message.order.quote.id` (first-class ONDC field)
- **Internal IDs (`search_id`, `dispatch_order_id`) NEVER leak into ONDC payloads**
- **No tag-based correlation** - tags MUST NOT be used to carry internal identifiers
- **Buyer NP is NOT expected to echo tags** and is allowed to drop them
- **`dispatch_order_id` is NEVER echoed by Buyer NP** - it is Seller-owned and internal-only. Buyer requests after `/confirm` are always addressed using `order.id` (ONDC, seller-generated)

**Terminology:**
- `order.id` (ONDC): Seller-generated order identifier (network-facing)
  - **Who creates it**: Seller NP (UOIS Gateway / Order Service)
  - **Where it originates**: Generated during `/confirm` flow, sent in `/on_confirm` → `message.order.id`
  - **Stability**: Stable across lifecycle
  - **Semantic meaning**: ONDC network-facing order identifier (not internal dispatch_order_id)
  - **Ownership**: Seller-owned, network-facing
  - **Flow requirement**: Buyer MUST echo this `order.id` in all post-order APIs
  - **Internal use-cases**:
    - Callback composition: `/on_status`, `/on_track`, `/on_cancel` MUST return `order.id` (ONDC)
    - Audit & dispute resolution: Support teams reference ONDC order.id
    - IGM/grievance flows: Issues raised against ONDC order.id
    - Customer support tooling: Search by ONDC order.id
    - Reconciliation: Matching ONDC order.id vs internal orders
    - Rider App / Ops UI: ONDC order.id shown alongside dispatch ID
  - **Echo contract**: Buyer echoes seller-generated `order.id` in all post-order APIs
  - **Internal mapping**: `order.id` (ONDC) → `dispatch_order_id` (internal-only)
- `dispatch_order_id`: Internal order identifier used by Order Service (e.g., "ABC0000001")
  - **Source**: Generated by Order Service when order is created
  - **Ownership**: Order Service domain
  - **Usage**: Used for all internal service calls to Order Service (gRPC methods)
  - **Critical**: MUST NEVER be sent in ONDC payloads - internal-only

**ID Propagation Lifecycle (Clear Flow):**
```
/search
  → internal search_id (internal-only, never sent in ONDC payloads)
  → store order record with search_id

/on_search
  → provider.id (stable identifier, e.g., "P1") - opaque, buyer-echoed
  → NO internal IDs in ONDC payload

/init
  → extract provider.id from message.order.provider.id (echoed from /on_search)
  → use internal search_id (from order record) for event correlation
  → generate quote_id (Order Service)

/on_init
  → quote_id (canonical transactional identifier)
  → send ONLY in message.order.quote.id (first-class ONDC field)
  → include TTL (PT15M)
  → persist quote keyed by quote_id

/confirm
  → extract quote_id from message.order.quote.id (echoed from /on_init)
  → lookup quote by quote_id (primary correlation key)
  → validate: existence, TTL not expired, pricing integrity
  → use quote_id for order creation

/on_confirm
  → generate ONDC order.id (seller-generated, network-facing)
  → send order.id in message.order.id (NOT dispatch_order_id)
  → Buyer MUST echo order.id in all post-order APIs
  → dispatch_order_id (internal-only, NEVER sent in ONDC payloads)
  → Internal mapping: order.id (ONDC) → dispatch_order_id (internal)

/post-order APIs (/status, /track, /cancel, /update, /rto)
  → extract order.id from message.order_id (echoed from /on_confirm)
  → lookup order record by order.id (ONDC)
  → retrieve dispatch_order_id from order record
  → use dispatch_order_id for internal Order Service calls
```

**Detailed Implementation Steps:**

- **`/search`**: 
  - UOIS Gateway generates `search_id` (internal-only)
  - Stores order record with `search_id` (no `quote_id` or `dispatch_order_id` yet)
  - Sends stable `provider.id` (e.g., "P1") in `/on_search` → `message.catalog.bpp/providers[].id`
  - **NO internal IDs sent in ONDC payloads**

- **`/init`**: 
  - Extracts `provider.id` from `message.order.provider.id` (echoed from `/on_search`)
  - Validates `provider.id` matches configured stable provider identifier
  - Uses internal `search_id` (from order record lookup) to publish `INIT_REQUESTED` event
  - Receives `QUOTE_CREATED` event with `quote_id` (generated by Order Service)
  - **Generates `quote_id` and persists quote keyed by `quote_id`**
  - Sends `quote_id` in `/on_init` callback → `message.order.quote.id` (ONLY location)
  - Includes TTL (PT15M) in quote response
  - Updates order record: stores `quote_id` alongside `search_id` on same record

- **`/confirm`**: 
  - Extracts `quote_id` from `message.order.quote.id` (echoed from `/on_init` - canonical transactional identifier)
  - **Lookup quote by `quote_id`** (primary correlation key):
    - Validate `quote_id` exists in quote store
    - Validate `quote_id` TTL not expired (PT15M)
    - Validate pricing integrity
  - If missing or invalid → return NACK immediately (error code `65005`)
  - Publishes `CONFIRM_REQUESTED` event with `quote_id`
  - Receives `ORDER_CONFIRMED` event with `dispatch_order_id` (generated by Order Service)
  - Generates ONDC `order.id` (seller-generated, network-facing)
  - Sends ONDC `order.id` in `/on_confirm` callback → `message.order.id` (NOT dispatch_order_id)
  - Stores order record with: `quote_id`, `order.id` (ONDC), `dispatch_order_id` (internal-only)
  - **Critical**: Quote correlation MUST be done using `message.order.quote.id` - this solves the need for correlation, audit, debugging, and lookup
  - **Critical**: `dispatch_order_id` is internal-only and MUST NEVER be sent in ONDC payloads
- **Post-confirmation** (`/status`, `/track`, `/cancel`, `/update`, `/rto`):
  - Extracts ONDC `order.id` from `message.order_id` (echoed from `/on_confirm` - seller-generated)
  - Looks up order record using `order.id` (ONDC)
  - Retrieves `dispatch_order_id` from order record
  - If order record not found → return NACK immediately (error code `65006`)
  - Uses `dispatch_order_id` ONLY for internal Order Service calls (gRPC methods)
  - **Critical**: `order.id` (ONDC) is the canonical post-confirmation identifier - Buyer echoes seller-generated `order.id`

**Fail-Fast Behavior:**
- If expected echoed ID is missing from predefined protocol location → return NACK immediately
- If ID was not previously issued by UOIS or an owning service → return NACK immediately
- No fallback resolution, no inference, no guessing
- No backward compatibility lookups

**Order Record Storage:**
- Order record stores multiple identifiers on the same record:
  - `order.id` (ONDC) - network-facing, seller-generated
  - `dispatch_order_id` (internal execution identifier)
  - `quote_id` (commercial lock identifier)
  - `search_id` (serviceability identifier)
- **Purpose**: Multiple identifiers coexist on same order record; no identifier replaces or represents another
- **Lookup rule**: Lookup is always performed by primary identifier expected by the API. No identifier is inferred, derived, or substituted.
- Store in Redis (temporary, TTL: 30 days) and Postgres-E (audit, 7-year retention)

### 4.2 Coordinate Field Transformation

**MANDATORY:** UOIS Gateway MUST transform ONDC coordinate format to internal Dispatch Network format when proxying requests to domain services (field name transformation only, not ID mapping).

#### ONDC Format (Incoming Requests)

**ONDC API Format:**
- `fulfillment.start.location.gps` = `"lat,lng"` (comma-separated string, e.g., `"12.453544,77.928379"`)
- `fulfillment.end.location.gps` = `"lat,lng"` (comma-separated string, e.g., `"12.453544,77.928379"`)

**Example from `/search` request:**
```json
{
  "message": {
    "intent": {
      "fulfillment": {
        "start": {
          "location": {
            "gps": "12.453544,77.928379"
          }
        },
        "end": {
          "location": {
            "gps": "12.9716,77.5946"
          }
        }
      }
    }
  }
}
```

#### Internal Format (To Domain Services)

**Dispatch Network Format (Required):**
- `origin_lat` (float64) - Pickup latitude
- `origin_lng` (float64) - Pickup longitude  
- `destination_lat` (float64) - Drop latitude
- `destination_lng` (float64) - Drop longitude

**Transformation Rules:**
- ✅ **USE:** `origin_lat`, `origin_lng`, `destination_lat`, `destination_lng`
- ❌ **FORBIDDEN:** `pickup_lat`, `pickup_lng`, `drop_lat`, `drop_lng`

**Applies To:**
- All events published to Location Service (`SEARCH_REQUESTED`, `REVALIDATION_REQUESTED`)
- All events published to Order Service (`INIT_REQUESTED`, `CONFIRM_REQUESTED`)
- All events published to Quote Service (via Location Service events)
- All internal models, structs, and transformations

#### Translation Implementation

**Parsing Logic:**
```go
// Parse ONDC GPS string "lat,lng" → (lat, lng)
func ParseONDCGPS(gpsString string) (lat float64, lng float64, error) {
    parts := strings.Split(gpsString, ",")
    if len(parts) != 2 {
        return 0, 0, fmt.Errorf("invalid GPS format")
    }
    lat, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
    if err != nil {
        return 0, 0, fmt.Errorf("invalid latitude: %w", err)
    }
    lng, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
    if err != nil {
        return 0, 0, fmt.Errorf("invalid longitude: %w", err)
    }
    return lat, lng, nil
}
```

**Event Schema Compliance:**
- Location Service expects: `origin_lat`, `origin_lng`, `destination_lat`, `destination_lng` (see `REVALIDATION_REQUESTED` contract)
- Order Service expects: `origin_lat`, `origin_lng`, `destination_lat`, `destination_lng` (see `INIT_REQUESTED` contract)
- Quote Service receives coordinates via Location Service events (already in internal format)

**Example Transformation:**
```go
// Extract from ONDC request
ondcStartGPS := request.Message.Intent.Fulfillment.Start.Location.GPS  // "12.453544,77.928379"
ondcEndGPS := request.Message.Intent.Fulfillment.End.Location.GPS      // "12.9716,77.5946"

// Translate to internal format
originLat, originLng, _ := ParseONDCGPS(ondcStartGPS)
destinationLat, destinationLng, _ := ParseONDCGPS(ondcEndGPS)

// Publish event with internal format
event := SearchRequestedEvent{
    OriginLat:      originLat,      // NOT pickup_lat
    OriginLng:      originLng,      // NOT pickup_lng
    DestinationLat: destinationLat, // NOT drop_lat
    DestinationLng: destinationLng, // NOT drop_lng
}
```

### 4.3 Client Registry Schema

**UOIS Gateway's Postgres-E:**
```sql
CREATE TABLE client_registry (
  id UUID PRIMARY KEY,
  client_id UUID NOT NULL UNIQUE,  -- From Admin Service
  client_code VARCHAR(50) NOT NULL,
  client_secret_hash TEXT NOT NULL,  -- Synced from Admin Service events
  allowed_ips CIDR[],
  status VARCHAR(20) NOT NULL,  -- ACTIVE, SUSPENDED, REVOKED
  metadata JSONB,
  created_at TIMESTAMP NOT NULL,
  updated_at TIMESTAMP NOT NULL,
  last_synced_at TIMESTAMP NOT NULL  -- Track event sync timestamp
);
```

**Caching Strategy:**
- Redis cache: `client:{client_id}` → full client record
- TTL: 5 minutes (invalidate on events)
- Cache invalidation: On any `client.*` event, delete cache entry

### 4.4 Audit Log Schema

**PostgreSQL-E (`audit.request_response_logs`):**
- `request_id` (PK, UUID)
- `client_id` (FK to client config)
- `protocol_type` (enum: 'ONDC', 'BECKN')
- `request_hash` (SHA-256 of canonical request)
- `signature` (HMAC-SHA256 for non-repudiation)
- `timestamp` (Unix timestamp)
- `nonce` (UUID, for replay prevention)
- `source_ip` (client IP address)
- `request_payload` (JSONB, sanitized) - **Original request in ONDC/Beckn format**
- `response_payload` (JSONB, sanitized) - **Original response in ONDC/Beckn format**
- `processing_time_ms` (integer)
- `status_code` (HTTP status)
- `error_code` (if applicable)
- `search_id` (UUID, nullable) - Serviceability ID for `/search` and `/init` correlation
- `quote_id` (UUID, nullable) - Quote ID for `/init` and `/confirm` correlation
- `dispatch_order_id` (UUID, nullable) - Order ID for `/confirm` correlation
- `transaction_id` (string, nullable) - ONDC transaction ID for callback correlation
- `message_id` (string, nullable) - ONDC message ID for callback correlation (from ONDC `context.message_id`, NOT Redis Stream message_id)
- `bap_uri` (string, nullable) - Buyer NP URI for callback URL construction
- `trace_id` (string, nullable) - Distributed tracing identifier
- `traceparent` (string, nullable) - W3C traceparent header
- `created_at` (timestamp, immutable)

### 4.5 Issue Storage (Redis)

**Storage Keys:**
- `ondc:issue:{issue_id}` (TTL: 30 days) - Issue details
- `ondc:zendesk_ticket:{zendesk_ticket_id}` → `issue_id` - Zendesk ticket reference
- `ondc:financial:{issue_id}` - Financial resolution data

**Data Stored Per Issue:**
- `ondc_issue_id` (unique ONDC issue identifier)
- `zendesk_ticket_id` (Zendesk ticket identifier)
- `transaction_id` (ONDC transaction ID for correlation)
- `order_id` (ONDC order ID)
- `issue_type` (ISSUE, GRIEVANCE, DISPUTE)
- `status` (OPEN, CLOSED)
- `created_at`, `updated_at` (timestamps)
- `resolution_provider` (respondent info, GRO details)
- `financial_resolution` (refund amount, payment method, transaction ref)
- `full_ondc_payload` (for callback reconstruction)
- `category`, `sub_category`, `description`
- `complainant_info` (buyer NP information)
- `order_details` (order ID, item IDs, fulfillment IDs)

---

## 5. Main Flows & Logic

### 5.1 `/search` Flow

**Purpose**: Serviceability and quote request

**Delta from Common Request Processing Contract:**
- **Unique Processing**: Generate search_id (Serviceability ID) for request correlation (UOIS Gateway generates search_id, never derives from request payload)
- **Coordinate Transformation**: 
  - **ONDC Format (Incoming)**: `fulfillment.start.location.gps` = "lat,lng" (comma-separated), `fulfillment.end.location.gps` = "lat,lng"
  - **Internal Format (To Domain Services)**: Transform to `origin_lat`, `origin_lng`, `destination_lat`, `destination_lng` (NOT `pickup_lat`, `pickup_lng`, `drop_lat`, `drop_lng`)
  - **Transformation Required**: All events published to Location Service, Quote Service, Order Service MUST use internal coordinate naming (field name transformation only, not ID mapping)
- **Event Publishing**: Publish `SEARCH_REQUESTED` event to stream `stream.location.search` with transformed coordinates
- **Event Consumption**: Consume `QUOTE_COMPUTED` events from stream `quote:computed` using consumer group `uois-gateway-consumers`, filter by `search_id` correlation
- **Response Composition**: Quote Service passes through serviceability fields from `SERVICEABILITY_FOUND` to `QUOTE_COMPUTED`, so UOIS only needs to consume from `quote:computed` stream
- **Field Transformation**: Transform `eta_*` fields to `tat_*` (ONDC-compliant: eta_origin → tat_to_pickup, eta_destination → tat_to_drop, field name transformation only)
- **Timeout Handling**: Return "serviceable: false" response if QUOTE_COMPUTED not received within TTL

```pseudo
function HandleSearchRequest(request):
    // 1. Edge Processing (from Common Contract)
    traceparent = GenerateOrExtractTraceparent(request.headers)
    client_id, err = AuthenticateClient(request)
    if err != nil:
        return NACK(401, error_code=65002)
    
    err = ValidateRequestStructure(request)
    if err != nil:
        return NACK(400, error_code=65001)
    
    // 2. Generate search_id
    search_id = GenerateUUID()
    
    // 3. Immediate Response
    return HTTP_200_OK_ACK()
    
    // 4. Asynchronous Processing
    go ProcessSearchAsync(request, traceparent, client_id, search_id)
end

function ProcessSearchAsync(request, traceparent, client_id, search_id):
    // Extract coordinates from ONDC format and translate to internal format
    // ONDC format: fulfillment.start.location.gps = "lat,lng" (comma-separated string)
    // ONDC format: fulfillment.end.location.gps = "lat,lng" (comma-separated string)
    ondc_start_gps = ExtractONDCGPS(request.message.intent.fulfillment.start.location.gps)  // "12.453544,77.928379"
    ondc_end_gps = ExtractONDCGPS(request.message.intent.fulfillment.end.location.gps)      // "12.453544,77.928379"
    
    // Transform ONDC format to internal format (required for domain services, field name transformation only)
    origin_lat, origin_lng = ParseGPSString(ondc_start_gps)      // Parse "lat,lng" → (lat, lng)
    destination_lat, destination_lng = ParseGPSString(ondc_end_gps)  // Parse "lat,lng" → (lat, lng)
    
    // Publish SEARCH_REQUESTED event with internal coordinate naming
    event = {
        event_type: "SEARCH_REQUESTED",
        event_id: GenerateUUID(),
        search_id: search_id,
        origin_lat: origin_lat,           // Internal format (NOT pickup_lat)
        origin_lng: origin_lng,           // Internal format (NOT pickup_lng)
        destination_lat: destination_lat, // Internal format (NOT drop_lat)
        destination_lng: destination_lng, // Internal format (NOT drop_lng)
        traceparent: traceparent,
        timestamp: Now()
    }
    PublishEvent("stream.location.search", event)
    
    // Consume QUOTE_COMPUTED events from consumer group
    // Filter by search_id (lifecycle correlation ID, NOT WebSocket correlation_id)
    timeout = ParseTTL(request.context.ttl)  // Typically PT30S = 30 seconds
    quote_event = ConsumeFromStream("quote:computed", "uois-gateway-consumers", search_id, timeout)
    // Implementation: XREADGROUP with BLOCK timeout, filter events by search_id, ACK after processing
    
    if quote_event == nil:
        // Timeout - return serviceable: false
        callback_payload = {
            serviceable: false,
            search_id: search_id,
            message: "Service temporarily unavailable. Please try again.",
            requires_research: true
        }
    else:
        // Compose response from QUOTE_COMPUTED event
        callback_payload = {
            serviceable: quote_event.serviceable,
            search_id: search_id,
            distance_origin_to_destination: quote_event.distance_origin_to_destination,
            tat_to_pickup: quote_event.eta_origin,  // Transform eta_origin → tat_to_pickup (field name transformation only)
            tat_to_drop: quote_event.eta_destination,  // Transform eta_destination → tat_to_drop (field name transformation only)
            price: {
                value: quote_event.price.value,
                currency: quote_event.price.currency
            },
            expires_in: quote_event.ttl_seconds
        }
    
    // Send callback
    callback_url = ConstructCallbackURL(request.bap_uri, "on_search")
    SendCallbackWithRetry(callback_url, callback_payload, request.context.ttl)
    
    // Audit logging
    PersistAuditLog(request, callback_payload, traceparent, search_id)
end
```

### 5.2 `/init` Flow

**Purpose**: Quote initialization request

**Delta from Common Request Processing Contract:**
- **Pre-Processing**: Call Order Service (gRPC) to validate search_id TTL and quote validity (return immediate NACK if validation fails)
- **Unique Processing**: Extract `provider.id` from `message.order.provider.id` (echoed from `/on_search`), validate it matches configured stable provider identifier, use internal `search_id` (from order record) for event correlation
- **Event Publishing**: Publish `INIT_REQUESTED` event to stream `stream.uois.init_requested` with internal `search_id`
- **Event Consumption**: Consume `QUOTE_CREATED` events from stream `stream.uois.quote_created` or `QUOTE_INVALIDATED` events from stream `stream.uois.quote_invalidated` using consumer group `uois-gateway-consumers`, filter by `search_id` correlation
- **Response Composition**: Extract quote_id, price, eta fields, ttl (PT15M quote validity period) from `QUOTE_CREATED` event
- **Quote Persistence**: Generate `quote_id` and persist quote keyed by `quote_id` (canonical transactional identifier)
- **ID Propagation**: Send `quote_id` ONLY in `message.order.quote.id` (first-class ONDC field) - NO tags, NO custom fields
- **Order Record Storage**: Store `quote_id` alongside `search_id` on the same order record (no mapping between them)

```pseudo
function HandleInitRequest(request):
    // 1. Edge Processing (from Common Contract)
    traceparent = GenerateOrExtractTraceparent(request.headers)
    client_id, err = AuthenticateClient(request)
    if err != nil:
        return NACK(401, error_code=65002)
    
    err = ValidateRequestStructure(request)
    if err != nil:
        return NACK(400, error_code=65001)
    
    // 2. Extract provider.id from echoed location
    provider_id = ExtractProviderID(request)  // From message.order.provider.id (echoed from /on_search)
    
    // 3. Validate provider.id matches configured stable provider identifier
    if provider_id != CONFIGURED_PROVIDER_ID:
        return NACK(400, error_code=65001)  // Invalid provider.id
    
    // 4. Lookup internal search_id from order record (internal correlation, not from ONDC payload)
    order_record = LookupOrderRecordByProviderID(provider_id)  // Internal lookup
    if order_record == nil:
        return NACK(400, error_code=65001)  // Order record not found
    
    search_id = order_record.search_id  // Internal-only identifier
    
    // 5. Pre-Processing: Validate search_id TTL
    err = OrderService.ValidateSearchIDTTL(search_id)
    if err != nil:
        return NACK(400, error_code=65005)  // Quote Invalid
    
    // 6. Immediate Response
    return HTTP_200_OK_ACK()
    
    // 7. Asynchronous Processing
    go ProcessInitAsync(request, traceparent, client_id, search_id)
end

function ProcessInitAsync(request, traceparent, client_id, search_id):
    // Extract coordinates from ONDC format and translate to internal format
    // ONDC format: fulfillment.start.location.gps = "lat,lng" (comma-separated string)
    // ONDC format: fulfillment.end.location.gps = "lat,lng" (comma-separated string)
    ondc_start_gps = ExtractONDCGPS(request.message.order.fulfillment.start.location.gps)
    ondc_end_gps = ExtractONDCGPS(request.message.order.fulfillment.end.location.gps)
    
    // Transform ONDC format to internal format (required for domain services, field name transformation only)
    origin_lat, origin_lng = ParseGPSString(ondc_start_gps)
    destination_lat, destination_lng = ParseGPSString(ondc_end_gps)
    
    // Extract other request details
    origin_address = ExtractAddress(request.message.order.fulfillment.start.location.address)
    destination_address = ExtractAddress(request.message.order.fulfillment.end.location.address)
    package_info = ExtractPackageInfo(request)
    
    // Publish INIT_REQUESTED event with internal coordinate naming
    event = {
        event_type: "INIT_REQUESTED",
        event_id: GenerateUUID(),
        search_id: search_id,
        origin_lat: origin_lat,           // Internal format (NOT pickup_lat)
        origin_lng: origin_lng,           // Internal format (NOT pickup_lng)
        origin_address: origin_address,
        destination_lat: destination_lat, // Internal format (NOT drop_lat)
        destination_lng: destination_lng, // Internal format (NOT drop_lng)
        destination_address: destination_address,
        package_info: package_info,
        traceparent: traceparent,
        timestamp: Now()
    }
    PublishEvent("stream.uois.init_requested", event)
    
    // Consume QUOTE_CREATED or QUOTE_INVALIDATED events from consumer groups
    // Filter by search_id (lifecycle correlation ID, NOT WebSocket correlation_id)
    timeout = ParseTTL(request.context.ttl)  // Typically PT30S = 30 seconds
    quote_event = ConsumeFromStream("stream.uois.quote_created", "uois-gateway-consumers", search_id, timeout)
    invalidated_event = ConsumeFromStream("stream.uois.quote_invalidated", "uois-gateway-consumers", search_id, timeout)
    // Implementation: XREADGROUP with BLOCK timeout, filter events by search_id, ACK after processing
    
    if quote_event != nil:
        // Success - compose response
        callback_payload = {
            quote_id: quote_event.quote_id,
            price: quote_event.price,
            distance_origin_to_destination: quote_event.distance_origin_to_destination,
            eta_origin: quote_event.eta_origin,
            eta_destination: quote_event.eta_destination,
            expires_in: ParseDuration(quote_event.ttl)  // PT15M = 15 minutes
        }
        
        // Persist quote keyed by quote_id (canonical transactional identifier)
        quote_store.Store(quote_event.quote_id, {
            quote_id: quote_event.quote_id,
            price: quote_event.price,
            ttl: "PT15M",
            expires_at: Now() + 15_minutes,
            search_id: search_id,
            timestamp: Now()
        })
        
        // Store quote_id alongside search_id on the same order record
        UpdateOrderRecord(search_id, quote_id=quote_event.quote_id)
    else if invalidated_event != nil:
        // Failure - compose error response
        callback_payload = {
            quote_id: invalidated_event.quote_id,
            error: invalidated_event.error,
            message: invalidated_event.message,
            requires_research: invalidated_event.requires_research
        }
    else:
        // Timeout
        callback_payload = {
            error: "TIMEOUT",
            message: "Quote computation timed out. Please try again.",
            requires_research: true
        }
    
    // Send callback
    callback_url = ConstructCallbackURL(request.bap_uri, "on_init")
    SendCallbackWithRetry(callback_url, callback_payload, request.context.ttl)
    
    // Audit logging
    PersistAuditLog(request, callback_payload, traceparent, search_id, quote_event.quote_id if quote_event else nil)
end
```

### 5.3 `/confirm` Flow

**Purpose**: Order confirmation request

**Delta from Common Request Processing Contract:**
- **Input Processing**: Extract `quote_id` from `message.order.quote.id` (echoed from `/on_init` - canonical transactional identifier)
- **Quote Lookup & Validation**: 
  - Lookup quote by `quote_id` (primary correlation key)
  - Validate: existence, TTL not expired (PT15M), pricing integrity
  - If missing or invalid → return NACK immediately (error code `65005`)
- **Event Publishing**: Publish `CONFIRM_REQUESTED` event to stream `stream.uois.confirm_requested` with `quote_id`
- **Event Consumption**: Consume `ORDER_CONFIRMED` events from stream `stream.uois.order_confirmed` or `ORDER_CONFIRM_FAILED` events from stream `stream.uois.order_confirm_failed` using consumer group `uois-gateway-consumers`, filter by `quote_id` correlation
- **Order ID Generation**: When `ORDER_CONFIRMED` event received, generate ONDC `order.id` (seller-generated, network-facing)
- **Order Record Storage**: On `ORDER_CONFIRMED`, store `order.id` (ONDC) alongside `quote_id` and `dispatch_order_id` (internal-only) on the same order record
- **Response Composition**: Include rider assignment status, order.id (ONDC), rider_id if assigned from `ORDER_CONFIRMED` event
- **Order Lifecycle**: Does NOT block on rider assignment (async callback when assignment completes)
- **Critical**: Quote correlation MUST be done using `message.order.quote.id` - NO tag-based correlation, NO custom fields
- **Critical**: `dispatch_order_id` is internal-only and MUST NEVER be sent in ONDC payloads

```pseudo
function HandleConfirmRequest(request):
    // 1. Edge Processing (from Common Contract)
    traceparent = GenerateOrExtractTraceparent(request.headers)
    client_id, err = AuthenticateClient(request)
    if err != nil:
        return NACK(401, error_code=65002)
    
    err = ValidateRequestStructure(request)
    if err != nil:
        return NACK(400, error_code=65001)
    
    // 2. Extract quote_id
    quote_id = ExtractQuoteID(request)  // From message.order.quote.id (echoed from /on_init - canonical transactional identifier)
    
    // 3. Lookup quote by quote_id (primary correlation key)
    quote = quote_store.Get(quote_id)
    if quote == nil:
        return NACK(400, error_code=65005)  // Quote Invalid
    
    // 4. Validate quote TTL not expired (PT15M)
    if quote.expires_at < Now():
        return NACK(400, error_code=65004)  // Quote Expired
    
    // 5. Validate pricing integrity (if applicable)
    if !ValidatePricingIntegrity(quote, request):
        return NACK(400, error_code=65005)  // Quote Invalid
    
    // 6. Immediate Response
    return HTTP_200_OK_ACK()
    
    // 7. Asynchronous Processing
    go ProcessConfirmAsync(request, traceparent, client_id, quote_id)
end

function ProcessConfirmAsync(request, traceparent, client_id, quote_id):
    // Extract payment info
    payment_info = ExtractPaymentInfo(request)
    
    // Publish CONFIRM_REQUESTED event
    event = {
        event_type: "CONFIRM_REQUESTED",
        event_id: GenerateUUID(),
        quote_id: quote_id,
        client_id: client_id,
        payment_info: payment_info,
        traceparent: traceparent,
        timestamp: Now()
    }
    PublishEvent("stream.uois.confirm_requested", event)
    
    // Consume ORDER_CONFIRMED or ORDER_CONFIRM_FAILED events from consumer groups
    // Filter by quote_id (lifecycle correlation ID, NOT WebSocket correlation_id)
    timeout = ParseTTL(request.context.ttl)  // Typically PT30S = 30 seconds, but may be longer for /confirm
    confirmed_event = ConsumeFromStream("stream.uois.order_confirmed", "uois-gateway-consumers", quote_id, timeout)
    failed_event = ConsumeFromStream("stream.uois.order_confirm_failed", "uois-gateway-consumers", quote_id, timeout)
    // Implementation: XREADGROUP with BLOCK timeout, filter events by quote_id, ACK after processing
    
    if confirmed_event != nil:
        // Generate ONDC order.id (seller-generated, network-facing)
        order_id_ondc = GenerateONDCOrderID()
        
        // Success - compose response
        callback_payload = {
            quote_id: quote_id,
            order_id: order_id_ondc,  // ONDC order.id (NOT dispatch_order_id)
            rider_assigned: true,
            rider_id: confirmed_event.rider_id
        }
        
        // Store order.id (ONDC) alongside quote_id and dispatch_order_id on the same order record
        UpdateOrderRecord(quote_id, order_id_ondc=order_id_ondc, dispatch_order_id=confirmed_event.dispatch_order_id)
    else if failed_event != nil:
        // Failure - compose error response
        callback_payload = {
            quote_id: quote_id,
            rider_assigned: false,
            message: failed_event.reason,
            requires_research: true
        }
    else:
        // Timeout
        callback_payload = {
            quote_id: quote_id,
            error: "TIMEOUT",
            message: "Order confirmation timed out. Please check status later.",
            requires_research: true
        }
    
    // Send callback
    callback_url = ConstructCallbackURL(request.bap_uri, "on_confirm")
    SendCallbackWithRetry(callback_url, callback_payload, request.context.ttl)
    
    // Audit logging
    PersistAuditLog(request, callback_payload, traceparent, quote_id, confirmed_event.dispatch_order_id if confirmed_event else nil)
end
```

### 5.3.1 Detailed `/on_init` Implementation

**Purpose**: Generate and propagate `quote_id` as canonical transactional identifier

**Implementation Steps**:

1. **Generate `quote_id`**:
   - Order Service generates `quote_id` during quote creation
   - UOIS Gateway receives `quote_id` via `QUOTE_CREATED` event

2. **Persist Quote Keyed by `quote_id`**:
   ```pseudo
   quote_store.Store(quote_id, {
       quote_id: quote_id,
       price: quote_event.price,
       ttl: "PT15M",
       expires_at: Now() + 15_minutes,
       search_id: search_id,
       timestamp: Now()
   })
   ```

3. **Send `quote_id` in `/on_init` Callback**:
   - **ONLY location**: `message.order.quote.id` (first-class ONDC field)
   - **MUST include**: TTL (PT15M) in quote response
   - **MUST NOT**: Send `quote_id` in tags, custom fields, or any other location
   - **MUST NOT**: Send internal `search_id` in ONDC payload

4. **Response Structure**:
   ```json
   {
     "message": {
       "order": {
         "quote": {
           "id": "quote_id_here",  // ONLY location for quote_id
           "price": {...},
           "ttl": "PT15M"
         }
       }
     }
   }
   ```

**Critical Rules**:
- `quote_id` is the canonical transactional identifier
- Sent ONLY in `message.order.quote.id` (first-class ONDC field)
- NO tag-based correlation
- NO custom fields for internal IDs

### 5.3.2 Detailed `/confirm` Implementation

**Purpose**: Lookup quote by `quote_id` and validate before order creation

**Implementation Steps**:

1. **Extract `quote_id`**:
   ```pseudo
   quote_id = ExtractFromRequest("message.order.quote.id")  // Echoed from /on_init
   ```

2. **Lookup Quote by `quote_id`** (Primary Correlation Key):
   ```pseudo
   quote = quote_store.Get(quote_id)
   if quote == nil:
       return NACK(400, error_code=65005)  // Quote Invalid
   ```

3. **Validate Quote**:
   ```pseudo
   // Validate existence
   if quote == nil:
       return NACK(400, error_code=65005)
   
   // Validate TTL not expired
   if quote.expires_at < Now():
       return NACK(400, error_code=65004)  // Quote Expired
   
   // Validate pricing integrity (if applicable)
   if !ValidatePricingIntegrity(quote, request):
       return NACK(400, error_code=65005)  // Quote Invalid
   ```

4. **Publish `CONFIRM_REQUESTED` Event**:
   ```pseudo
   event = {
       quote_id: quote_id,  // Use validated quote_id
       client_id: client_id,
       payment_info: payment_info,
       traceparent: traceparent
   }
   PublishEvent("stream.uois.confirm_requested", event)
   ```

5. **Generate ONDC `order.id` on `ORDER_CONFIRMED`**:
   ```pseudo
   // When ORDER_CONFIRMED event received
   order_id_ondc = GenerateONDCOrderID()  // Seller-generated, network-facing
   // Send in /on_confirm → message.order.id (NOT dispatch_order_id)
   ```

**Critical Rules**:
- Quote correlation MUST be done using `message.order.quote.id`
- Lookup quote by `quote_id` (primary correlation key)
- Validate: existence, TTL not expired (PT15M), pricing integrity
- NO tag-based correlation
- NO custom fields for internal IDs
- NO fallback resolution

### 5.4 Post-Confirmation Flows (`/status`, `/track`, `/cancel`, `/update`, `/rto`)

**Common Pattern**: All post-confirmation flows inherit request validation, ID propagation via echo contract, ACK/callback semantics, TTL handling, retries, and audit logging from Common Request Processing Contract.

**Delta from Common Request Processing Contract:**
- **Input Processing**: Extract ONDC `order.id` from `message.order_id` (echoed from `/on_confirm` - seller-generated), look up order record using `order.id` (ONDC), retrieve `dispatch_order_id` from order record
- **Service Call**: Order Service gRPC method (GetOrder, GetOrderTracking, CancelOrder, UpdateOrder, InitiateRTO)
- **Response Composition**: Transform order state, rider info, timeline, fulfillment states to ONDC format
- **Critical**: `order.id` (ONDC) is the canonical post-confirmation identifier - Buyer echoes seller-generated `order.id`

```pseudo
function HandlePostConfirmationRequest(action, request):
    // 1. Edge Processing (from Common Contract)
    traceparent = GenerateOrExtractTraceparent(request.headers)
    client_id, err = AuthenticateClient(request)
    if err != nil:
        return NACK(401, error_code=65002)
    
    err = ValidateRequestStructure(request)
    if err != nil:
        return NACK(400, error_code=65001)
    
    // 2. Extract ONDC order.id (echoed from /on_confirm - seller-generated)
    order_id_ondc = ExtractOrderID(request)  // From message.order_id (echoed from /on_confirm)
    
    // Look up order record using order.id (ONDC)
    order_record = LookupOrderRecord(order_id_ondc)
    if order_record == nil:
        return NACK(404, error_code=65006)  // Order Not Found
    
    dispatch_order_id = order_record.dispatch_order_id  // Retrieve from order record
    
    // 3. Immediate Response
    return HTTP_200_OK_ACK()
    
    // 4. Asynchronous Processing
    go ProcessPostConfirmationAsync(action, request, traceparent, client_id, dispatch_order_id)
end

function ProcessPostConfirmationAsync(action, request, traceparent, client_id, dispatch_order_id):
    // Call Order Service gRPC
    grpc_method = MapActionToGRPCMethod(action)  // GetOrder, GetOrderTracking, CancelOrder, UpdateOrder, InitiateRTO
    grpc_response, err = CallOrderService(grpc_method, dispatch_order_id, request)
    
    if err != nil:
        callback_payload = {
            error: MapGRPCErrorToONDCError(err),
            message: err.message
        }
    else:
        // Transform gRPC response to ONDC format
        callback_payload = TransformToONDCFormat(grpc_response, action)
    
    // Send callback
    callback_url = ConstructCallbackURL(request.bap_uri, "on_{action}")
    SendCallbackWithRetry(callback_url, callback_payload, request.context.ttl)
    
    // Audit logging
    PersistAuditLog(request, callback_payload, traceparent, dispatch_order_id)
end
```

### 5.5 Issue & Grievance Management (IGM) Flows

#### 5.5.1 `/issue` Endpoint

**Purpose**: Receive issues from Buyer NPs

**Note**: Buyers have their own ticket dashboard and create tickets using endpoints which should be proxied to Zendesk. The `/issue` endpoint receives ONDC-compliant issue requests from Buyer NPs and creates corresponding tickets in Zendesk.

```pseudo
function HandleIssueRequest(request):
    // 1. Edge Processing
    traceparent = GenerateOrExtractTraceparent(request.headers)
    client_id, err = AuthenticateClient(request)
    if err != nil:
        return NACK(401, error_code=65002)
    
    err = ValidateIssueRequest(request)
    if err != nil:
        return NACK(400, error_code=65001)
    
    // 2. Extract issue details
    issue_id = ExtractIssueID(request)
    category = ExtractCategory(request)
    sub_category = ExtractSubCategory(request)
    description = ExtractDescription(request)
    complainant_info = ExtractComplainantInfo(request)
    order_details = ExtractOrderDetails(request)
    
    // 3. Immediate Response
    return HTTP_200_OK_ACK()
    
    // 4. Asynchronous Processing
    go ProcessIssueAsync(request, traceparent, client_id, issue_id, category, sub_category, description, complainant_info, order_details, zendeskService, issueRepo)
end

function ProcessIssueAsync(request, traceparent, client_id, issue_id, category, sub_category, description, complainant_info, order_details, zendeskService, issueRepo):
    // Store issue in Redis via issue repository
    issue_data = {
        ondc_issue_id: issue_id,
        category: category,
        sub_category: sub_category,
        description: description,
        complainant_info: complainant_info,
        order_details: order_details,
        status: "OPEN",
        created_at: Now(),
        full_ondc_payload: request
    }
    issueRepo.StoreIssue(issue_id, issue_data, TTL=30_DAYS)
    
    // Create Zendesk ticket via ZendeskService (proxied from buyer ticket dashboard endpoint)
    // Buyers create tickets via their own dashboard, which are proxied to Zendesk
    zendesk_ticket, err = zendeskService.CreateTicket(issue_data)
    if err != nil:
        // Log error, but continue processing (graceful degradation)
        LogError("Failed to create Zendesk ticket", err, issue_id)
    else:
        issueRepo.StoreZendeskReference(zendesk_ticket.id, issue_id)
    
    // Compose callback response
    callback_payload = {
        issue_id: issue_id,
        status: "OPEN",
        issue_actions: BuildIssueActions(issue_data)
    }
    
    // Send callback
    callback_url = ConstructCallbackURL(request.bap_uri, "on_issue")
    SendCallbackWithRetry(callback_url, callback_payload, request.context.ttl)
    
    // Audit logging
    PersistAuditLog(request, callback_payload, traceparent, issue_id)
end
```

#### 5.5.2 `/issue_status` Endpoint

**Purpose**: Handle status check requests

```pseudo
function HandleIssueStatusRequest(request):
    // 1. Edge Processing
    traceparent = GenerateOrExtractTraceparent(request.headers)
    client_id, err = AuthenticateClient(request)
    if err != nil:
        return NACK(401, error_code=65002)
    
    issue_id = ExtractIssueID(request)
    if issue_id == nil:
        return NACK(400, error_code=65001)
    
    // 2. Immediate Response
    return HTTP_200_OK_ACK()
    
    // 3. Asynchronous Processing
    go ProcessIssueStatusAsync(request, traceparent, client_id, issue_id, issueRepo, groService)
end

function ProcessIssueStatusAsync(request, traceparent, client_id, issue_id, issueRepo, groService):
    // Retrieve issue from Redis via issue repository
    issue_data = issueRepo.GetIssue(issue_id)
    if issue_data == nil:
        callback_payload = {
            error: "ISSUE_NOT_FOUND",
            message: "Issue not found"
        }
    else:
        // Get GRO details via GROService
        gro_details = groService.GetGRODetails(issue_data.issue_type)
        
        // Compose callback response
        callback_payload = {
            issue_id: issue_id,
            status: issue_data.status,
            issue_actions: BuildIssueActions(issue_data),
            gro_details: gro_details,
            resolution_details: issue_data.resolution_provider if issue_data.status == "CLOSED" else nil
        }
    
    // Send callback
    callback_url = ConstructCallbackURL(request.bap_uri, "on_issue_status")
    SendCallbackWithRetry(callback_url, callback_payload, request.context.ttl)
    
    // Audit logging
    PersistAuditLog(request, callback_payload, traceparent, issue_id)
end
```

#### 5.5.3 Zendesk Webhook Handler

**Purpose**: Receive ticket updates from Zendesk Helpdesk

```pseudo
function HandleZendeskWebhook(request, issueRepo, zendeskService, groService, callbackService):
    // 1. Validate webhook signature
    err = ValidateWebhookSignature(request.headers["X-Zendesk-Webhook-Signature"])
    if err != nil:
        return HTTP_401_UNAUTHORIZED()
    
    // 2. Extract ticket update details
    ticket_id = ExtractTicketID(request)
    status = ExtractStatus(request)
    resolution = ExtractResolution(request)
    updated_at = ExtractUpdatedAt(request)
    
    // 3. Map Zendesk status to ONDC status via ZendeskService
    ondc_status = zendeskService.MapZendeskStatusToONDC(status)
    
    // 4. Lookup issue_id from ticket reference via issue repository
    issue_id = issueRepo.GetIssueIDByZendeskTicket(ticket_id)
    if issue_id == nil:
        return HTTP_404_NOT_FOUND()
    
    // 5. Update issue status in Redis via issue repository
    issue_data = issueRepo.GetIssue(issue_id)
    issue_data.status = ondc_status
    issue_data.resolution_provider = resolution
    issue_data.updated_at = updated_at
    issueRepo.UpdateIssue(issue_id, issue_data)
    
    // 6. Trigger ONDC callback
    bap_uri = ExtractBapURI(issue_data.full_ondc_payload)
    gro_details = groService.GetGRODetails(issue_data.issue_type)
    callback_payload = {
        issue_id: issue_id,
        status: ondc_status,
        issue_actions: BuildIssueActions(issue_data),
        gro_details: gro_details,
        resolution_details: resolution if ondc_status == "CLOSED" else nil
    }
    
    callback_url = ConstructCallbackURL(bap_uri, "on_issue_status")
    callbackService.SendCallbackWithRetry(callback_url, callback_payload, DEFAULT_TTL)
    
    // 7. Audit logging
    PersistAuditLog(request, callback_payload, nil, issue_id)
end
```

---

## 6. Integration Points

### 6.1 Location Service

**Communication Pattern**: Event-driven (Redis Streams)

**Events Published to Location Service:**
- `SEARCH_REQUESTED` → `stream.location.search`

**Events Consumed from Location Service:**
- None directly (Location Service publishes `SERVICEABILITY_FOUND` to Quote Service, Quote Service passes through to `QUOTE_COMPUTED`)

### 6.2 Quote Service

**Communication Pattern**: Event-driven (Redis Streams)

**Events Consumed from Quote Service:**
- `QUOTE_COMPUTED` → `quote:computed`

**Event Payload (QUOTE_COMPUTED):**
- `event_type`: "QUOTE_COMPUTED"
- `search_id`: UUID (correlation key)
- `serviceable`: boolean (pass-through from SERVICEABILITY_FOUND)
- `price`: { value: number, currency: string }
- `ttl`: ISO8601 duration (e.g., "PT10M")
- `ttl_seconds`: integer (e.g., 600)
- `eta_origin`: ISO8601 timestamp (pass-through from SERVICEABILITY_FOUND)
- `eta_destination`: ISO8601 timestamp (pass-through from SERVICEABILITY_FOUND)
- `distance_origin_to_destination`: number (pass-through from SERVICEABILITY_FOUND)
- `timestamp`: ISO8601 timestamp
- `traceparent`: W3C traceparent header
- `trace_id`: string (extracted from traceparent)

### 6.3 Order Service

**Communication Pattern**: 
- Event-driven (Redis Streams) for pre-order flows
- gRPC for post-confirmation flows

**Events Published to Order Service:**
- `INIT_REQUESTED` → `stream.uois.init_requested`
- `CONFIRM_REQUESTED` → `stream.uois.confirm_requested`

**Events Consumed from Order Service:**
- `QUOTE_CREATED` → `stream.uois.quote_created`
- `QUOTE_INVALIDATED` → `stream.uois.quote_invalidated`
- `ORDER_CONFIRMED` → `stream.uois.order_confirmed`
- `ORDER_CONFIRM_FAILED` → `stream.uois.order_confirm_failed`

**gRPC Methods Called:**
- `GetOrder` - Fetch order status for `/status` flow
- `GetOrderTracking` - Fetch tracking data for `/track` flow
- `CancelOrder` - Cancel order for `/cancel` flow
- `UpdateOrder` - Update order for `/update` flow
- `InitiateRTO` - Initiate Return to Origin for `/rto` flow

### 6.4 Admin Service

**Communication Pattern**: 
- Event-driven (Redis Streams) for client registry sync
- gRPC for client configuration fetching (cache warm-up, not hot-path)

**Events Consumed from Admin Service:**
- `client.created` → `stream:admin.client.events`
- `client.updated` → `stream:admin.client.events`
- `client.suspended` → `stream:admin.client.events`
- `client.revoked` → `stream:admin.client.events`
- `client.api_key_rotated` → `stream:admin.client.events`

**gRPC Methods Called:**
- `GetClientConfig` - Fetch client-specific configuration (cache warm-up only)
- `AuthenticateClient` - Optional fallback for cache warm-up (not in hot-path)

### 6.5 Zendesk Helpdesk

**Communication Pattern**: HTTP REST

**Note**: Buyers have their own ticket dashboard and create tickets using endpoints which should be proxied to Zendesk.

**ZendeskService Responsibilities** (Service Layer):
- **Create Tickets**: Map ONDC issue types to Zendesk priorities (DISPUTE → Urgent, GRIEVANCE → High, ISSUE → Medium)
- **Update Ticket Status**: Sync status changes bidirectionally
- **Get Ticket Details**: Retrieve ticket details for status queries
- **Add Comments**: Add comments to Zendesk tickets from ONDC issue updates
- **Authentication**: Use Zendesk API Key/Secret (token-based auth)

**HTTP Methods (via ZendeskService):**
- `POST /api/v2/tickets.json` - Create ticket from ONDC issue (proxied from buyer ticket dashboard)
- `GET /api/v2/tickets/{ticket_id}.json` - Get ticket details
- `PUT /api/v2/tickets/{ticket_id}.json` - Update ticket status
- `POST /api/v2/tickets/{ticket_id}/comments.json` - Add comments to ticket

**Webhook Receiver:**
- `POST /webhooks/zendesk/ticket_update` - Receive ticket update webhooks from Zendesk

### 6.6 GRO (Grievance Redressal Officer) Management

**Communication Pattern**: Internal Service (Redis-backed)

**GROService Responsibilities**:
- **Store and Retrieve GRO Details**: From Redis with fallback to defaults
- **GRO Level Assignment**: L1 for ISSUE, L2 for GRIEVANCE, L3 for DISPUTE
- **Provide Default GRO Details**: If Redis lookup fails
- **Maintain GRO Contact Information**: Contact details and escalation paths

**Storage**:
- GRO details stored in Redis (TTL: 30 days)
- Default GRO details configured in service

### 6.7 Financial Notifications Integration

**Communication Pattern**: HTTP REST / Events (TBD)

**Functional Requirements** (per FR Section 9.4):
- Receive payment status notifications from Admin Backend
- Receive settlement status notifications
- Receive RTO status notifications
- Store financial resolution data in Redis (`ondc:financial:{issue_id}`)
- Update related issues with financial status information
- Support financial action tracking (refunds, settlements)
- Link financial resolutions to ONDC issues for status callbacks

**Note**: Implementation details (endpoints, event types, integration pattern) not fully defined in current FR; requires clarification.

---

## 7. Configuration & Environment Assumptions

### 7.1 Required Configuration

**Server Configuration:**
- `SERVER_PORT` (default: 8080)
- `SERVER_HOST`
- `SERVER_READ_TIMEOUT` (default: 30s)
- `SERVER_WRITE_TIMEOUT` (default: 30s)

**PostgreSQL-E (Audit Database) Configuration:**
- `POSTGRES_E_HOST`
- `POSTGRES_E_PORT` (default: 5432)
- `POSTGRES_E_USER`
- `POSTGRES_E_PASSWORD`
- `POSTGRES_E_DB` (default: postgres_audit)
- `POSTGRES_E_SSL_MODE` (default: require)
- `POSTGRES_E_MAX_CONNECTIONS` (default: 25)
- `POSTGRES_E_MAX_IDLE_CONNECTIONS` (default: 5)
- `POSTGRES_E_CONNECTION_MAX_LIFETIME` (default: 5m)

**Redis Configuration:**
- `REDIS_HOST`
- `REDIS_PORT` (default: 6379)
- `REDIS_PASSWORD`
- `REDIS_DB` (default: 0)
- `REDIS_TLS` (default: false)
- `REDIS_KEY_PREFIX` (default: uois-gateway)
- `REDIS_POOL_SIZE` (default: 10)
- `REDIS_MIN_IDLE_CONNS` (default: 5)

**Order Service gRPC:**
- `ORDER_SERVICE_GRPC_HOST`
- `ORDER_SERVICE_GRPC_PORT` (default: 50052)
- `ORDER_SERVICE_GRPC_TIMEOUT` (default: 30s)

**Admin Service gRPC:**
- `ADMIN_SERVICE_GRPC_HOST`
- `ADMIN_SERVICE_GRPC_PORT`
- `ADMIN_SERVICE_GRPC_TIMEOUT` (default: 5s)

**Event Streams:**
- `STREAM_SEARCH_REQUESTED` (default: stream.location.search)
- `STREAM_INIT_REQUESTED` (default: stream.uois.init_requested)
- `STREAM_CONFIRM_REQUESTED` (default: stream.uois.confirm_requested)
- `STREAM_QUOTE_COMPUTED` (default: quote:computed)
- `STREAM_QUOTE_CREATED` (default: stream.uois.quote_created)
- `STREAM_QUOTE_INVALIDATED` (default: stream.uois.quote_invalidated)
- `STREAM_ORDER_CONFIRMED` (default: stream.uois.order_confirmed)
- `STREAM_ORDER_CONFIRM_FAILED` (default: stream.uois.order_confirm_failed)
- `STREAM_CLIENT_EVENTS` (default: stream:admin.client.events)

**TTLs (in seconds):**
- `REQUEST_CONTEXT_TTL` (default: 3600)
- `ORDER_REFERENCE_TTL` (default: 2592000 = 30 days)
- `IDEMPOTENCY_KEY_TTL` (default: 86400 = 24 hours)
- `ISSUE_STORAGE_TTL` (default: 2592000 = 30 days)
- `CLIENT_CONFIG_CACHE_TTL` (default: 900 = 15 minutes)
- `CLIENT_REGISTRY_CACHE_TTL` (default: 300 = 5 minutes)

**Retry Configuration:**
- `CALLBACK_MAX_RETRIES` (default: 5)
- `CALLBACK_RETRY_BACKOFF_1S` (default: 1s)
- `CALLBACK_RETRY_BACKOFF_2S` (default: 2s)
- `CALLBACK_RETRY_BACKOFF_4S` (default: 4s)
- `CALLBACK_RETRY_BACKOFF_8S` (default: 8s)
- `CALLBACK_RETRY_BACKOFF_15S` (default: 15s) - Adjusted to fit within ONDC Request TTL (PT30S)
- `ORDER_SERVICE_MAX_RETRIES` (default: 3)
- `ADMIN_SERVICE_MAX_RETRIES` (default: 3)
- `EVENT_PUBLISH_MAX_RETRIES` (default: 3)

**TTL-Aware Defaults for ONDC Flows:**
- `ONDC_REQUEST_TTL_SECONDS` (default: 30) - ONDC Request TTL: PT30S (30 seconds) - callback delivery deadline
- `ONDC_QUOTE_TTL_SECONDS` (default: 900) - ONDC Quote TTL: PT15M (15 minutes) - quote validity period
- **Formula**: Total retry duration = `sum(CALLBACK_RETRY_BACKOFF_*) <= ONDC_REQUEST_TTL_SECONDS`
- **Example Calculation**: 1s + 2s + 4s + 8s + 15s = 30s (within ONDC Request TTL limit)
- **Note**: Original 16s backoff (1s + 2s + 4s + 8s + 16s = 31s) exceeds PT30S limit, so last retry adjusted to 15s

**ONDC Configuration:**
- `ONDC_NETWORK_REGISTRY_URL` (default: https://registry.ondc.org)
- `ONDC_PRIVATE_KEY_PATH`
- `ONDC_PUBLIC_KEY_PATH`
- `ONDC_TIMESTAMP_WINDOW` (default: 300)

**Zendesk Helpdesk Configuration:**
- `ZENDESK_API_URL` - Zendesk API endpoint URL
- `ZENDESK_API_EMAIL` - Zendesk API email for authentication
- `ZENDESK_API_TOKEN` - Zendesk API token for authentication
- `ZENDESK_WEBHOOK_SECRET` - Webhook signature secret for validation

**Logging:**
- `LOG_LEVEL` (default: info)
- `LOG_ENCODING` (default: json)

**Distributed Tracing:**
- `TRACING_ENABLED` (default: true)
- `TRACING_SAMPLE_RATE` (default: 0.1)
- `JAEGER_ENDPOINT` (default: http://localhost:14268/api/traces)

**Rate Limiting:**
- `RATE_LIMIT_ENABLED` (default: true)
- `RATE_LIMIT_REDIS_KEY_PREFIX` (default: rate_limit:uois)

### 7.2 Configuration Validation

**Config.Validate() Requirements**:
- Application MUST validate all critical configuration at startup and fail early if any required configuration is missing or invalid
- **Required Validations**:
  - Postgres-E connection string and database accessibility
  - Redis connection and cluster accessibility
  - Order Service gRPC endpoint (`ORDER_SERVICE_GRPC_*` configuration)
  - Admin Service gRPC endpoint (`ADMIN_SERVICE_GRPC_*` configuration)
  - ONDC key paths and signing configuration
  - All mandatory environment variables must be present and valid
- **Failure Behavior**: Application MUST NOT start if any critical configuration validation fails
- **Error Reporting**: Validation failures MUST include clear error messages indicating which configuration is missing or invalid

### 7.3 Environment Assumptions

**Not defined in current FR; requires clarification.**

---

## 8. Observability, Logging, Metrics

### 8.1 Distributed Tracing

**Trace ID Generation (Edge Service Responsibility):**
- Generate W3C `traceparent` header when receiving HTTP request (if not present in incoming request)
- Format: `00-<trace-id>-<span-id>-<flags>` (W3C Trace Context standard)
- Start root span using OpenTelemetry SDK
- Extract trace_id from traceparent for logging convenience

**Trace Context Propagation:**
- Include `traceparent` in all Redis Stream events published (`SEARCH_REQUESTED`, `INIT_REQUESTED`, `CONFIRM_REQUESTED`)
- Propagate trace context in all gRPC calls to Order Service (OpenTelemetry SDK handles this automatically)
- Include `traceparent` in callback headers if useful for client support (careful re: privacy/security)

**Trace Continuity:**
- Extract `traceparent` from consumed events (`QUOTE_COMPUTED`, `QUOTE_CREATED`, `ORDER_CONFIRMED`)
- Create child spans when processing events (do NOT generate new trace IDs)
- Maintain trace continuity across sync + async hops (HTTP → Redis Streams → services → callbacks)

**Logging:**
- Always log both `trace_id` and lifecycle IDs (`search_id`, `quote_id`, `dispatch_order_id`) together
- Format: `INFO [service=uois] trace_id=4bf92f3577b34da6... search_id=550e8400-e29b-41d4... msg="published SEARCH_REQUESTED"`
- Include trace_id in audit logs (Postgres-E) for fast troubleshooting
- **Never log or use `correlation_id`** (WebSocket Gateway responsibility only)

**Sampling:**
- Default sampling at edge (sample p95/p99 traces to avoid cost explosion)
- Use adaptive sampling if necessary
- Ensure spans are created minimally for high-volume endpoints

### 8.2 ID Stack & Ownership

**UOIS Gateway Responsibilities:**
- **Generates**: `trace_id` (via W3C `traceparent` header at edge)
- **Extracts from auth**: `client_id` (from JWT/API key)
- **Generates**: `search_id` (Serviceability ID for `/search` requests, UOIS Gateway-generated, never derived from request payload)
- **Passes downstream**: `traceparent`, `client_id`, lifecycle IDs (`search_id`, `quote_id`)
- **Never generates or uses**: `correlation_id` (WebSocket Gateway responsibility only)

**ID Stack Summary:**
```
┌────────────────────────────────────┐
│ trace_id                           │  ← observability (SRE, OpenTelemetry)
├────────────────────────────────────┤
│ correlation_id                     │  ← UI / session / conversation (WebSocket)
├────────────────────────────────────┤
│ client_id                          │  ← tenant boundary (auth, billing)
├────────────────────────────────────┤
│ search_id / quote_id /             │
│ dispatch_order_id                  │  ← business lifecycle
├────────────────────────────────────┤
│ event_id                           │  ← event-level idempotency
├────────────────────────────────────┤
│ message_id (Redis Stream ID)       │  ← transport sequencing
└────────────────────────────────────┘
```

**One-line meaning:**
* **trace_id** → *What happened across services* (generated by UOIS Gateway)
* **correlation_id** → *What belongs to one UI/session* (WebSocket Gateway only)
* **client_id** → *Who owns this business* (extracted from auth)
* **lifecycle IDs** → *What the business object is* (search_id/quote_id/dispatch_order_id)
* **event_id** → *Did we already process this event* (for deduplication)
* **message_id** → *Where is this message in the stream* (Redis Streams, ACK only)

**Strict Rules:**
- **`trace_id`**: Generated by UOIS Gateway, propagated everywhere, logs + spans only, ❌ never business logic
- **`correlation_id`**: Generated by WebSocket Gateway, ❌ never stored in DB, ❌ never enters core services, ❌ UOIS Gateway never generates or uses
- **`client_id`**: Extracted from auth, passed to all core services, used for pricing/billing/multi-tenancy
- **`search_id/quote_id/dispatch_order_id`**: Pure business lifecycle identifiers that coexist on the same order record; different APIs select different IDs for lookup and correlation
- **`event_id`**: Generated by event publisher, used only for deduplication, TTL-based storage
- **`message_id`**: Generated by Redis Streams, used only for ACK/replay/lag monitoring, ❌ never stored in business tables

**Note on `message_id` terminology:**
- **ONDC `message_id`**: ONDC protocol field from request context (`context.message_id`) - used for callback correlation, stored in audit logs
- **Redis Stream `message_id`**: Redis Stream message ID (e.g., `"1234567890-0"`) - used only for ACK/replay, never stored in business tables

**One-Line Law:**
> **UOIS Gateway generates `trace_id`, extracts `client_id`, generates `search_id`, passes lifecycle IDs downstream, uses `event_id` for deduplication, and NEVER generates or uses `correlation_id` (WebSocket Gateway responsibility exclusively).**

### 8.2 Request Logging

**Log all incoming requests with:**
- Request ID (unique per request)
- Client ID
- **Trace ID** (extracted from traceparent)
- Request hash (for non-repudiation)
- Timestamp
- Request payload (sanitized)
- Response payload (sanitized)
- Processing time
- search_id (for `/search` and `/init` requests)
- Quote ID (for `/init` and `/confirm` requests)
- Dispatch Order ID (for `/confirm` requests)

### 8.3 Event Logging

**Log all events published:**
- SEARCH_REQUESTED, INIT_REQUESTED, CONFIRM_REQUESTED
- Include lifecycle IDs (search_id, quote_id, dispatch_order_id) for event correlation
- Include **traceparent** and **trace_id** in all event logs
- Include timing and status for event processing
- **Never log**: WebSocket `correlation_id` (not used by UOIS Gateway)

**Log all events consumed:**
- QUOTE_COMPUTED, QUOTE_CREATED, ORDER_CONFIRMED, ORDER_CONFIRM_FAILED
- Include lifecycle IDs (search_id, quote_id, dispatch_order_id) for event correlation
- Include **traceparent** and **trace_id** in all event logs
- Include timing and status for event processing
- **Never log**: WebSocket `correlation_id` (not used by UOIS Gateway)

### 8.4 Callback Delivery Logging

**Log all callback delivery attempts:**
- Order callbacks: `/on_search`, `/on_init`, `/on_confirm`, `/on_status`, `/on_cancel`, `/on_update`, `/on_track`
- IGM callbacks: `/on_issue`, `/on_issue_status`
- Include timing and status
- Track retry attempts and failures
- Store callback URL, ONDC correlation IDs (`transaction_id`, ONDC `message_id`), and delivery status
- **Note**: ONDC `message_id` is from ONDC `context.message_id`, NOT Redis Stream message_id (which is only used for ACK)

### 8.5 Audit Trail

**Tamper-resistant storage:**
- Request/response pairs in Postgres-E (`audit` schema)
- Request hashing for integrity verification
- Support dispute resolution with complete audit trail
- Link requests to events via search_id, quote_id, dispatch_order_id
- Include **trace_id** in audit logs for end-to-end correlation
- **Data Retention**: Minimum 7 years for financial transactions

### 8.6 Metrics

**Not explicitly defined in current FR; requires clarification.**

**Suggested metrics (if implemented):**
- Request rate, latency, error rate
- Callback delivery success/failure rate
- Event publishing/consumption rate
- Database connection pool metrics

---

## 9. Testing Strategy

### 9.1 Unit Tests

**Coverage Requirements:**
- All handlers, services, repositories, clients, utils
- Mock all external dependencies (gRPC clients, HTTP clients, Redis, Postgres)
- Use Testify framework (`testify/assert` for assertions, `testify/mock` for mocking)

**Test Scenarios:**
- Success case
- Missing headers
- Invalid signature
- Malformed JSON
- Validation failure
- Service failure
- Correct 200 response

### 9.2 Integration Tests

**Purpose**: Test full request flows with real dependencies (optional, can use testcontainers)

**Scope**: End-to-end flows (e.g., `/search` → event → callback)

### 9.3 Contract Tests

**Purpose**: Validate event schemas and API contracts

**Tools**: JSON Schema validation, Protobuf validation

### 9.4 Test Coverage Requirements

**Minimum**: 80% code coverage

**Critical paths**: 100% coverage (authentication, callback delivery, event publishing)

---

## 10. Error Handling & Standardization

### 10.1 Error Taxonomy

| Error Code | Category | HTTP Status | Retryable | Example | Action |
|------------|----------|-------------|-----------|---------|--------|
| `65001` | Validation | 400 Bad Request | No | Missing required field (`search_id`, `quote_id`, `order_id`) | Return NACK immediately |
| `65002` | Authentication | 401 Unauthorized | No | Invalid client credentials, missing `Authorization` header | Return NACK immediately |
| `65003` | Stale Request | 400 Bad Request | No | Request timestamp earlier than previously processed request (same `transaction_id` + `message_id`) | Return NACK with error code `65003` |
| `65004` | Quote Expired | 400 Bad Request | No | Quote TTL (`PT15M`) expired before `/confirm` | Return NACK, require new `/init` |
| `65005` | Quote Invalid | 400 Bad Request | No | Quote not found or invalid state | Return NACK, require new `/init` |
| `65006` | Order Not Found | 404 Not Found | No | Order record not found using `order.id` (ONDC) | Return NACK, verify order_id |
| `65007` | Invalid State Transition | 400 Bad Request | No | Order state does not allow requested operation (e.g., cancel after delivery) | Return NACK with current state |
| `65010` | Dependency Timeout | 503 Service Unavailable | Yes | Quote Service timeout, Order Service timeout | Retry with exponential backoff, return timeout after max retries |
| `65011` | Dependency Unavailable | 503 Service Unavailable | Yes | Quote Service down, Order Service down | Retry with exponential backoff, return error after max retries |
| `65012` | Rate Limit Exceeded | 429 Too Many Requests | Yes | Client rate limit exceeded | Return 429, include `Retry-After` header |
| `65020` | Internal Error | 500 Internal Server Error | No | Database error, unexpected exception | Log error, return generic error to client |
| `65021` | Callback Delivery Failed | N/A (async) | Yes | HTTP POST to `{bap_uri}/on_*` failed | Retry with exponential backoff (1s → 2s → 4s → 8s → 16s), max 5 attempts, then DLQ |

### 10.2 Error Propagation

- Propagate Order Service and Quote Service errors correctly
- Map internal service errors to UOIS error codes (see table above)
- Mask sensitive internal error details (database errors, stack traces)
- Include actionable error messages for clients
- Log full error details with `trace_id` for troubleshooting

---

## 11. Security & Non-Repudiation

### 11.1 ONDC Request/Response Signing

**ONDC Authentication Requirements** (per ONDC API Contract v1.2.0 Section 2 & 3):

**Key Pair Generation:**
- Use ed25519 for signing and X25519 for encryption
- Generate key pairs using standard libraries (e.g., libsodium)
- Update base64 encoded public keys in ONDC network registry
- Reference implementation: [ONDC Signing Utilities](https://github.com/ONDC-Official/reference-implementations/tree/main/utilities/signing_and_verification)

**Request Signature Verification** (Incoming ONDC Requests):
1. **Extract Auth Header**:
   - Parse `Authorization` header with format: `keyId="{subscriber_id}|{unique_key_id}|{algorithm}"`
   - Extract `subscriber_id`, `ukId` (unique_key_id), and `algorithm` from keyId
   - Extract encoded signature from authorization header

2. **Registry Lookup**:
   - Use ONDC network registry `/lookup` API to fetch `signing_public_key` for the `ukId`
   - Registry lookup request format:
     ```json
     {
       "subscriber_id": "lsp.com",
       "domain": "nic2004:60232",
       "ukId": "UKID1",
       "country": "IND",
       "city": "std:080",
       "type": "BPP"
     }
     ```
   - Cache public keys locally (TTL: 1 hour) to reduce registry calls
   - Support local registry cache with refresh at regular intervals

3. **Signature Verification Process**:
   - Extract the digest from the encoded signature in the request
   - Create UTF-8 byte array from the **raw payload** (original JSON request body)
   - Generate Blake2b hash from UTF-8 byte array
   - Compare generated Blake2b hash with the decoded digest from the signature
   - **On verification failure**: Return HTTP 401 Unauthorized with error code `65002`

4. **Timestamp Validation**:
   - Verify request timestamp to prevent replay attacks
   - Reject requests outside acceptable time window (configurable, default: 300 seconds)
   - Check for stale requests (timestamp earlier than previously processed request with same `transaction_id` + `message_id`)
   - Return NACK with error code `65003` for stale requests

**Response Signing** (Outgoing ONDC Responses):
1. **Generate Signature**:
   - Create UTF-8 byte array from JSON response payload
   - Generate Blake2b hash from UTF-8 byte array
   - Create base64 encoding of Blake2b hash (this becomes the digest for signing)
   - Sign the digest using gateway's private signing key (ed25519)
   - Add signature to response `Authorization` header with format: `keyId="{subscriber_id}|{unique_key_id}|{algorithm}"`

2. **Include Timestamp**:
   - Include timestamp in signed payload (from `context.timestamp`)
   - Ensure timestamp is within acceptable window for recipient

**Replay Protection:**
- Track processed request hashes (idempotency) using `transaction_id` + `message_id`
- Reject duplicate requests within time window
- Support configurable time window for timestamp validation (default: 300 seconds per `ONDC_TIMESTAMP_WINDOW` config)

**Compliance**: Must comply with ONDC network security requirements as specified in ONDC API Contract v1.2.0 and [ONDC Protocol Network Extension documentation](https://docs.google.com/document/d/1-xECuAHxzpfF8FEZw9iN3vT7D3i6yDDB1u2dEApAjPA/edit).

### 11.2 Client Authentication

**Runtime Authentication Flow:**
1. **Extract Credentials**:
   - Parse `Authorization` header:
     - `Basic` auth: `base64(client_id:client_secret)` → extract `client_id` and `client_secret`
     - `Bearer` token: Single opaque API key → extract `client_id` from key format or lookup
   - Extract client IP from `X-Real-IP` or `X-Forwarded-For` (trusted proxy headers, not `req.RemoteAddr`)

2. **Lookup Client Registry** (local, no cross-service call):
   - Check Redis cache first: `client:{client_id}`
   - If cache miss, query local Postgres-E: `client_registry` table
   - If still not found, optional fallback: Single API call to Admin Service (cache warm-up only, not hot-path)

3. **Validate Credentials**:
   - Compare provided `client_secret` with `client_secret_hash` using bcrypt/argon2
   - Check `status == 'ACTIVE'` (reject if SUSPENDED or REVOKED)
   - Validate IP against `allowed_ips` (CIDR array)
   - Check rate limits (if configured)

4. **Enrich Request Context**:
   - Attach `client_id`, `client_code`, `client_info` to request context
   - Proceed with request processing

### 11.3 Rate Limiting

**Functional Requirements:**
- Apply per-client rate limiting
- Return HTTP 429 when rate limit exceeded
- Support configurable rate limits per client
- Include rate limit headers in responses
- Log rate limit violations for monitoring

---

## 12. Storage & Caching

### 12.1 Temporary Storage (Redis)

**Order References:**
- Order record stores `quote_id` alongside `search_id` (created in `/init`, search_id UOIS Gateway-generated, quote_id Order Service-generated)
- Order record stores `order.id` (ONDC) alongside `quote_id` and `dispatch_order_id` (created in `/confirm` when `ORDER_CONFIRMED` event received, order.id seller-generated, dispatch_order_id Order Service-generated)
- Order record stores `order.id` (ONDC, network-facing) alongside `dispatch_order_id` (internal execution identifier) on same record
- TTL: 30 days

**Request Context:**
- search_id (Serviceability ID) tracking for `/search` and `/init` correlation
- quote_id tracking for `/init` and `/confirm` correlation
- Temporary entities (dispatch_order_id, quote_id, package details)
- Request context for response reconstruction (search_id, quote_id, dispatch_order_id)
- Callback context for ONDC requests (bap_uri, transaction_id, message_id, callback URL)
- TTL: 1 hour

**Idempotency Keys:**
- Track request hashes for replay protection (ONDC `transaction_id` + `message_id` hash)
- TTL: 24 hours

**Issue Storage:**
- Issue storage: `ondc:issue:{issue_id}` (TTL: 30 days)
- Zendesk ticket reference: `ondc:zendesk_ticket:{zendesk_ticket_id}` → `issue_id`
- Financial resolution data: `ondc:financial:{issue_id}`
- TTL: 30 days

**Client Configuration Cache:**
- `client_config:{client_id}` (TTL: 15 minutes)
- `client:{client_id}` (TTL: 5 minutes)

### 12.2 Persistent Storage (Postgres-E)

**Audit Logs:**
- Request/response pairs in `audit.request_response_logs`
- Order references in order reference table
- Idempotency keys for audit
- Request hashes for non-repudiation
- Webhook delivery logs
- **Retention**: 7 years minimum

**Client Registry:**
- `client_registry` table (synced from Admin Service via events)
- Client configuration snapshots in `client_registry.client_configs` table
- **Retention**: Permanent (synced from Admin Service)

---

## 13. Out of Scope (v0)

The following are explicitly out of scope for v0:

- **Business Logic**: Pricing, capacity management, routing decisions (Quote Service, Location Service, DroneAI own)
- **Order Lifecycle**: Order state management, fulfillment orchestration (Order Service owns)
- **Grievance Resolution**: Issue resolution, customer support (External Helpdesk owns)
- **Client Configuration Management**: Client onboarding, configuration updates (Admin Service owns)
- **Payment Processing**: Payment gateway integration (Payment Service owns)
- **Event Stream Management**: Event stream infrastructure and routing (infrastructure layer owns)

---

## 14. Implementation Notes

### 14.1 Common Request Processing Contract

**All ONDC APIs follow the same processing pattern (unless explicitly overridden):**

1. **Edge Processing**: Generate `traceparent`, validate auth, validate request
2. **Immediate Response**: Return HTTP 200 OK ACK/NACK immediately (< 1 second)
3. **Asynchronous Processing**: Publish events, subscribe to response events, call Order Service
4. **Callback Delivery**: Send callback within TTL period with retry
5. **Audit & Observability**: Persist to Postgres-E, log with correlation IDs

### 14.2 Event-Driven Architecture

**UOIS Gateway is event-driven:**
- Publishes events for async processing (`SEARCH_REQUESTED`, `INIT_REQUESTED`, `CONFIRM_REQUESTED`)
- Subscribes to events for response composition (`QUOTE_COMPUTED`, `QUOTE_CREATED`, `ORDER_CONFIRMED`)
- Uses Redis Streams for reliable event delivery
- Maintains correlation across sync + async hops using lifecycle IDs (`search_id`, `quote_id`) for event correlation and `order.id` (ONDC) for post-confirmation requests

### 14.3 Handler & Orchestration Rules

**MANDATORY:** Follow these strict rules for handler and service layer separation:

**Handler Layer Rules:**
- Each consumed event MUST have its own dedicated handler file
- Event handlers MUST NOT publish events
- Event handlers MUST NOT call gRPC directly
- Handlers validate input, invoke service, return result
- Handlers MUST NOT contain business logic

**Service Layer Rules:**
- Only the service layer may orchestrate event publishing and external calls
- Services may call repositories, call clients, and publish events
- Services MUST NOT route events or parse transport payloads

**Client Layer Rules:**
- gRPC clients MUST NOT emit events
- Clients are pure transport layer (gRPC, HTTP, Redis)

**Purpose:** Maintain clean separation of concerns and prevent circular dependencies.

### 14.4 TTL Handling

**Request TTL** (`PT30S` per ONDC spec): Callback delivery deadline - callback must be sent within this period (as specified in `context.ttl` field)

**Quote TTL** (`PT15M` per ONDC spec): Quote validity period - quote must be valid at time of `/confirm` request (as specified in `quote.ttl` field in `/on_init` response)

**Priority Rule**: If quote TTL (`PT15M`) expires before callback can be sent, callback fails with error code `65004` (Quote Expired); if quote expires after callback has been successfully delivered, order lifecycle continues (quote validation already completed at callback time)

**Independence**: Request TTL (`PT30S`) and Quote TTL (`PT15M`) are independent - callback delivery deadline is separate from quote validity period

### 14.5 Retry Policy

**Exponential backoff**: 1s → 2s → 4s → 8s → 15s (adjusted to fit within PT30S)

**Max retries**: 5 attempts

**After max retries**: Move to Dead Letter Queue (DLQ) for manual replay

**Retryable errors**: Only retry errors with `retryable: true` in error taxonomy

**TTL Constraint**: Total retry duration (sum of all backoff delays) MUST NOT exceed ONDC Request TTL (`PT30S` = 30 seconds). Adjust max retries or backoff intervals to ensure all retries complete within TTL period.

**TTL-Aware Retry Formula**:
- Formula: `sum(CALLBACK_RETRY_BACKOFF_*) <= ONDC_REQUEST_TTL_SECONDS`
- Example calculation: 1s + 2s + 4s + 8s + 15s = 30s (within limit)
- Original 16s backoff would result in 31s total (exceeds limit), so last retry adjusted to 15s

---

## 15. References

### 15.1 Canonical Documents

- **Functional Requirements**: `UOISGateway_FR.md`
- **Integration Boundary**: `docs/production-docs/INTEGRATION_BOUNDARY.md`
- **Repository Setup Guide**: `UOIS_REPOSITORY_SETUP_GUIDE.md`
- **Development Rules**: `.cursorrules`
- **Redis Streams Architecture Alignment**: `docs/analysis/REDIS_STREAMS_ARCHITECTURE_ALIGNMENT.md` - Detailed analysis of Redis Streams consumer group patterns and alignment with implementation

### 15.2 Event Contracts

- **QUOTE_COMPUTED**: `contracts/events/consumed/quote/quote_computed.json`
- **Order Service Events**: `contracts/events/produced/confirmation/` (ORDER_CONFIRMED, ORDER_CONFIRM_FAILED)
- **UOIS Published Events**: `contracts/events/produced/` (SEARCH_REQUESTED, INIT_REQUESTED, CONFIRM_REQUESTED)

### 15.3 Related Services

- **Location Service Contracts**: `/docs/04_DispatchContracts/06_location_service/`
- **Order Service Contracts**: `Order-Service-Dispatch/contracts/`
- **Quote Service Contracts**: `Quote-Service-Dispatch/contracts/`

---

**End of Implementation Plan**

