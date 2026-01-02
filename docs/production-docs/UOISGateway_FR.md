---
sideFbar_position: 11
---

## Functional Requirements – UOIS Gateway

## 0. Context & Scope

**Business Context**: Dispatch/UOIS uses **P2P (Point-to-Point) delivery only**, not hyperlocal delivery. All fulfillment operations are configured for P2P delivery type.

The **Universal Order Interface Service (UOIS) Gateway** is a middleware service that acts as a protocol translation and routing layer between:

- **External Clients**: ONDC network participants
- **Internal Services**: Order Service, Location Service, Quote Service, Admin Service, DroneAI, Notification Service

**Core Responsibilities**:
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
- Issue & Grievance Management (IGM) for ONDC compliance (issue creation, status tracking, Zendesk Helpdesk integration, bidirectional sync)

**Service Boundaries**:
- **UOIS Gateway owns**:
  - Protocol translation and validation
  - Client authentication and rate limiting
  - Request/response signing (ONDC)
  - Event publishing and subscription:
    - Publish `SEARCH_REQUESTED` to stream `stream.location.search`
    - Publish `INIT_REQUESTED` to stream `stream.uois.init_requested`
    - Publish `CONFIRM_REQUESTED` to stream `stream.uois.confirm_requested`
  - Event consumption for response composition:
    - Consume `QUOTE_COMPUTED` from stream `quote:computed`
    - Consume `QUOTE_CREATED` from stream `stream.uois.quote_created`
    - Consume `QUOTE_INVALIDATED` from stream `stream.uois.quote_invalidated`
    - Consume `ORDER_CONFIRMED` from stream `stream.uois.order_confirmed`
    - Consume `ORDER_CONFIRM_FAILED` from stream `stream.uois.order_confirm_failed`
  - search_id (Serviceability ID) generation for `/search` requests (UOIS Gateway generates search_id, never derives from request payload)
  - **Trace ID generation** (W3C traceparent) at edge for distributed tracing
  - Idempotency enforcement
  - Callback relay via event consumption
  - Issue & Grievance Management (IGM) API endpoints (`/issue`, `/issue_status`, `/on_issue`, `/on_issue_status`)
  - IGM state management and Zendesk Helpdesk integration (bidirectional sync)
  - Issue storage and references (Redis)
  - Client configuration processing and caching (Redis cache, Postgres-E client registry)
  - Temporary caching for performance
  - Audit logging to Postgres-E (audit schema)

- **UOIS Gateway does NOT own**:
  - Business logic (pricing, capacity, routing) → Quote Service, Location Service, DroneAI
  - Order lifecycle management → Order Service
  - Issue resolution and ticket content → External Helpdesk (Zendesk) - UOIS Gateway maintains sync and references only
  - Client configuration source of truth → Admin Service (UOIS Gateway fetches and processes config, but Admin Service owns the authoritative configuration)

---

## 1. Common Request Processing Contract

**Purpose**: Define the standard processing pattern that applies to ALL APIs unless explicitly overridden.

**Authoritative Rule**: Unless explicitly overridden, all sections MUST NOT restate ACK, TTL, retry, callback, audit, or tracing behavior defined here.

**Standard Processing Flow** (Applies to: `/search`, `/init`, `/confirm`, `/status`, `/track`, `/cancel`, `/update`, `/rto`):

1. **Edge Processing**:
   - Generate W3C `traceparent` header if not present in incoming request
   - Start root span using OpenTelemetry
   - Extract and validate client credentials (authentication/authorization)
   - Validate request structure and required fields

2. **Immediate Response**:
   - Return synchronous HTTP 200 OK ACK/NACK immediately (< 1 second)
   - Does NOT block on Order Service calls or event processing
   - Include error details in NACK responses

3. **Asynchronous Processing**:
   - Publish events to event stream (when applicable)
   - Subscribe to response events (when applicable)
   - Call Order Service gRPC methods (when applicable)
   - Process within request TTL period

4. **Callback Delivery**:
   - Send asynchronous callback to client within TTL period
   - Use exponential backoff retry policy for failed deliveries
   - Dead Letter Queue for persistent failures after max retries
   - Construct callback URL: `{bap_uri}/on_{action}`

5. **Audit & Observability**:
   - Persist request/response to Postgres-E (`audit.request_response_logs`)
   - Include trace_id, correlation IDs, and processing metadata
   - Log delivery attempts and status
   - Maintain correlation across sync + async hops

**TTL Handling** (ONDC Contract):
- **Request TTL**: Buyer NP sends `ttl` in request context (typically `PT30S` for most APIs, `PT15M` for quote validity)
- **Response within TTL**: Send callback asynchronously within TTL period
- **Quote TTL**: Quote validity period (`PT15M`) - validate before `/confirm`

**Timeout & Retry Defaults**:

| Flow | Event Subscription | Default Timeout | Retry Policy | Max Retries |
|------|-------------------|-----------------|--------------|-------------|
| `/search` → `QUOTE_COMPUTED` | `quote:computed` | 30 seconds | N/A (single event) | N/A |
| `/init` → `QUOTE_CREATED` | `stream.uois.quote_created` | 30 seconds | N/A (single event) | N/A |
| `/confirm` → `ORDER_CONFIRMED` | `stream.uois.order_confirmed` | 60-120 seconds | N/A (single event) | N/A |
| Callback delivery retries | HTTP POST to `{bap_uri}/on_*` | Per attempt: 5 seconds | Exponential backoff: 1s → 2s → 4s → 8s → 15s (adjusted to fit within PT30S) | 5 attempts |
| Dead Letter Queue | After max retries | N/A | Manual replay only | N/A |

**Critical Constraint - Retry Timeout vs ONDC TTL**:
- **Internal retries MUST NOT exceed ONDC Request TTL** (`PT30S` = 30 seconds per ONDC spec)
- Total retry duration (including all backoff delays) must complete within the request TTL period
- Example: For `PT30S` TTL, exponential backoff is 1s → 2s → 4s → 8s → 15s (total = 30s, within limit)
- **Enforcement**: Calculate total retry duration before initiating retries; if it exceeds request TTL, reduce max retries or backoff intervals to ensure completion within TTL
- **Callback delivery**: All callback retries must complete within the request TTL period specified in `context.ttl` (typically `PT30S`)
- **TTL-Aware Defaults for ONDC Flows**:
  - **ONDC Request TTL**: `PT30S` (30 seconds) - callback delivery deadline
  - **ONDC Quote TTL**: `PT15M` (15 minutes) - quote validity period
  - **Formula**: Total retry duration = `sum(CALLBACK_RETRY_BACKOFF_*) <= ONDC_REQUEST_TTL_SECONDS`
  - **Example calculation**: 1s + 2s + 4s + 8s + 15s = 30s (within PT30S limit)

**Quote TTL vs Callback TTL Priority**:
- **Request TTL** (`PT30S` per ONDC spec): Callback delivery deadline - callback must be sent within this period (as specified in `context.ttl` field)
- **Quote TTL** (`PT15M` per ONDC spec): Quote validity period - quote must be valid at time of `/confirm` request (as specified in `quote.ttl` field in `/on_init` response)
- **Priority Rule**: If quote TTL (`PT15M`) expires before callback can be sent, callback fails with error code `65004` (Quote Expired); if quote expires after callback has been successfully delivered, order lifecycle continues (quote validation already completed at callback time)
- **Independence**: Request TTL (`PT30S`) and Quote TTL (`PT15M`) are independent - callback delivery deadline is separate from quote validity period

**Error Handling**:
- Synchronous errors: Return NACK immediately with error code
- Asynchronous errors: Include in callback payload
- Service failures: Log timeout events for monitoring/alerting

**Idempotency & Correlation**:

| Operation | Idempotency Key | Correlation ID | Storage Key Pattern |
|-----------|----------------|----------------|---------------------|
| `/search` | `search_id` (generated by UOIS) | `search_id` | `search:{search_id}` |
| `/init` | `search_id` (from `/search`) | `search_id` → `quote_id` | `search:{search_id}` → `quote:{quote_id}` |
| `/confirm` | `quote_id` (from `/init`) | `quote_id` | `quote:{quote_id}` |
| `/status` | `order.id` (ONDC, from `message.order_id`) | `order.id` (ONDC) | `order:{order.id}` |
| `/track` | `order.id` (ONDC, from `message.order_id`) | `order.id` (ONDC) | `order:{order.id}` |
| `/cancel` | `order.id` (ONDC, from `message.order_id`) | `order.id` (ONDC) | `order:{order.id}` |
| `/update` | `(order.id + update_type)` (ONDC, from `message.order_id`) | `order.id` (ONDC) | `order:{order.id}:update:{update_type}` |
| `/rto` | `order.id` (ONDC, from `message.order_id`) | `order.id` (ONDC) | `order:{order.id}` |

**Idempotency Rules**:
- Track request hashes for replay protection (ONDC `transaction_id` + `message_id` hash)
- Use correlation IDs: `search_id` (search/init), `quote_id` (init/confirm), `dispatch_order_id` (post-confirmation)
- Support idempotent request replay - return existing response if duplicate detected
- Store idempotency keys in Redis with TTL (24 hours) and Postgres-E for audit (7 years)

**Exceptions**: Individual flows may override specific steps with documented deltas.

---

## 2. Event-Driven Request Processing

### 1.1 Event Publishing Pattern

**Purpose**: Process client requests asynchronously via event-driven architecture.

**Functional Requirements**:
- Unless stated otherwise, all flows inherit ID propagation via echo contract, ACK/callback behavior, TTL handling, retry logic, and audit logging from the Common Request Processing Contract (Section 1).
- **`/search` Flow** — **Delta from Common Request Processing Contract**:
  - **Unique Processing**: Generate search_id (Serviceability ID) for request correlation (UOIS Gateway generates search_id, never derives from request payload)
  - **Event Publishing**: Publish `SEARCH_REQUESTED` event to stream `stream.location.search` with search_id, pickup/drop coordinates, traceparent
  - **Event Consumption**: Consume `QUOTE_COMPUTED` events from stream `quote:computed` using consumer group `uois-gateway-consumers`, filter by `search_id` correlation
  - **Response Composition**: Quote Service passes through serviceability fields from `SERVICEABILITY_FOUND` to `QUOTE_COMPUTED`, so UOIS only needs to subscribe to `QUOTE_COMPUTED` event which contains all needed fields: `serviceable`, `distance_origin_to_destination`, `eta_origin`, `eta_destination`, `price`, `ttl`
  - **Field Transformation**: Convert `eta_*` fields to `tat_*` (ONDC-compliant: eta_origin → tat_to_pickup, eta_destination → tat_to_drop)
  - **Timeout Handling**: Return "serviceable: false" response if QUOTE_COMPUTED not received within TTL

- **`/init` Flow** — **Delta from Common Request Processing Contract**:
  - **Pre-Processing**: Call Order Service (gRPC) to validate search_id TTL and quote validity (return immediate NACK if validation fails)
  - **Unique Processing**: Extract `provider.id` from `message.order.provider.id` (echoed from `/on_search`), validate it matches configured stable provider identifier, use internal `search_id` (from order record) for event correlation
  - **Event Publishing**: Publish `INIT_REQUESTED` event to stream `stream.uois.init_requested` with search_id, pickup/drop coordinates + addresses, package info, traceparent
  - **Event Consumption**: Consume `QUOTE_CREATED` events from stream `stream.uois.quote_created` or `QUOTE_INVALIDATED` events from stream `stream.uois.quote_invalidated` using consumer group `uois-gateway-consumers`, filter by `search_id` correlation
  - **Response Composition**: Extract quote_id, price, eta fields, ttl (PT15M quote validity period) from `QUOTE_CREATED` event
  - **Reference Storage**: Store search_id → quote_id reference in Redis/Postgres-E for correlation lookup

- **`/confirm` Flow** — **Delta from Common Request Processing Contract**:
  - **Input Processing**: Extract `quote_id` from `message.order.quote.id` (echoed from `/on_init`)
  - **Event Publishing**: Publish `CONFIRM_REQUESTED` event to stream `stream.uois.confirm_requested` with quote_id, client_id, payment_info, traceparent (triggers order creation + rider assignment)
  - **Order ID Generation**: When `ORDER_CONFIRMED` event received, generate ONDC `order.id` (seller-generated, network-facing)
  - **Event Consumption**: Consume `ORDER_CONFIRMED` events from stream `stream.uois.order_confirmed` or `ORDER_CONFIRM_FAILED` events from stream `stream.uois.order_confirm_failed` using consumer group `uois-gateway-consumers`, filter by `quote_id` correlation
  - **Reference Storage**: On `ORDER_CONFIRMED`, generate ONDC `order.id` (seller-generated) and store quote_id → `order.id` (ONDC) → dispatch_order_id reference for routing lookup
  - **Response Composition**: Include rider assignment status, `order.id` (ONDC, seller-generated), rider_id if assigned from `ORDER_CONFIRMED` event
  - **Order Lifecycle**: Does NOT block on rider assignment (async callback when assignment completes)

- **Post-confirmation Flows Inheritance**: All post-confirmation flows (`/status`, `/track`, `/cancel`, `/update`, `/rto`) inherit request validation, ID propagation via echo contract, ACK/callback semantics, TTL handling, retries, and audit logging from Section 1 and Section 1.4.

- **`/status` Flow** — **Delta from Common Request Processing Contract**:
  - **Input Processing**: Extract ONDC `order.id` from `message.order_id` (echoed from `/on_confirm`), look up order record using `order.id` (ONDC), retrieve `dispatch_order_id` from order record (per Section 1.4)
  - **Service Call**: Order Service gRPC `GetOrder` to fetch current status
  - **Response Composition**: Transform order state, rider info, timeline, fulfillment states, proof of pickup/delivery to ONDC format
  - **Caching**: Optional short TTL cache (15-30 seconds) for status responses

- **`/track` Flow** — **Delta from Common Request Processing Contract**:
  - **Input Processing**: Extract ONDC `order.id` from `message.order_id` (echoed from `/on_confirm`), look up order record using `order.id` (ONDC), retrieve `dispatch_order_id` from order record (per Section 1.4)
  - **Service Call**: Order Service gRPC `GetOrderTracking` (aggregates location data internally from Location Service)
  - **Response Composition**: Transform live location, ETA, timeline to ONDC format (GPS coordinates or tracking URL)
  - **ONDC Note**: As of July 2023, `callback_url` removed from `/track`; use polling only
  - **Caching**: Very short TTL cache (5-10 seconds) for tracking responses

- **`/cancel` Flow** — **Delta from Common Request Processing Contract**:
  - **Input Processing**: Extract ONDC `order.id` from `message.order_id` (echoed from `/on_confirm`), look up order record using `order.id` (ONDC), retrieve `dispatch_order_id` from order record (per Section 1.4)
  - **Validation**: Validate cancellation eligibility (Order Service enforces business rules)
  - **Service Call**: Order Service gRPC `CancelOrder` with `dispatch_order_id`
  - **Response Composition**: Include cancellation details and updated quote (if applicable)

- **`/update` Flow** — **Delta from Common Request Processing Contract**:
  - **Input Processing**: Extract ONDC `order.id` from `message.order_id` (echoed from `/on_confirm`), look up order record using `order.id` (ONDC), retrieve `dispatch_order_id` from order record (per Section 1.4)
  - **Validation**: Validate update eligibility (Order Service enforces business rules)
  - **Service Call**: Order Service gRPC `UpdateOrder` with `dispatch_order_id` (handles RTS, authorization, weight differential)
  - **Response Composition**: Include updated order details and updated quote (if weight/dimensions changed)

- **`/rto` Flow** (Return to Origin) — **Delta from Common Request Processing Contract**:
  - **Input Processing**: Extract ONDC `order.id` from `message.order_id` (echoed from `/on_confirm`), look up order record using `order.id` (ONDC), retrieve `dispatch_order_id` from order record (per Section 1.4)
  - **Validation**: Validate RTO eligibility (Order Service enforces business rules - must be in eligible states)
  - **Service Call**: Order Service gRPC `InitiateRTO` to transition order to `RTO_INITIATED` state
  - **Response Composition**: Send via `/on_update` callback (RTO handled via `/update` flow in ONDC)

**Event Correlation**:
- Use search_id for `/search` and `/init` correlation
- Use quote_id for `/init` and `/confirm` correlation
- **Internal event correlation**: Use dispatch_order_id for internal service calls and event processing (post-confirmation)
- **External protocol correlation**: Use order.id (ONDC) for ONDC callbacks and client-facing responses
- For ONDC callbacks: Use `transaction_id` and `message_id` from request context to correlate callbacks with original requests

**TTL Handling**: Follows ONDC contract as defined in Common Request Processing Contract (Section 1).

**Stale Request Detection**: Reject requests with timestamp earlier than previously processed request (same `transaction_id` + `message_id`) and return NACK with error code `65003`.

**Quote TTL**: Quote provided in `/on_init` has `ttl: "PT15M"` (15 minutes) - validate quote validity before processing `/confirm` requests.

### 1.2 Response Composition

**Purpose**: Compose client responses from multiple events.

**Functional Requirements**:
- **`/search` Response Composition**:
  - Consume events from stream `quote:computed` using consumer group `uois-gateway-consumers`, filter events by `search_id` correlation
  - **Note**: Quote Service passes through serviceability fields (`serviceable`, `distance_origin_to_destination`, `eta_origin`, `eta_destination`) from `SERVICEABILITY_FOUND` to `QUOTE_COMPUTED`, so UOIS only needs to subscribe to `QUOTE_COMPUTED` event which contains all needed fields
  - Extract fields from `QUOTE_COMPUTED` event:
    - `serviceable` (pass-through from SERVICEABILITY_FOUND)
    - `distance_origin_to_destination` (pass-through from SERVICEABILITY_FOUND)
    - `eta_origin` (pass-through from SERVICEABILITY_FOUND)
    - `eta_destination` (pass-through from SERVICEABILITY_FOUND)
    - `price` (computed by Quote Service)
    - `ttl` (expires_in)
  - Transform internal field names to client-facing ONDC field names:
    - `eta_origin` → `tat_to_pickup` (Turn Around Time to pickup)
    - `eta_destination` → `tat_to_drop` (Turn Around Time to drop)
  - Compose response with: `serviceable`, `search_id`, `distance_origin_to_destination`, `tat_to_pickup`, `tat_to_drop`, `price`, `expires_in`
  - Compose and send `/on_search` callback to client callback URL (`{bap_uri}/on_search`)
  - **Note**: Location Service publishes `SERVICEABILITY_FOUND` event to stream `location:serviceability:found`; Quote Service consumes it and publishes `QUOTE_COMPUTED` event to stream `quote:computed` with all fields included; UOIS Gateway transforms `eta_*` to `tat_*` for ONDC-compliant client response
  - **Timeout handling**: If `QUOTE_COMPUTED` event not received within request TTL period (typically 30 seconds), return timeout error response

- **`/init` Response Composition**:
  - Consume events from stream `stream.uois.quote_created` using consumer group `uois-gateway-consumers`, filter events by `search_id` correlation for `QUOTE_CREATED` events
  - Consume events from stream `stream.uois.quote_invalidated` using consumer group `uois-gateway-consumers`, filter events by `search_id` correlation for `QUOTE_INVALIDATED` events
  - Receive `QUOTE_CREATED` or `QUOTE_INVALIDATED` event (correlated by `search_id`)
  - Extract fields from event:
    - From `QUOTE_CREATED`: `quote_id`, `search_id`, `price` (formatted string, e.g., "₹60"), `distance_origin_to_destination`, `eta_origin`, `eta_destination`, `expires_in` (ttl: PT15M), `timestamp`, `traceparent`, `trace_id`
    - From `QUOTE_INVALIDATED`: `quote_id`, `search_id`, `error`, `message`, `requires_research`, `timestamp`, `traceparent`, `trace_id`
  - Compose response:
    - Success: `quote_id`, `price`, `distance_origin_to_destination`, `eta_origin`, `eta_destination`, `expires_in` (PT15M)
    - Failure: `error`, `message`, `requires_research`
  - Compose and send `/on_init` callback to client callback URL (`{bap_uri}/on_init`)
  - **Note**: Order Service publishes `QUOTE_CREATED` event to stream `stream.uois.quote_created` or `QUOTE_INVALIDATED` event to stream `stream.uois.quote_invalidated` after TTL validation and quote creation; Location Service and Quote Service communicate via events during revalidation (not directly to UOIS)
  - **Timeout handling**: If `QUOTE_CREATED` or `QUOTE_INVALIDATED` event not received within request TTL period (typically 30 seconds), return timeout error response

- **`/on_confirm` Callback Composition**:
  - Consume events from stream `stream.uois.order_confirmed` using consumer group `uois-gateway-consumers`, filter events by `quote_id` correlation for `ORDER_CONFIRMED` events
  - Consume events from stream `stream.uois.order_confirm_failed` using consumer group `uois-gateway-consumers`, filter events by `quote_id` correlation for `ORDER_CONFIRM_FAILED` events
  - Receive `ORDER_CONFIRMED` or `ORDER_CONFIRM_FAILED` event (correlated by `quote_id`)
  - Extract fields from event:
    - From `ORDER_CONFIRMED`: `event_id`, `dispatch_order_id` (internal-only, used for internal mapping), `quote_id`, `rider_id`, `timestamp`, `traceparent`, `trace_id`
    - From `ORDER_CONFIRM_FAILED`: `event_id`, `dispatch_order_id` (internal-only, used for internal mapping), `quote_id`, `reason`, `timestamp`, `traceparent`, `trace_id`
  - Generate ONDC `order.id` (seller-generated, network-facing) for callback response
  - Compose callback response:
    - Success: `order.id` (ONDC, seller-generated), `rider_assigned: true`, `rider_id` (optional)
    - Failure: `order.id` (ONDC, seller-generated), `rider_assigned: false`, `message`, `requires_research`
    - **Critical**: `dispatch_order_id` is extracted from event for internal mapping only; it is NEVER included in ONDC callback payload
  - Send callback to client callback URL (`{bap_uri}/on_confirm`)
  - **Note**: Order Service publishes `ORDER_CONFIRMED` event to stream `stream.uois.order_confirmed` or `ORDER_CONFIRM_FAILED` event to stream `stream.uois.order_confirm_failed` after rider assignment (via DroneAI) or assignment failure
  - **Note**: `/confirm` returns immediate HTTP 200 OK ACK; `/on_confirm` callback is sent asynchronously when assignment completes

- **`/status` Response Composition**:
  - Fetch order data from Order Service (gRPC: GetOrder)
  - Transform order state, rider info, timeline, fulfillment states, proof of pickup/delivery to ONDC/Beckn format
  - Compose and send `/on_status` callback to client callback URL (`{bap_uri}/on_status`)
  - **Note**: Order Service aggregates all order information including location data (if needed) from internal services

- **`/track` Response Composition**:
  - Fetch order tracking data from Order Service (gRPC: GetOrderTracking)
  - Order Service provides complete tracking information including live location, ETA, and timeline
  - Transform to ONDC/Beckn format
  - Compose and send `/on_track` callback to client callback URL (`{bap_uri}/on_track`) with tracking information (GPS coordinates or tracking URL)
  - **Note**: UOIS Gateway does not directly call Location Service; Order Service aggregates tracking data from Location Service internally

- **`/cancel` Response Composition**:
  - Receive cancellation result from Order Service (gRPC: CancelOrder)
  - Transform cancellation confirmation and updated quote to ONDC/Beckn format
  - Compose and send `/on_cancel` callback to client callback URL (`{bap_uri}/on_cancel`)

- **`/update` Response Composition**:
  - Receive update result from Order Service (gRPC: UpdateOrder)
  - Transform updated order details and updated quote (if weight/dimensions changed) to ONDC/Beckn format
  - Compose and send `/on_update` callback to client callback URL (`{bap_uri}/on_update`)

- **`/rto` Response Composition**:
  - Receive RTO initiation result from Order Service (gRPC: InitiateRTO)
  - Transform RTO confirmation to ONDC/Beckn format
  - Compose and send `/on_update` callback to client callback URL (`{bap_uri}/on_update`) (RTO is handled via `/update` flow in ONDC)

---

## 1.3 Quote Service to UOIS Payload Integration

This section documents the payload structure for events published by Quote Service that are consumed by UOIS Gateway.

### 1.3.1 QUOTE_COMPUTED Event (Quote Service → UOIS)

**Event Type:** `QUOTE_COMPUTED`  
**Stream:** `quote:computed`  
**Flow:** `/search` flow  
**Published By:** Quote Service  
**Consumed By:** UOIS Gateway

**Payload:**

```json
{
  "event_type": "QUOTE_COMPUTED",
  "search_id": "550e8400-e29b-41d4-a716-446655440000",
  "serviceable": true,
  "price": {
    "value": 58.00,
    "currency": "INR"
  },
  "ttl": "PT10M",
  "ttl_seconds": 600,
  "eta_origin": "2024-01-15T14:30:00Z",
  "eta_destination": "2024-01-15T14:45:00Z",
  "distance_origin_to_destination": 3.2,
  "timestamp": "2024-01-15T14:22:30Z",
  "traceparent": "00-4bf92f3577b34da6a3ce929d0e0e4736-8f2a1b2c3d4e5f6a-01",
  "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736",
  "metadata": {
    "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736",
    "rate_card_id": "rc_001",
    "zone_id": "zone_001"
  }
}
```

**Field Descriptions:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `event_type` | string | Yes | Event identifier: `"QUOTE_COMPUTED"` |
| `search_id` | UUID | Yes | Serviceability ID for event correlation with pending `/search` request (exact same as SEARCH_REQUESTED) |
| `serviceable` | boolean | Yes | **Pass-through from SERVICEABILITY_FOUND** - Location Service is source of truth |
| `price.value` | number | Yes | Total computed price value (base price + pickup fee/surge) - MUST be numeric, NOT formatted string |
| `price.currency` | string | Yes | ISO 4217 currency code (e.g., "INR") |
| `ttl` | string (ISO8601 duration) | Yes | Time-to-live as ISO8601 duration (e.g., "PT10M" = 10 minutes) |
| `ttl_seconds` | integer | Recommended | Time-to-live in seconds (typically 600 = 10 minutes) |
| `eta_origin` | ISO 8601 timestamp | Recommended | **Pass-through from SERVICEABILITY_FOUND** - When rider will reach pickup |
| `eta_destination` | ISO 8601 timestamp | Recommended | **Pass-through from SERVICEABILITY_FOUND** - When package will reach drop |
| `distance_origin_to_destination` | number | Optional | **Pass-through from SERVICEABILITY_FOUND** - Route distance in km |
| `timestamp` | ISO 8601 | Yes | Event publication timestamp in UTC |
| `traceparent` | string (W3C) | Yes | W3C traceparent header (format: `00-<trace-id>-<span-id>-<flags>`). Propagated from UOIS Gateway through Location Service and Quote Service. UOIS extracts to maintain trace continuity. |
| `trace_id` | string | Recommended | Distributed tracing identifier (extracted from traceparent). Used for logging alongside search_id. |
| `metadata` | object | Optional | Additional metadata (trace_id, rate_card_id, zone_id) |

**How UOIS Uses This Event:**

1. **Event Subscription:** UOIS subscribes to event stream `quote:computed` filtering for `QUOTE_COMPUTED` events
2. **Correlation:** Matches `search_id` from event with pending `/search` request context
3. **Data Extraction:** Quote Service passes through serviceability fields from `SERVICEABILITY_FOUND` to `QUOTE_COMPUTED`, so UOIS only needs to extract all fields from `QUOTE_COMPUTED` event (no need to merge with `SERVICEABILITY_FOUND`)
4. **Field Transformation:**
   - `ttl` → `expires_in` (for client response)
   - `eta_origin` → `tat_to_pickup` (pass-through from SERVICEABILITY_FOUND via Quote Service)
   - `eta_destination` → `tat_to_drop` (pass-through from SERVICEABILITY_FOUND via Quote Service)
5. **Response Composition:** Builds final `/on_search` callback payload using all fields from `QUOTE_COMPUTED` event

**Example Response Composition:**

**Input Event:**
- `QUOTE_COMPUTED` from stream `quote:computed`: `{event_type: "QUOTE_COMPUTED", search_id: "550e8400-e29b-41d4-a716-446655440000", serviceable: true, distance_origin_to_destination: 3.2, eta_origin: "2024-01-15T14:30:00Z", eta_destination: "2024-01-15T14:45:00Z", price: {value: 58.00, currency: "INR"}, ttl: "PT10M", ttl_seconds: 600, traceparent: "00-4bf92f3577b34da6a3ce929d0e0e4736-8f2a1b2c3d4e5f6a-01"}`

**Note:** Quote Service passes through all serviceability fields (`serviceable`, `distance_origin_to_destination`, `eta_origin`, `eta_destination`) from `SERVICEABILITY_FOUND` to `QUOTE_COMPUTED`, so UOIS only needs to subscribe to `QUOTE_COMPUTED` event.

**UOIS Composed Response (Consumed by Client):**
```json
{
  "serviceable": true,
  "search_id": "550e8400-e29b-41d4-a716-446655440000",
  "distance_origin_to_destination": 3.2,
  "tat_to_pickup": "2024-01-15T14:30:00Z",
  "tat_to_drop": "2024-01-15T14:45:00Z",
  "price": {
    "value": 58.00,
    "currency": "INR"
  },
  "expires_in": 600
}
```

**Note:** UOIS extracts `traceparent` from events for logging and trace correlation, but does not include it in client-facing responses (privacy/security consideration).

**Error Handling:**

- **Timeout:** If `QUOTE_COMPUTED` not received within request TTL (30 seconds), UOIS returns:
  ```json
  {
    "serviceable": false,
    "search_id": "<UUID>",
    "message": "Service temporarily unavailable. Please try again.",
    "requires_research": true
  }
  ```

- **Missing Fields:** If required fields (`price`, `ttl`) are missing, UOIS logs error and returns timeout response

---

### 1.4 ID Propagation & Echo Contract

**Purpose**: Manage internal processing identifiers through propagation and echo contracts, not mapping or derivation.

**Core Principle**: UOIS Gateway manages internal processing identifiers by generating or receiving them from owning services and propagating them to Buyer NPs inside ONDC-compliant first-class ID fields. Buyer NPs are required to echo these identifiers unchanged in subsequent requests. UOIS Gateway extracts echoed identifiers only from predefined protocol locations and uses them strictly for internal correlation. UOIS Gateway never maps, derives, or infers identifiers across domains.

**Critical ONDC Protocol Constraint**: ONDC guarantees echo ONLY for specific first-class ID fields, NOT for tags or custom fields. Therefore:
- Internal IDs MUST NOT be sent in tags for logging/debugging
- If an ID must be echoed back by Buyer NP, it MUST be placed in an ONDC-defined echoable field
- Tags MUST NOT be used to carry internal identifiers for correlation
- Buyer NP is NOT expected to echo tags and is allowed to drop them

**ID Domain Isolation Law**: UOIS Gateway never maps, translates, derives, infers, or substitutes identifiers across domains. Each identifier is owned by exactly one domain and is treated as an opaque value outside that domain. Multiple identifiers may coexist on the same order record, but none replace or represent another.

**Functional Requirements**:
- **ID Generation & Ownership**:
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
    - **Internal mapping**: `order.id` (ONDC) resolves internally to `dispatch_order_id` (internal-only)

- **ID Propagation (UOIS → Buyer NP)** - **ONDC Canonical Echo Fields**:
  - **`/on_search` → `message.catalog.bpp/providers[].id`**:
    - Seller provides stable `provider.id` (e.g., "P1") - opaque identifier, NOT internal search_id
    - Buyer echoes this in `/init` → `message.order.provider.id`
    - **Note**: `search_id` is internal-only and NEVER sent in ONDC payloads
    - **Correlation**: Internal correlation for `/init` is maintained via internal order records, not via ONDC payloads
  - **`/on_init` → `message.order.quote.id`**:
    - Seller generates `quote_id` (canonical transactional identifier)
    - Seller MUST send `quote_id` ONLY in `message.order.quote.id` (first-class ONDC field)
    - Buyer MUST echo this exact `quote_id` in `/confirm` → `message.order.quote.id`
    - This is the GUARANTEED lookup key for quote validation
    - **Critical**: `quote_id` is the canonical transactional identifier for quote correlation, audit, debugging, and lookup
  - **`/on_confirm` → `message.order.id`**:
    - Seller generates `order.id` (ONDC, seller-generated, network-facing)
    - Seller sends `order.id` (ONDC) in `message.order.id` for Buyer NP reference
    - Buyer MUST echo this `order.id` (ONDC) in all post-order APIs (`/status`, `/track`, `/cancel`, `/update`, `/rto`)
    - **Critical**: `dispatch_order_id` is internal-only and NEVER visible to Buyer NP

- **ID Echo Contract (Buyer NP → UOIS)**:
  - **`provider.id`**: Buyer NP echoes stable provider identifier in `/init` request → `message.order.provider.id`
  - **`quote_id`**: Buyer NP MUST echo in `/confirm` request → `message.order.quote.id` (canonical transactional identifier)
  - **`order.id` (ONDC)**: Buyer NP MUST echo in all post-order APIs (`/status`, `/track`, `/cancel`, `/update`, `/rto`) → `message.order_id` (seller-generated)
  - **Note**: `search_id` is internal-only and never echoed (Buyer NP never sees it)
  - **Note**: `dispatch_order_id` is internal-only and never echoed (Buyer NP never sees it)

- **ID Extraction & Validation**:
  - **For `/init` requests**:
    - Extract `provider.id` from `message.order.provider.id` (echoed from `/on_search`)
    - Validate `provider.id` matches configured stable provider identifier
    - Lookup internal order record using `provider.id` + `transaction_id` (or internal correlation mechanism)
    - Retrieve internal `search_id` from order record (internal-only, not from ONDC payload)
    - Use internal `search_id` to publish `INIT_REQUESTED` event
    - **Note**: `search_id` is internal-only; correlation is maintained via internal order records, not via ONDC payloads
  - **For `/confirm` requests**:
    - Extract `quote_id` from `message.order.quote.id` (echoed from `/on_init` - canonical transactional identifier)
    - **Lookup quote by `quote_id`** (primary correlation key):
      - Validate `quote_id` exists in quote store
      - Validate `quote_id` TTL not expired (PT15M)
      - Validate pricing integrity
    - If missing or invalid → return NACK immediately (error code `65005`)
    - Use `quote_id` to publish `CONFIRM_REQUESTED` event
    - When `ORDER_CONFIRMED` event received:
      - Generate ONDC `order.id` (seller-generated, network-facing)
      - Store order record with: `quote_id`, `order.id` (ONDC), `dispatch_order_id` (internal-only)
      - Send ONDC `order.id` in `/on_confirm` → `message.order.id` (NOT dispatch_order_id)
    - **Critical**: Quote correlation MUST be done using `message.order.quote.id` - this solves the need for correlation, audit, debugging, and lookup
    - **Critical**: `dispatch_order_id` is internal-only and MUST NEVER be sent in ONDC payloads
  - **For post-confirmation requests** (`/status`, `/track`, `/cancel`, `/update`, `/rto`):
    - Extract ONDC `order.id` from `message.order_id` (echoed from `/on_confirm` - seller-generated)
    - Look up the internal order record using `order.id` (ONDC)
    - Retrieve `dispatch_order_id` from the order record
    - If order record not found → return NACK immediately (error code `65006`)
    - Use `dispatch_order_id` ONLY for internal Order Service calls (gRPC methods)
    - **Critical**: `order.id` (ONDC) is the canonical post-confirmation identifier - Buyer echoes seller-generated `order.id`

- **Fail-Fast Behavior**:
  - If expected echoed ID is missing from predefined protocol location → return NACK immediately
  - If ID was not previously issued by UOIS or an owning service → return NACK immediately
  - No fallback resolution, no inference, no guessing
  - No backward compatibility lookups (e.g., no `quote_id` fallback for post-confirmation requests)

- **Order Reference Storage**:
  - Order record stores multiple identifiers on the same record:
    - `order.id` (ONDC) - network-facing, seller-generated
    - `dispatch_order_id` (internal execution identifier)
    - `quote_id` (commercial lock identifier)
    - `search_id` (serviceability identifier)
  - **Purpose**: Multiple identifiers coexist on same order record; no identifier replaces or represents another
  - **Lookup rule**: Lookup is always performed by primary identifier expected by the API. No identifier is inferred, derived, or substituted.
  - **Terminology**: 
    - `order.id` (ONDC): Seller-generated order identifier (network-facing)
      - **Source**: Generated by Seller NP (UOIS Gateway / Order Service) during `/confirm` flow
      - **Ownership**: Seller domain, network-facing
      - **Flow requirement**: Sent in `/on_confirm` → `message.order.id`, Buyer echoes in all post-order APIs
      - **Internal use-cases**:
        - Callback composition: `/on_status`, `/on_track`, `/on_cancel` MUST return `order.id` (ONDC)
        - Audit & dispute resolution: Support teams reference ONDC order.id
        - IGM/grievance flows: Issues raised against ONDC order.id
        - Customer support tooling: Search by ONDC order.id
        - Reconciliation: Matching ONDC order.id vs internal orders
        - Rider App / Ops UI: ONDC order.id shown alongside dispatch ID
      - **Echo contract**: Buyer echoes seller-generated `order.id` in all post-order APIs
      - **Internal mapping**: `order.id` (ONDC) → `dispatch_order_id` (internal-only)
    - `dispatch_order_id`: Internal Order Service identifier (human-readable format, e.g., "ABC0000001")
      - **Source**: Generated by Order Service when order is created
      - **Ownership**: Order Service domain
      - **Usage**: Used for all internal service calls to Order Service
      - **Critical**: MUST NEVER be sent in ONDC payloads - internal-only
  - Store in Redis (temporary, TTL: 30 days) and Postgres-E (audit, 7-year retention)
  - Update reference when `ORDER_CONFIRMED` event received: store `quote_id` → `order.id` (ONDC) → `dispatch_order_id` (internal) mapping

- **Service Calls**:
  - Call Order Service (gRPC: GetOrder, GetOrderTracking, CancelOrder, UpdateOrder, InitiateRTO)
  - Order Service aggregates all order information including location/tracking data from internal services
  - Handle gRPC timeouts and errors gracefully
  - Transform service responses to ONDC/Beckn format
  - **Note**: UOIS Gateway does not directly call Location Service; all communication with Location Service is via event stream (for `/search` and `/init`) or through Order Service (for `/track` and `/status`)

- **Response Caching** (Optional):
  - Cache status responses (short TTL, 15-30 seconds)
  - Cache tracking responses (very short TTL, 5-10 seconds)
  - Invalidate cache on state transitions (via event consumption)

**ONDC Endpoint Pattern**:
- All endpoints (`/search`, `/init`, `/confirm`, `/status`, `/track`, `/cancel`, `/update`) use POST method
- Return immediate HTTP 200 OK ACK/NACK synchronously
- Send callback response asynchronously to buyer NP callback URL (`{bap_uri}/on_{action}`)
- Extract `order_id` from `message.order_id` (or `message.order.quote.id` for `/confirm`) in request body
- Look up `dispatch_order_id` from order reference table before processing

**ID Lifecycle (Clear Flow)**:
```
/search
  → internal search_id (internal-only, never sent in ONDC payloads)
  → store order record with search_id
  → correlate via transaction_id + message_id (internal mechanism)

/on_search
  → provider.id (stable identifier, e.g., "P1") - opaque, buyer-echoed
  → NO internal IDs in ONDC payload
  → Internal correlation: order record keyed by transaction_id + message_id

/init
  → extract provider.id from message.order.provider.id (echoed from /on_search)
  → validate provider.id matches configured stable identifier
  → lookup internal order record using transaction_id + message_id (internal correlation)
  → retrieve internal search_id from order record (NOT from ONDC payload)
  → use internal search_id for event correlation
  → generate quote_id (Order Service)

/on_init
  → quote_id (canonical transactional identifier)
  → send ONLY in message.order.quote.id (first-class ONDC field)
  → include TTL (PT15M)
  → persist quote keyed by quote_id (primary lookup key)

/confirm
  → extract quote_id from message.order.quote.id (echoed from /on_init)
  → lookup quote by quote_id (primary correlation key)
  → validate: existence, TTL not expired, pricing integrity
  → use quote_id for order creation

/on_confirm
  → order.id (ONDC, seller-generated, network-facing)
  → send in message.order.id for Buyer NP reference
  → Buyer MUST echo order.id (ONDC) in all post-order APIs
  → dispatch_order_id (internal-only, NEVER visible to Buyer NP)
```

**Implementation Rules (DO NOT VIOLATE)**:
- Do NOT introduce new fields into ONDC payloads
- Do NOT rely on tags for correlation
- Treat all ONDC IDs as opaque on the buyer side
- Follow ONDC Logistics v1.2.0 strictly
- Internal IDs (`search_id`, `dispatch_order_id`) NEVER leak into ONDC payloads
- `quote_id` is the canonical transactional identifier - sent ONLY in `message.order.quote.id`
- `order.id` (ONDC) is seller-generated and sent in `/on_confirm` → `message.order.id`
- `dispatch_order_id` is internal-only and MUST NEVER be sent in ONDC payloads
- Buyer echoes seller-generated `order.id` in all post-order APIs

**Critical Rules**:
- Internal IDs (`search_id`, `dispatch_order_id`) NEVER leak into ONDC payloads
- `quote_id` is the canonical transactional identifier - sent ONLY in `message.order.quote.id`
- No tag-based correlation - tags MUST NOT be used to carry internal identifiers
- Buyer NP is NOT expected to echo tags and is allowed to drop them

**Error Handling**:
- Return 404 if order not found
- Return 400 if cancellation/RTO not allowed (invalid state)
- Return 500 if Order Service unavailable
- Include actionable error messages

---

## 2. Protocol & Channel Integration

### 2.1 ONDC Gateway

**Purpose**: Expose ONDC-compliant APIs for network participants.

**Protocol Compliance**:
- **API Methods**: POST (all endpoints: search, init, confirm, status, track, cancel, update)
- **Processing Model**: All APIs follow Common Request Processing Contract (asynchronous ACK + callback pattern)
- **ONDC Versioning**: Support ONDC protocol versioning and backward compatibility
- **Network Registry**: Maintain ONDC network registry integration (subscribe to network events, handle verification callbacks, fetch public keys for signature validation)

**Endpoint Contracts**:
- **Pre-order APIs**: `search`, `init`, `confirm` (async via event stream)
- **Post-order APIs**: `status`, `track`, `cancel`, `update` (async via event stream or direct gRPC)

**ONDC Callback Requirements** (all 7 callbacks required for Logistics Seller NP):
- `/on_search`: Catalog with fulfillment options, pricing, and terms
- `/on_init`: Quote, cancellation terms, and transaction-level contract terms
- `/on_confirm`: Order acceptance/rejection, fulfillment slots, agent details, AWB number
- `/on_status`: Current order status, fulfillment states, proof of pickup/delivery
- `/on_cancel`: Cancellation details and updated quote
- `/on_update`: Updated order details and updated quote (if weight/dimensions changed)
- `/on_track`: Tracking information (GPS coordinates or tracking URL) - Note: As of July 2023, `callback_url` removed; use polling only

**Request Processing**:
- Extract `order_id` from `message.order_id` (or `message.order.quote.id` for `/confirm`)
- Validate stale requests (timestamp validation with `transaction_id` + `message_id`) - return NACK with error code `65003`
- Transform ONDC payloads to internal requests and back to ONDC responses

**Error Schema**: Include `error.type` and `error.code` in all error responses (synchronous or asynchronous)

---

## 3. Security & Non-Repudiation

### 3.1 ONDC Request/Response Signing

**Purpose**: Ensure non-repudiation and authenticity for ONDC transactions per ONDC API Contract v1.2.0.

**ONDC Authentication Requirements** (per ONDC API Contract Section 2 & 3):

**Key Pair Generation**:
- Use ed25519 for signing and X25519 for encryption
- Generate key pairs using standard libraries (e.g., libsodium)
- Update base64 encoded public keys in ONDC network registry
- Reference implementation: [ONDC Signing Utilities](https://github.com/ONDC-Official/reference-implementations/tree/main/utilities/signing_and_verification)

**Request Signature Verification** (Incoming ONDC Requests):
1. **Authorization Header Parsing**:
   - Parse `Authorization` header as key-value pairs (comma-separated)
   - **Mandatory fields**: `keyId`, `signature`
   - **Optional fields**: `created`, `expires` (parsed but not used in signature verification)
   - Reject empty or whitespace-only headers immediately (error code `65002`)
   - Validate required fields (`keyId`, `signature`) are present and non-empty

2. **KeyId Validation**:
   - Parse `keyId` format: `{subscriber_id}|{ukId}|{algorithm}`
   - **Strict validation**: Reject requests where `algorithm != "ed25519"` (error code `65002`)
   - Extract `subscriber_id`, `ukId` (unique_key_id), and `algorithm` from keyId
   - Validate keyId format (must contain exactly 3 parts separated by `|`)

3. **Registry Lookup**:
   - Use ONDC network registry client to fetch `signing_public_key` for the `subscriber_id` and `ukId`
   - **Registry public key validation**: Validate decoded public key size matches `ed25519.PublicKeySize` (32 bytes)
   - Return error code `65002` for invalid registry public key size
   - **Note**: Registry caching (LRU + TTL) is optional optimization, not required for correctness

4. **Signature Verification Process** (Logistics-correct per ONDC v1.2.0):
   - **Critical**: Payload must be exact raw JSON bytes as received (no re-marshaling or whitespace normalization)
   - Generate Blake2b hash (256-bit) from raw JSON payload bytes
   - Decode base64-encoded signature from authorization header
   - Verify ed25519 signature against Blake2b hash directly
   - **On verification failure**: Return HTTP 401 Unauthorized with error code `65002`
   - **Explicitly NOT implemented**: HTTP Signature canonical strings, (created)/(expires) as mandatory signature inputs, header-based canonicalization

5. **Timestamp Validation**:
   - Verify request timestamp (`context.timestamp`) to prevent replay attacks
   - Reject requests outside acceptable time window (configurable via `ONDC_TIMESTAMP_WINDOW`, default: 300 seconds)
   - Check for stale requests (timestamp earlier than previously processed request with same `transaction_id` + `message_id`)
   - Return NACK with error code `65003` for stale requests or invalid timestamp format

**Response Signing** (Outgoing ONDC Responses):
1. **Subscriber Identity Binding**:
   - Configure Seller NP `subscriber_id` and `ukId` at startup via `ONDCConfig.SubscriberID` and `ONDCConfig.UkID`
   - Config validation requires both fields to be present
   - Use configured subscriber identity automatically in `SignResponse` (no parameters needed)

2. **Generate Signature** (Logistics-correct per ONDC v1.2.0):
   - **Critical**: Payload must be exact raw JSON bytes to be sent (no re-marshaling or whitespace normalization)
   - Generate Blake2b hash (256-bit) from raw JSON payload bytes
   - Sign the Blake2b hash using gateway's private signing key (ed25519)
   - Base64 encode the signature
   - Construct `Authorization` header: `keyId="{subscriber_id}|{ukId}|ed25519", signature="{base64_signature}"`
   - **Explicitly NOT implemented**: HTTP Signature canonical strings, (created)/(expires) as mandatory signature inputs

3. **Key Loading**:
   - **Fail-fast initialization**: Service initialization returns error if private/public keys cannot be loaded or decoded
   - Validate key sizes: `ed25519.PrivateKeySize` (64 bytes) and `ed25519.PublicKeySize` (32 bytes)
   - Prevent partially initialized service state
   - Keys must be base64-encoded in files specified by `ONDC_PRIVATE_KEY_PATH` and `ONDC_PUBLIC_KEY_PATH`

**Replay Protection**:
- Track processed request hashes (idempotency) using `transaction_id` + `message_id`
- Reject duplicate requests within time window
- Support configurable time window for timestamp validation (default: 300 seconds per `ONDC_TIMESTAMP_WINDOW` config)

**Error Codes**:
- `65002`: Authentication failed (invalid header, missing fields, signature verification failure, unsupported algorithm, invalid key size)
- `65003`: Stale request (timestamp outside acceptable window, invalid timestamp format)
- `65011`: Registry unavailable (dependency error when registry lookup fails)
- `65020`: Internal error (key not loaded, subscriber identity not configured)

**Configuration Requirements**:
- `ONDC_PRIVATE_KEY_PATH`: Path to base64-encoded ed25519 private key file
- `ONDC_PUBLIC_KEY_PATH`: Path to base64-encoded ed25519 public key file
- `ONDC_SUBSCRIBER_ID`: Seller NP subscriber identifier (required)
- `ONDC_UK_ID`: Seller NP unique key identifier (required)
- `ONDC_TIMESTAMP_WINDOW`: Timestamp validation window in seconds (default: 300)

**Implementation Notes**:
- **Service Location**: `internal/services/ondc/ondc_auth_service.go`
- **Dependency Injection**: Uses `RegistryClient` interface for ONDC network registry lookup
- **Defensive Validation**: Registry public key size validation, early header validation, empty header rejection
- **Payload Canonicalization**: Upstream must preserve exact raw JSON bytes (architectural requirement)

**Compliance**: Must comply with ONDC network security requirements as specified in [ONDC API Contract v1.2.0](https://docs.google.com/document/d/1-xECuAHxzpfF8FEZw9iN3vT7D3i6yDDB1u2dEApAjPA/edit) and ONDC Logistics API Contract v1.2.0. Implementation follows Logistics-correct signature verification (Blake2b hash of raw payload bytes, ed25519 signing, no HTTP canonical strings).

### 3.2 Client Authentication

**Purpose**: Authenticate and authorize client requests at runtime (data plane).

**Architecture Context**:
- **UOIS Gateway** is the **data plane** for client authentication
- **Admin Service** is the **control plane** (generates credentials, publishes events)
- UOIS Gateway maintains its own **client registry** (Postgres-E) synced via events
- **No shared databases**: UOIS Gateway does NOT query Admin Service's database directly
- **Event-driven sync**: UOIS Gateway consumes `client.*` events from Admin Service

**Functional Requirements**:

- **Client Registry Ownership**:
  - UOIS Gateway owns `client_registry` table in Postgres-E (its own database)
  - Client registry is a local projection of Admin Service's client data
  - Synced via event consumption (not direct DB queries)

- **Event Consumption**:
  - Subscribe to Redis Streams: `stream:admin.client.events`
  - Consume events:
    - `client.created` → Upsert client into local registry
    - `client.updated` → Update client in local registry
    - `client.suspended` → Update status to SUSPENDED
    - `client.revoked` → Update status to REVOKED
    - `client.api_key_rotated` → Update `client_secret_hash` in local registry
  - **Idempotency**: Events are idempotent (safe to replay/retry)
  - **At-least-once delivery**: Handle duplicate events gracefully (upsert based on `client_id`)

- **Runtime Authentication Flow**:
  1. **Extract Credentials**:
     - Parse `Authorization` header:
       - `Basic` auth: `base64(client_id:client_secret)` → extract `client_id` and `client_secret`
       - `Bearer` token: Single opaque API key → extract `client_id` from key format or lookup
     - Extract client IP from `X-Real-IP` or `X-Forwarded-For` (trusted proxy headers, not `req.RemoteAddr`)
     - **Implementation Note**: Trusted proxy checker validates `RemoteAddr` against configured CIDR ranges before trusting headers (prevents IP allowlist bypass via header spoofing)

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

- **Client Registry Schema** (UOIS Gateway's Postgres-E):
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

  CREATE INDEX idx_client_registry_client_id ON client_registry(client_id);
  CREATE INDEX idx_client_registry_status ON client_registry(status);
  ```

- **Caching Strategy**:
  - Redis cache: `client:{client_id}` → full client record
  - TTL: 5 minutes (invalidate on events)
  - Cache invalidation: On any `client.*` event, delete cache entry

- **Scope Enforcement**:
  - Enforce client-specific API scopes (if configured)
  - Validate IP allowlisting (CIDR matching)
  - Support per-client rate limits

- **Internal Authentication**:
  - Authenticate internal/admin calls using internal API keys
  - Separate authentication for internal service webhooks
  - Internal keys bypass client registry (static validation)

- **Transition from API-based Validation**:
  - **Current (v0)**: UOIS Gateway calls Admin Service API per request
    - `authResp, err := adminService.AuthenticateClient(r.Context(), apiKey, "")`
    - Valid per ADR-003 (API calls allowed), but adds latency
  - **Target (v1)**: UOIS Gateway validates locally (Redis + Postgres-E)
    - No synchronous Admin Service calls in hot-path
    - Optional fallback API call only for cache warm-up (first-time lookup)
  - **Migration Path**:
    1. Implement event consumer and local registry
    2. Dual-write: Validate locally AND call Admin API (for verification)
    3. Monitor sync lag and validation accuracy
    4. Once stable, remove Admin API call from hot-path
    5. Keep Admin API call as optional fallback for cache warm-up only

### 3.3 Rate Limiting

**Purpose**: Prevent abuse and ensure fair resource usage.

**Functional Requirements**:
- Apply per-client rate limiting
- Return HTTP 429 when rate limit exceeded
- Support configurable rate limits per client
- Include rate limit headers in responses (X-RateLimit-Remaining, X-RateLimit-Reset, Retry-After)
- Log rate limit violations for monitoring
- **Implementation Note**: Redis-based sliding window counter with expiry set only on first increment (prevents window reset bug), accurate resetAt calculation using Redis TTL

### 3.4 Client Configuration Processing

**Purpose**: Fetch and process client-specific configuration from Admin Service to comply with client requirements.

**Functional Requirements**:
- **Configuration Fetching**:
  - Fetch client configuration from Admin Service (gRPC) using `client_id`
  - Cache client configuration in Redis with TTL (default: 15 minutes)
  - Invalidate cache on configuration updates (via Admin Service webhook or cache invalidation event)

- **Configuration Processing**:
  - Parse client-specific settings 
  - Determine processing rules based on client requirements:
    - Protocol handling (ONDC vs Beckn)
    - Callback URL overrides (if client-specific callback URLs configured)
    - Feature enablement (service types, payment methods, delivery options)
    - SLA requirements and timeout configurations
  - Apply client-specific transformations and validations

- **Storage**:
  - **Redis**: Store active client configuration cache (`client_config:{client_id}`)
    - TTL: 15 minutes (refreshed on access)
    - Used for fast lookup during request processing
  - **Postgres-E (Client Registry)**: Persist client configuration snapshot in `client_registry.client_configs` table
    - Store configuration history for audit and compliance
    - Fields: `client_id`, `config_snapshot` (JSONB), `effective_from`, `effective_to`, `created_at`
    - Used for dispute resolution and historical analysis

- **Configuration Usage**:
  - Apply client config during request validation and transformation
  - Override default callback URLs if client-specific URLs configured
  - Enforce client-specific rate limits and SLA requirements
  - Enable/disable features based on client feature flags

**Data Flow**:
1. Request arrives → Extract `client_id` from authentication context
2. Check Redis cache for client config
3. If cache miss → Fetch from Admin Service (gRPC)
4. Process config → Apply client-specific rules
5. Store in Redis cache (for subsequent requests)
6. Persist snapshot to Postgres-E client registry (for audit)

---

## 4. Request Validation & Transformation

**Purpose**: Ensure data integrity and protocol compatibility.

**Functional Requirements**:
- **Validation**:
  - Validate required fields, enums, data formats
  - Validate coordinates, IDs, timestamps
  - Validate ONDC/Beckn schema compliance
  - **TTL Validation**:
    - Extract `ttl` from request context (ISO 8601 duration format, e.g., `PT30S`, `PT15M`)
    - Validate TTL format and ensure callback can be sent within TTL period
  - **Stale Request Detection**:
    - Check if request timestamp is earlier than previously processed request (same `transaction_id` + `message_id`)
    - Return NACK with error code `65003` for stale requests
    - Implement timestamp validation middleware
  - **Quote Validation**:
    - For `/init` requests: Call Order Service (gRPC) to validate search_id TTL and quote validity before publishing INIT_REQUESTED event
    - For `/confirm` requests: Validate `quote_id` exists and is within TTL (15 minutes from `/on_init`)
    - Reject expired quotes with appropriate error response
  - Reject invalid requests with standardized error responses

- **Transformation**:
  - Normalize payment types (ONDC → internal)
  - Map order states and fulfillment states
  - Transform categories (FOOD, GROCERY, etc.) to internal service types
  - Handle protocol version differences
  - Transform client requests to event payloads (SEARCH_REQUESTED, INIT_REQUESTED, CONFIRM_REQUESTED)
  - Transform event responses back to ONDC/Beckn format
  - Extract callback URL information from ONDC requests:
    - Extract `bap_uri` from `context.bap_uri`
    - Extract `transaction_id` and `message_id` from `context` for callback correlation
    - Construct callback URL: `{bap_uri}/on_{action}`

- **Event Publishing**:
  - Generate search_id (Serviceability ID) for `/search` requests (UOIS Gateway generates search_id, never derives from request payload)
  - Publish `SEARCH_REQUESTED` event to stream `stream.location.search` with search_id, pickup coordinates (lat, lng), drop coordinates (lat, lng), traceparent
  - Publish `INIT_REQUESTED` event to stream `stream.uois.init_requested` with search_id, pickup, drop, package info, traceparent
  - Publish `CONFIRM_REQUESTED` event to stream `stream.uois.confirm_requested` with quote_id, client_id, payment_info, traceparent
  - Subscribe to corresponding response events:
    - `QUOTE_COMPUTED` from stream `quote:computed`
    - `QUOTE_CREATED` from stream `stream.uois.quote_created`
    - `QUOTE_INVALIDATED` from stream `stream.uois.quote_invalidated`
    - `ORDER_CONFIRMED` from stream `stream.uois.order_confirmed`
    - `ORDER_CONFIRM_FAILED` from stream `stream.uois.order_confirm_failed`

- **Response Composition**:
  - Compose `/search` response from `QUOTE_COMPUTED` event (Quote Service passes through serviceability fields, so all data is in one event)
  - Compose `/init` response from `QUOTE_CREATED` event from stream `stream.uois.quote_created` or `QUOTE_INVALIDATED` event from stream `stream.uois.quote_invalidated` (include quote TTL: `PT15M`)
  - Compose `/on_confirm` callback from `ORDER_CONFIRMED` event from stream `stream.uois.order_confirmed` or `ORDER_CONFIRM_FAILED` event from stream `stream.uois.order_confirm_failed`
  - Handle timeout scenarios when events are not received within request TTL period (30 seconds for most APIs)
  - Log timeout events when callbacks cannot be sent within TTL for monitoring and alerting

- **Error Normalization**:
  - Map internal errors to ONDC/Beckn error codes
  - Standardize error response format across all channels
  - Include error context and actionable messages
  - Propagate error status codes correctly

---

## 5. Idempotency & Deduplication

**Purpose**: Prevent duplicate order creation and ensure idempotent operations.

**Functional Requirements**:
- **Order Creation Idempotency**:
  - Enforce idempotency keys for order creation requests
  - Maintain order reference table with unique client request hash
  - Return existing order details if duplicate request detected
  - Support idempotency across retries and event callbacks

- **search_id-based Deduplication**:
  - Track search_id (Serviceability ID) for `/search` requests (generated by UOIS Gateway)
  - Prevent duplicate `/search` requests using search_id
  - Resolve search_id from transaction_id reference in `/init` requests (correlates to original `/search` request)

- **Request Deduplication**:
  - Track processed request hashes (ONDC requests)
  - Detect and handle duplicate requests within time window
  - Support configurable deduplication window

- **Event Deduplication**:
  - Prevent duplicate event processing (idempotent event handlers)
  - Track processed event IDs to prevent duplicate callbacks
  - Handle event retries gracefully

**Idempotency Keys**:
- ONDC request hash (ONDC requests)
- search_id (Serviceability ID) for `/search` and `/init` correlation
- Internal request ID (event stream events)

---

## 6. Event-Driven Callback Relay

### 6.1 Event Consumption for Callbacks

**Purpose**: Relay order status updates from event stream to clients via callbacks.

**Functional Requirements**:
- All callback delivery follows the *Common Request Processing Contract (Section 1)* for ACK behavior, TTL handling, retry logic, and audit logging unless explicitly overridden.
- Subscribe to event streams for order status events:
  - `QUOTE_COMPUTED` from stream `quote:computed` (for `/search` response composition → `/on_search` callback)
  - `QUOTE_CREATED` from stream `stream.uois.quote_created` / `QUOTE_INVALIDATED` from stream `stream.uois.quote_invalidated` (for `/init` response → `/on_init` callback)
  - `ORDER_CONFIRMED` from stream `stream.uois.order_confirmed` / `ORDER_CONFIRM_FAILED` from stream `stream.uois.order_confirm_failed` (for `/confirm` response → `/on_confirm` callback)
  - `ORDER_STATUS_UPDATED` (for `/status` response → `/on_status` callback) - Note: Currently fetched via gRPC, not events
  - `ORDER_CANCELLED` (for `/cancel` response → `/on_cancel` callback) - Note: Currently fetched via gRPC, not events
  - `ORDER_UPDATED` (for `/update` response → `/on_update` callback) - Note: Currently fetched via gRPC, not events
  - `ORDER_TRACKING_UPDATED` (for `/track` response → `/on_track` callback) - Note: Currently fetched via gRPC, not events
- **Callback URL Construction** (for ONDC clients):
  - Extract `bap_uri` from request context (`context.bap_uri`)
  - Construct callback URL: `{bap_uri}/on_{action}`
  - Example: If `bap_uri = "https://logistics_buyer.com/ondc"`, then `/on_search` callback URL is `https://logistics_buyer.com/ondc/on_search`
- Look up client/callback configuration from Admin Service
- Compose callback payloads from event data
- Send callbacks to client callback URLs:
  - `/on_search`: Catalog with fulfillment options, pricing, and terms
  - `/on_init`: Quote, cancellation terms, and transaction-level contract terms
  - `/on_confirm`: Order acceptance/rejection, fulfillment slots, agent details, AWB number
  - `/on_status`: Current order status, fulfillment states, proof of pickup/delivery
  - `/on_cancel`: Cancellation details and updated quote
  - `/on_update`: Updated order details and updated quote (if weight/dimensions changed)
  - `/on_track`: Tracking information (GPS coordinates or tracking URL)
- **Callback Correlation**:
  - Use `transaction_id` and `message_id` from request context to correlate callbacks with original requests
  - Store correlation references in Redis for callback delivery
- **Retry Logic**:
  - Implement exponential backoff retry policy
  - Support configurable max retries
  - Handle transient failures (network, timeouts)
  - Dead Letter Queue (DLQ) for failed deliveries after max retries
- **Observability**:
  - Log all delivery attempts with timing
  - Track delivery status (success, failed, retrying)
  - Alert on persistent failures
  - Support manual replay from DLQ

**Retry Policy**: 
- **Exponential backoff**: 1s → 2s → 4s → 8s → 15s (adjusted to fit within PT30S)
- **Max retries**: 5 attempts
- **After max retries**: Move to Dead Letter Queue (DLQ) for manual replay
- **Retryable errors**: Only retry errors with `retryable: true` in error taxonomy (see Section 7)
- **TTL Constraint**: Total retry duration (sum of all backoff delays) MUST NOT exceed ONDC Request TTL (`PT30S` = 30 seconds). Adjust max retries or backoff intervals to ensure all retries complete within TTL period.
- **TTL-Aware Retry Formula**: `sum(CALLBACK_RETRY_BACKOFF_*) <= ONDC_REQUEST_TTL_SECONDS`
  - Example calculation: 1s + 2s + 4s + 8s + 15s = 30s (within PT30S limit)
  - Note: Original 16s backoff (1s + 2s + 4s + 8s + 16s = 31s) exceeds PT30S limit, so last retry adjusted to 15s

**Event Consumption Pattern (Redis Streams Consumer Groups)**:
- **Consumer Group**: `uois-gateway-consumers` (shared across all UOIS Gateway instances)
- **Consumer Name**: `uois-gateway-{instance-id}` (unique per instance)
- **Consumption Method**: Use `XREADGROUP` command with blocking reads (`BLOCK` parameter)
- **Event Filtering**: Filter consumed events by business correlation ID (`search_id`, `quote_id`) after consumption (Redis Streams does not support filtering by field)
- **Note**: These are business IDs for event correlation, NOT WebSocket `correlation_id` (which UOIS Gateway never uses)
- **ACK Mechanism**: ACK messages after successful processing using `XACK` command
- **Timeout Handling**: Implement timeout logic using `BLOCK` parameter and request start time comparison

UOIS consumes events with business correlation IDs (search_id, quote_id, dispatch_order_id) from the following streams:
- **Note**: These are business IDs for event correlation, NOT WebSocket `correlation_id` (which UOIS Gateway never uses)
  - `quote:computed` - for `QUOTE_COMPUTED` events (filter by `search_id` after consumption)
  - `stream.uois.quote_created` - for `QUOTE_CREATED` events (filter by `search_id` after consumption)
  - `stream.uois.quote_invalidated` - for `QUOTE_INVALIDATED` events (filter by `search_id` after consumption)
  - `stream.uois.order_confirmed` - for `ORDER_CONFIRMED` events (filter by `quote_id` after consumption)
  - `stream.uois.order_confirm_failed` - for `ORDER_CONFIRM_FAILED` events (filter by `quote_id` after consumption)
- Event correlation: Match events to original requests using business IDs (search_id/quote_id/dispatch_order_id) after consuming from stream
  - **Note**: `dispatch_order_id` is used for internal event correlation only; external ONDC callbacks use `order.id` (ONDC)
- For ONDC callbacks: Also use `transaction_id` and ONDC `message_id` from request context for correlation
- **Note**: ONDC `message_id` is from ONDC `context.message_id`, NOT Redis Stream message_id (which is only used for ACK)

### 6.2 Async Callback Delivery

**Purpose**: Deliver callbacks asynchronously without blocking request processing.

**Functional Requirements**:
- All callback delivery follows the *Common Request Processing Contract (Section 1)* for ACK behavior, TTL handling, retry logic, and audit logging unless explicitly overridden.
- **Idempotency Requirement**: All `on_*` callbacks (`/on_search`, `/on_init`, `/on_confirm`, `/on_status`, `/on_cancel`, `/on_update`, `/on_track`) must be idempotent. Buyer NPs must handle duplicate callbacks gracefully (e.g., using `transaction_id` + `message_id` for deduplication). UOIS Gateway may retry callback delivery due to transient failures, and Buyer NPs should return ACK for duplicate callbacks without reprocessing.
- **Asynchronous Response Pattern**: Callbacks triggered by corresponding events:
  - `/on_search`: When `QUOTE_COMPUTED` event received
  - `/on_init`: When `QUOTE_CREATED` or `QUOTE_INVALIDATED` event received
  - `/on_confirm`: When `ORDER_CONFIRMED` or `ORDER_CONFIRM_FAILED` event received
  - `/on_status`: When order status is fetched from Order Service
  - `/on_cancel`: When `ORDER_CANCELLED` event received
  - `/on_update`: When `ORDER_UPDATED` event received
  - `/on_track`: When tracking data is fetched from Order Service (which aggregates location data internally)

---

## 7. Error Handling & Standardization

**Purpose**: Provide consistent error responses across all channels.

**Functional Requirements**:
- **Standard Error Schema**:
  - Consistent error response format for all APIs
  - Include error code, message, and context
  - Map internal errors to protocol-specific error codes (ONDC/Beckn)
  - Support error localization (if required)

**Error Taxonomy**:

| Error Code | Category | HTTP Status | Retryable | Example | Action |
|------------|----------|-------------|-----------|---------|--------|
| `65001` | Validation | 400 Bad Request | No | Missing required field (`search_id`, `quote_id`, `order_id`) | Return NACK immediately |
| `65002` | Authentication | 401 Unauthorized | No | Invalid client credentials, missing `Authorization` header | Return NACK immediately |
| `65003` | Stale Request | 400 Bad Request | No | Request timestamp earlier than previously processed request (same `transaction_id` + `message_id`) | Return NACK with error code `65003` |
| `65004` | Quote Expired | 400 Bad Request | No | Quote TTL (`PT15M`) expired before `/confirm` | Return NACK, require new `/init` |
| `65005` | Quote Invalid | 400 Bad Request | No | Quote not found or invalid state | Return NACK, require new `/init` |
| `65006` | Order Not Found | 404 Not Found | No | `dispatch_order_id` not found in order reference lookup | Return NACK, verify order_id |
| `65007` | Invalid State Transition | 400 Bad Request | No | Order state does not allow requested operation (e.g., cancel after delivery) | Return NACK with current state |
| `65010` | Dependency Timeout | 503 Service Unavailable | Yes | Quote Service timeout, Order Service timeout | Retry with exponential backoff, return timeout after max retries |
| `65011` | Dependency Unavailable | 503 Service Unavailable | Yes | Quote Service down, Order Service down | Retry with exponential backoff, return error after max retries |
| `65012` | Rate Limit Exceeded | 429 Too Many Requests | Yes | Client rate limit exceeded | Return 429, include `Retry-After` header |
| `65020` | Internal Error | 500 Internal Server Error | No | Database error, unexpected exception | Log error, return generic error to client |
| `65021` | Callback Delivery Failed | N/A (async) | Yes | HTTP POST to `{bap_uri}/on_*` failed | Retry with exponential backoff (1s → 2s → 4s → 8s → 16s), max 5 attempts, then DLQ |

**Error Categories**:
- **Validation errors** (400 Bad Request): `65001`, `65003`, `65004`, `65005`, `65007`
- **Authentication errors** (401 Unauthorized): `65002`
- **Not found errors** (404 Not Found): `65006`
- **Rate limit errors** (429 Too Many Requests): `65012`
- **Internal errors** (500 Internal Server Error): `65020`
- **Service unavailable** (503 Service Unavailable): `65010`, `65011`
- **Async callback errors** (N/A): `65021`

**Error Propagation**:
- Propagate Order Service and Quote Service errors correctly
- Map internal service errors to UOIS error codes (see table above)
- Mask sensitive internal error details (database errors, stack traces)
- Include actionable error messages for clients
- Log full error details with `trace_id` for troubleshooting
- **Implementation Note**: Middleware returns sanitized generic error message ("request rejected") to clients while logging full error details internally (prevents information leakage)

---

## 8. Storage & Caching

**Purpose**: Optimize performance and maintain request context.

**Functional Requirements**:
- **Temporary Storage** (Redis/key-value):
  - Order records (storing multiple identifiers on same record: order.id (ONDC), dispatch_order_id, quote_id, search_id):
    - `search_id` → `quote_id` reference (created in `/init`, search_id generated by UOIS Gateway, quote_id generated by Order Service)
    - `quote_id` → `order.id` (ONDC) → `dispatch_order_id` reference (created in `/confirm` when `ORDER_CONFIRMED` event received, order.id generated by Seller NP, dispatch_order_id generated by Order Service)
    - Order record stores `order.id` (ONDC, network-facing) alongside `dispatch_order_id` (internal execution identifier) on same record
  - search_id (Serviceability ID) tracking for `/search` and `/init` correlation
  - quote_id tracking for `/init` and `/confirm` correlation
  - Temporary entities (dispatch_order_id, quote_id, package details)
  - Request context for response reconstruction (search_id, quote_id, dispatch_order_id)
  - Callback context for ONDC requests (bap_uri, transaction_id, message_id, callback URL)
  - Billing and contacts for response reconstruction
  - Issue and grievance context:
    - Issue storage: `ondc:issue:{issue_id}` (TTL: 30 days)
    - Zendesk ticket reference: `ondc:zendesk_ticket:{zendesk_ticket_id}` → `issue_id`
    - Financial resolution data: `ondc:financial:{issue_id}`
    - GRO (Grievance Redressal Officer) details
  - Financial notification context
  - Idempotency keys and request hashes
  - Event subscription state (waiting for QUOTE_COMPUTED from `quote:computed`, QUOTE_CREATED from `stream.uois.quote_created`, ORDER_CONFIRMED from `stream.uois.order_confirmed`, ORDER_CONFIRM_FAILED from `stream.uois.order_confirm_failed`)

- **Caching**:
  - Cache serviceability responses (short TTL, 10 minutes per main flow)
  - Cache quote responses (short TTL, 15 minutes per main flow)
  - Cache status and tracking responses (short TTL)
  - Cache client configuration from Admin Service (with invalidation)
  - Reduce internal service load through intelligent caching

**Cache TTL**: Configurable per response type (typically 10-15 minutes for serviceability/quotes, 15-60 seconds for status)

---

## 9. Issue & Grievance Management (IGM)

**Purpose**: Handle ONDC-compliant issue and grievance management as Seller NP (BPP).

**Note**: IGM requires a separate database instance for data isolation and separation of concerns. All IGM-related data should be stored in a dedicated database separate from other service databases.

**Core Responsibilities**:
- Act as bridge between ONDC Network (Buyer NPs), Zendesk Helpdesk, and Redis storage
- Ensure bidirectional sync: ONDC issues → Zendesk tickets, and Zendesk ticket updates → ONDC status callbacks
- **Note**: Buyers have their own ticket dashboard and create tickets using endpoints which should be proxied to Zendesk
- Maintain issue state and references in Redis for callback reconstruction
- Support multiple issues per order (different items/fulfillments)
- Handle issue cascading and escalation

### 9.1 ONDC IGM API Endpoints (as Seller NP)

#### 9.1.1 `/issue` Endpoint (Receive Issues from Buyer NPs)

**Functional Requirements**:
- Validate ONDC issue request payload and required fields
- Extract issue details:
  - `issue_id` (unique identifier for the issue)
  - `category` (e.g., ORDER, FULFILLMENT, PAYMENT)
  - `sub_category` (specific sub-category within category)
  - `description` (issue description)
  - `complainant_info` (buyer NP information)
  - `order_details` (order ID, item IDs, fulfillment IDs)
- Store issue in Redis with key `ondc:issue:{issue_id}` (TTL: 30 days)
- Create issue ticket in Zendesk Helpdesk via `ZendeskService` (proxied from buyer ticket dashboard endpoint)
- Build `/on_issue` callback response with:
  - `issue_id`
  - `status` (e.g., OPEN, CLOSED)
  - `issue_actions` (respondent actions, cascaded levels)
- Return synchronous HTTP 200 OK ACK response to Buyer NP
- Send `/on_issue` callback to Buyer NP `bap_uri` asynchronously at `{bap_uri}/on_issue`
- Persist request/response to Postgres-E (`audit.request_response_logs`) with `issue_id`

#### 9.1.2 `/on_issue` Endpoint (Receive Callbacks from Buyer NPs)

**Functional Requirements**:
- Parse and validate `/on_issue` callback payload
- Extract issue details and actions from callback
- Update issue actions in Redis
- Acknowledge receipt of callback
- Persist callback to Postgres-E (`audit.request_response_logs`)

#### 9.1.3 `/issue_status` Endpoint (Handle Status Check Requests)

**Functional Requirements**:
- Validate ONDC `issue_status` request payload and `issue_id`
- Retrieve issue from Redis using `issue_id`
- Get GRO (Grievance Redressal Officer) details from Redis
- Build `/on_issue_status` callback response with:
  - Issue status
  - Resolution provider actions (respondent actions, cascaded levels)
  - GRO (Grievance Redressal Officer) details
  - Resolution details (if resolved)
- Return synchronous HTTP 200 OK ACK response
- Send `/on_issue_status` callback to Buyer NP `bap_uri` asynchronously at `{bap_uri}/on_issue_status`
- Persist request/response to Postgres-E (`audit.request_response_logs`) with `issue_id`

#### 9.1.4 `/on_issue_status` Endpoint (Receive Status Callbacks)

**Functional Requirements**:
- Parse and validate `/on_issue_status` callback payload
- Extract issue details and status from callback
- Update issue status in Redis
- Acknowledge receipt of callback
- Persist callback to Postgres-E (`audit.request_response_logs`)

### 9.2 Zendesk Helpdesk Integration

#### 9.2.1 ZendeskService Responsibilities

**Functional Requirements**:
- **Create Tickets in Zendesk Helpdesk from ONDC Issues**:
  - **Note**: Buyers have their own ticket dashboard and create tickets using endpoints which should be proxied to Zendesk
  - Map ONDC issue types to Zendesk priorities:
    - `DISPUTE` → Urgent priority
    - `GRIEVANCE` → High priority
    - `ISSUE` → Medium priority
  - Create/ensure contact exists in Zendesk for complainant
  - Build ticket with ONDC metadata (custom fields):
    - `ondc_issue_id`
    - `ondc_transaction_id`
    - `ondc_order_id`
    - Additional ONDC context fields
  - Add tags: `ondc`, `igm`, issue type, category, GRO level
  - Transform ISO 8601 durations to seconds for Zendesk Duration fields
- **Update Ticket Status in Zendesk**:
  - Update ticket status when ONDC issue status changes
  - Sync status changes bidirectionally
- **Get Ticket Details**:
  - Retrieve ticket details from Zendesk for status queries
  - Support ticket lookup by Zendesk ticket ID
- **Add Comments**:
  - Add comments to Zendesk tickets from ONDC issue updates
- **Authentication**:
  - Authenticate using Zendesk API Key/Secret (token-based auth)
  - Maintain secure credential storage

#### 9.2.2 Zendesk Webhook Handler Responsibilities

**Functional Requirements**:
- **Receive Webhooks from Zendesk**:
  - Endpoint: `/webhooks/zendesk/ticket_update`
  - Validate webhook signature (`X-Zendesk-Webhook-Signature` header)
  - Extract ticket update details:
    - `ticket_id`
    - `status`
    - `resolution`
    - `updated_at`
- **Map Zendesk Status to ONDC Status**:
  - Zendesk: `Open`, `Replied`, `Processing` → ONDC: `OPEN`
  - Zendesk: `Resolved`, `Closed` → ONDC: `CLOSED`
- **Update Issue Status**:
  - Update issue status in Redis using Zendesk ticket ID lookup
  - Support bidirectional lookup: `zendesk_ticket_id` → `issue_id` → issue details
- **Trigger ONDC Callbacks**:
  - Trigger `/on_issue_status` callback to Buyer NP when status changes
  - Build ONDC-compliant `/on_issue_status` response with:
    - Issue actions (respondent actions, cascaded levels)
    - Resolution provider information
    - GRO (Grievance Redressal Officer) details
    - Resolution details
  - Send callback to `{bap_uri}/on_issue_status`
- **Persist Webhook Events**:
  - Log webhook events to Postgres-E (`audit.request_response_logs`)

### 9.3 Issue Storage Management (Redis)

#### 9.3.1 Issue Storage Service

**Functional Requirements**:
- **Storage Keys**:
  - Store issues with key: `ondc:issue:{issue_id}` (TTL: 30 days)
  - Store Zendesk ticket reference: `ondc:zendesk_ticket:{zendesk_ticket_id}` → `issue_id`
  - Store financial resolution data: `ondc:financial:{issue_id}`
- **Bidirectional Lookup**:
  - Support `issue_id` → issue details lookup
  - Support `zendesk_ticket_id` → `issue_id` → issue details lookup
- **Data Updates**:
  - Update issue status, resolution provider, financial resolution
  - Store full ONDC payload for callback reconstruction
  - Maintain issue lifecycle state

#### 9.3.2 Data Stored Per Issue

**Data Structure**:
- `ondc_issue_id` (unique ONDC issue identifier)
- `zendesk_ticket_id` (Zendesk ticket identifier)
- `transaction_id` (ONDC transaction ID for correlation)
- `order_id` (ONDC order ID)
- `issue_type` (ISSUE, GRIEVANCE, DISPUTE)
- `status` (OPEN, CLOSED)
- `created_at` (timestamp)
- `updated_at` (timestamp)
- `resolution_provider` (respondent info, GRO details)
- `financial_resolution` (refund amount, payment method, transaction ref)
- `full_ondc_payload` (for callback reconstruction)
- `category`, `sub_category`, `description`
- `complainant_info` (buyer NP information)
- `order_details` (order ID, item IDs, fulfillment IDs)

### 9.4 Financial Notifications Integration

**Functional Requirements**:
- Receive payment status notifications from Admin Backend
- Receive settlement status notifications
- Receive RTO status notifications
- Store financial resolution data in Redis (`ondc:financial:{issue_id}`)
- Update related issues with financial status information
- Support financial action tracking (refunds, settlements)
- Link financial resolutions to ONDC issues for status callbacks

### 9.5 GRO (Grievance Redressal Officer) Management

**Functional Requirements**:
- Store and retrieve GRO details from Redis
- Provide default GRO details if Redis lookup fails
- Include GRO information in `/on_issue_status` responses
- Support GRO level assignment:
  - L1 for ISSUE
  - L2 for GRIEVANCE
  - L3 for DISPUTE
- Maintain GRO contact information and escalation paths

### 9.6 ONDC Compliance

**Functional Requirements**:
- Follow ONDC IGM API contract (v1.0.0)
- Maintain transaction trail using `transaction_id` and `message_id`
- Support multiple issues per order (different items/fulfillments)
- Handle issue cascading and escalation
- Provide proper error codes and NACK responses
- Ensure callback URL construction: `{bap_uri}/on_issue` and `{bap_uri}/on_issue_status`
- Use `transaction_id` and `message_id` from request context to correlate callbacks
- Support issue lifecycle: created → in-progress → resolved/closed
- Maintain audit trail for dispute resolution (7-year retention in Postgres-E)

**Boundary**: UOIS Gateway creates and syncs tickets with Zendesk Helpdesk. Issue resolution is handled by external helpdesk (Zendesk) support team. UOIS Gateway maintains bidirectional sync between ONDC network and Zendesk Helpdesk. Buyers have their own ticket dashboard and create tickets using endpoints which should be proxied to Zendesk.

---

## 10. Data Ownership & Storage

### 10.1 Data Ownership

**UOIS Gateway owns the following data** (per Bounded Contexts & Container Diagram):

| Data Entity | Purpose | Storage | Retention |
|-------------|---------|---------|-----------|
| **Client Registry** | Local projection of client credentials for runtime validation (synced via events from Admin Service) | Postgres-E (`client_registry` table) + Redis (cache) | Permanent (synced from Admin Service), 5 min cache TTL |
| **Order Records** | Multiple identifiers stored on same record: order.id (ONDC), dispatch_order_id, quote_id, search_id | Redis (temporary) + Postgres-E (audit) | 7 years (audit), 30 days (cache) |
| **Request/Response Logs** | Complete audit trail for dispute resolution | Postgres-E (audit database, `audit` schema) | 7 years minimum |
| **Idempotency Keys** | Prevent duplicate order creation | Redis (temporary) + Postgres-E (audit) | 7 years (audit), 24 hours (cache) |
| **Request Hashes** | Non-repudiation and integrity verification | Postgres-E (audit database) | 7 years minimum |
| **Webhook Delivery Logs** | Callback relay attempts and status | Postgres-E (audit database) | 7 years minimum |
| **Client Request Context** | Temporary request context for response reconstruction (search_id, quote_id, dispatch_order_id) | Redis (temporary) | 1 hour TTL |
| **Issue/Grievance State** | Issue state for callbacks (sync with Zendesk) | Redis (temporary) | 30 days |
| **Issue Data** | ONDC issue details, status, resolution provider, GRO details | Redis (temporary) | 30 days |
| **Zendesk Ticket References** | Reference lookup between Zendesk ticket IDs and ONDC issue IDs | Redis (temporary) | 30 days |
| **Financial Resolution Data** | Financial resolution data linked to issues | Redis (temporary) | 30 days |
| **IGM Request/Response Logs** | Complete audit trail for IGM requests and callbacks | Postgres-E (audit database) | 7 years minimum |

**UOIS Gateway does NOT own**:
- Order business data (Order Service owns)
- Client configuration source of truth (Admin Service owns - UOIS Gateway maintains local projection only)
- Issue resolution data and ticket content (Zendesk Helpdesk owns - UOIS Gateway only maintains sync and references)

**Client Registry Ownership**:
- **UOIS Gateway** owns `client_registry` table in Postgres-E (its own database)
- **Admin Service** owns source of truth (`clients` table in Postgres-A)
- **Sync mechanism**: Event-driven (Redis Streams), not direct DB queries
- **No shared databases**: UOIS Gateway does NOT query Admin Service's database directly (per ADR-003)

### 10.2 Storage Architecture

**Per Container Diagram (C4L2) & Database Topology**:

**Database Instance**: Following the architecture pattern (Postgres-A/B/C/D), UOIS Gateway uses **Postgres-E (Audit Database)** - a **completely separate PostgreSQL RDS instance**.

- **Primary Storage**: **PostgreSQL-E (Audit Database)** - **Separate Database Instance**
  - **Rationale for Separate Database**:
    - **Isolation**: Complete isolation from other services (cannot accidentally affect Order/Financial/Admin databases)
    - **Security**: Separate access controls, encryption keys, network policies
    - **Compliance**: Independent retention policies (7-year minimum vs 7-35 days for others)
    - **Performance**: Optimized for write-heavy audit workload (append-only, minimal indexes)
    - **Backup/DR**: Separate backup strategy (long-term archival to S3)
    - **Tamper Resistance**: Isolated database reduces risk of unauthorized access/modification
  - Purpose: Immutable audit logs, order references, request/response pairs
  - Database Name: `postgres_audit` (or `postgres_e`)
  - Access: Write-only for logs (immutable), read for dispute resolution
  - **RTO/RPO**: 30 min RTO, 1 hour RPO (per SLOs/RTO/RPO document)

- **Temporary Storage**: Redis Cluster (ElastiCache)
  - Purpose: Order references cache, idempotency keys, request context
  - TTL: Configurable (typically 1-24 hours)
  - Eviction: LRU policy for memory management

**Storage Pattern**:
```
┌─────────────────────────────────────────────────────────┐
│                    UOIS Gateway                         │
│                                                          │
│  ┌──────────────────┐      ┌──────────────────┐       │
│  │   Request Flow   │      │  Response Flow   │       │
│  └────────┬─────────┘      └────────┬─────────┘       │
│           │                          │                  │
│           ▼                          ▼                  │
│  ┌──────────────────────────────────────────┐          │
│  │      Redis (Temporary Cache)              │          │
│  │  • Order references (30 days)               │          │
│  │  • Idempotency keys (24 hours)            │          │
│  │  • Request context (1 hour)                │          │
│  └──────────────────────────────────────────┘          │
│           │                          │                  │
│           ▼                          ▼                  │
│  ┌──────────────────────────────────────────┐          │
│  │   PostgreSQL-E (Audit Database)          │          │
│  │   [Separate Database Instance]            │          │
│  │  • Request/response logs (7 years)        │          │
│  │  • Order references (7 years)                │          │
│  │  • Request hashes (7 years)               │          │
│  │  • Webhook delivery logs (7 years)         │          │
│  └──────────────────────────────────────────┘          │
└─────────────────────────────────────────────────────────┘
```

### 10.3 Log Storage Requirements

**Per Security Threat Model & SLOs/RTO/RPO**:

- **Audit Log Schema** (PostgreSQL, `audit.request_response_logs`):
  - `request_id` (PK, UUID)
  - `client_id` (FK to client config)
  - `protocol_type` (enum: 'ONDC', 'BECKN') - Source protocol format
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
  - `scopes_evaluated` (array of strings)
  - `environment` (sandbox/production)
  - `search_id` (UUID, nullable) - Serviceability ID for `/search` and `/init` correlation
  - `quote_id` (UUID, nullable) - Quote ID for `/init` and `/confirm` correlation
  - `dispatch_order_id` (UUID, nullable) - Internal execution ID (post-confirm mapping, never sent in ONDC payloads)
  - `transaction_id` (string, nullable) - ONDC transaction ID for callback correlation
  - `message_id` (string, nullable) - ONDC message ID for callback correlation (from ONDC `context.message_id`, NOT Redis Stream message_id)
  - `bap_uri` (string, nullable) - Buyer NP URI for callback URL construction
  - `trace_id` (string, nullable) - Distributed tracing identifier (extracted from traceparent). Used for end-to-end correlation and troubleshooting.
  - `traceparent` (string, nullable) - W3C traceparent header (full format: `00-<trace-id>-<span-id>-<flags>`). Used for distributed tracing across sync + async hops.
  - `created_at` (timestamp, immutable)

  **Audit Logging Pattern** (per main order flows):
  - `/search`: Persists request and `/on_search` callback response with `search_id`
  - `/init`: Persists request and `/on_init` callback response with `search_id` and `quote_id`
  - `/confirm`: Persists request and `/on_confirm` callback response with `quote_id` and `dispatch_order_id`
  - `/status`: Persists request and `/on_status` callback response with `dispatch_order_id`
  - `/track`: Persists request and `/on_track` callback response with `dispatch_order_id`
  - `/cancel`: Persists request and `/on_cancel` callback response with `dispatch_order_id`
  - `/update`: Persists request and `/on_update` callback response with `dispatch_order_id`
  - `/rto`: Persists request/response with `dispatch_order_id` (Note: ONDC uses `/update` flow for RTO)
  - **IGM flows**:
    - `/issue`: Persists request and `/on_issue` callback response with `issue_id`
    - `/on_issue`: Persists callback with `issue_id`
    - `/issue_status`: Persists request and `/on_issue_status` callback response with `issue_id`
    - `/on_issue_status`: Persists callback with `issue_id`
    - Zendesk webhook events: Persists webhook events with `zendesk_ticket_id` and `issue_id`
  - All logs stored in Postgres-E (`audit` schema) with 7-year retention
  - For ONDC requests: Also store `transaction_id` and `message_id` from request context for callback correlation
  
  **JSONB Columns**: 
  - `request_payload` and `response_payload` use JSONB to store **original formats** (ONDC/Beckn)
  - No schema changes needed when adding new client formats
  - Supports querying with PostgreSQL JSONB operators
  - Can create GIN indexes on frequently queried JSON paths

- **Webhook Delivery Log Schema**:
  - `webhook_id` (PK, UUID)
  - `request_id` (FK to audit log)
  - `client_id` (FK)
  - `webhook_url` (destination URL)
  - `delivery_attempt` (integer)
  - `status` (pending, success, failed, retrying)
  - `response_code` (HTTP status from client)
  - `response_body` (JSONB, if applicable)
  - `retry_count` (integer)
  - `next_retry_at` (timestamp)
  - `delivered_at` (timestamp)
  - `failure_reason` (text)
  - `created_at` (timestamp)

- **Order Reference Schema**:
  - `reference_id` (PK, UUID)
  - `order.id` (ONDC) (string, nullable) - Seller-generated order identifier (network-facing)
    - **Terminology**: This is the ONDC network-facing order identifier generated by Seller NP
    - **Source**: Generated by Seller NP (UOIS Gateway / Order Service) during `/confirm` flow, sent in `/on_confirm` → `message.order.id`
    - **Ownership**: Seller domain, network-facing
    - **Echo contract**: Buyer echoes seller-generated `order.id` in all post-order APIs
  - `dispatch_order_id` (string, FK to Order Service) - Internal order ID (created by Order Service in `/confirm`)
    - **Terminology**: This is the internal Order Service identifier in human-readable format (e.g., "ABC0000001")
    - **Ownership**: Order Service domain
    - **Reference**: `order.id` (ONDC) → `dispatch_order_id` (internal-only, routing lookup only)
    - **Critical**: MUST NEVER be sent in ONDC payloads - internal-only
  - `quote_id` (UUID, nullable) - Quote ID from `/init` (used in `/confirm` request)
  - `search_id` (UUID, nullable) - Serviceability ID from `/search` (used in `/init` request)
  - `client_id` (FK)
  - `protocol_type` (enum: 'ONDC', 'BECKN') - Source protocol
  - `request_id` (FK to audit log) - Links to full request/response
  - `created_at` (timestamp)
  - `updated_at` (timestamp)

**Reference Lifecycle (Echo Contract Pattern)**:
- **`/search`**: 
  - UOIS Gateway generates `search_id` (internal-only)
  - Sends stable `provider.id` in `/on_search` callback → `message.catalog.bpp/providers[].id`
  - Creates reference entry with `search_id` (no `quote_id` or `order.id` yet)
- **`/init`**: 
  - Extracts `provider.id` from `message.order.provider.id` (echoed from `/on_search`)
  - Validates `provider.id` matches configured stable identifier
  - Uses internal `search_id` (from order record) to publish `INIT_REQUESTED` event
  - Receives `QUOTE_CREATED` event with `quote_id` (generated by Order Service)
  - Sends `quote_id` in `/on_init` callback → `message.order.quote.id`
  - Updates order record: stores `quote_id` alongside `search_id` on same record
- **`/confirm`**: 
  - Extracts `quote_id` from `message.order.quote.id` (echoed from `/on_init`)
  - Validates `quote_id` was previously received from Order Service via `QUOTE_CREATED` event
  - Publishes `CONFIRM_REQUESTED` event with `quote_id`
  - Receives `ORDER_CONFIRMED` event with `dispatch_order_id` (generated by Order Service)
  - Generates ONDC `order.id` (seller-generated, network-facing)
  - Sends ONDC `order.id` in `/on_confirm` callback → `message.order.id` (NOT dispatch_order_id)
  - Stores order record with: `quote_id`, `order.id` (ONDC), `dispatch_order_id` (internal-only)
  - **Critical**: `dispatch_order_id` is internal-only and MUST NEVER be sent in ONDC payloads
- **Post-confirmation** (`/status`, `/track`, `/cancel`, `/update`, `/rto`):
  - Extracts ONDC `order.id` from `message.order_id` (echoed from `/on_confirm` - seller-generated)
  - Looks up order record using `order.id` (ONDC)
  - Retrieves `dispatch_order_id` from order record
  - If order record not found → return NACK immediately (error code `65006`)
  - Uses `dispatch_order_id` ONLY for internal Order Service calls (gRPC methods)
  - **Critical**: `order.id` (ONDC) is the canonical post-confirmation identifier - Buyer echoes seller-generated `order.id`

**Note**: Order records store multiple identifiers on the same record; no identifier replaces or represents another. Full order data is stored by Order Service in normalized format. UOIS Gateway stores identifiers for lookup only; it does not translate, map, derive, or resolve identifiers across domains.

**Terminology Summary**:
- `order.id` (ONDC): Seller-generated order identifier (network-facing)
  - **Source**: Generated by Seller NP (UOIS Gateway / Order Service) during `/confirm` flow
  - **Ownership**: Seller domain, network-facing
  - **Usage**: Sent in `/on_confirm` → `message.order.id`, Buyer echoes in all post-order APIs
  - **Internal mapping**: `order.id` (ONDC) → `dispatch_order_id` (internal-only)
- `dispatch_order_id`: Internal Order Service identifier (e.g., "ABC0000001")
  - **Source**: Generated by Order Service when order is created
  - **Ownership**: Order Service domain
  - **Usage**: Used for all internal service calls to Order Service (gRPC methods)
  - **Critical**: MUST NEVER be sent in ONDC payloads - internal-only
- **Echo Contract**: UOIS Gateway propagates IDs to Buyer NPs in ONDC-compliant first-class fields. Buyer NPs echo these IDs unchanged in subsequent requests. UOIS Gateway extracts echoed IDs only from predefined protocol locations.
  - **Storage**: UOIS Gateway stores `order.id` (ONDC, network-facing) alongside `dispatch_order_id` (internal-only) on the same order record
- **Note**: In post-confirmation ONDC requests, `message.order_id` contains the seller-generated `order.id` (ONDC). UOIS Gateway looks up the order record using `order.id` (ONDC) to retrieve `dispatch_order_id` for internal service calls.

### 10.3.1 Handling Different Order Formats

**Problem**: UOIS Gateway receives orders in ONDC/Beckn format:
- **ONDC/Beckn**: Structured JSON with specific Beckn schema (e.g., `context`, `message`, `order`)

**Solution**: Store original formats in JSONB columns, transform to internal format before Order Service.

**Storage Strategy**:

1. **Audit Logs (Original Format)**:
   - `request_payload` (JSONB): Store **original request** in client's format (ONDC/Beckn or client-specific)
   - `response_payload` (JSONB): Store **original response** in client's format
   - **Rationale**: 
     - Preserve exact request/response for dispute resolution
     - No data loss during translation
     - Can reconstruct original format for callbacks

2. **Order References (ID Only)**:
   - Store **only order IDs** (not full order data)
   - `order.id` (ONDC): Seller-generated order identifier (network-facing)
   - `dispatch_order_id`: Internal order ID (from Order Service, internal-only)
   - **Rationale**:
     - Minimal storage footprint
     - Fast lookups for ID translation
     - Full order data owned by Order Service

3. **Translation Flow**:
   ```
   ┌─────────────────────────────────────────────────────────┐
   │              UOIS Gateway Processing                     │
   └─────────────────────────────────────────────────────────┘
   
   Step 1: Receive Request (Original Format)
   └── ONDC: { "context": {...}, "message": {...} }
   
   Step 2: Store in Audit Log (JSONB - Original Format)
   ├── request_payload: JSONB (preserves original structure)
   └── No translation yet
   
   Step 3: Transform to Internal Format
   ├── Parse ONDC/Beckn → Internal REST/gRPC
   └── Normalize to common internal schema
   
   Step 4: Send to Order Service (Normalized Format)
   └── Order Service stores in its normalized schema
   
   Step 5: Store Order Reference (IDs Only)
   ├── order.id (ONDC) - seller-generated during /confirm
   ├── dispatch_order_id (from Order Service response, internal-only)
   └── Link to audit log via request_id
   
   Step 6: Generate Response (Client's Format)
   ├── Transform Order Service response back to client format
   ├── Store in audit log: response_payload (JSONB)
   └── Return to client in original format
   ```

4. **JSONB Column Benefits**:
   - **Flexible Schema**: Can store any JSON structure (ONDC, Beckn, client-specific)
   - **Query Support**: PostgreSQL JSONB operators for filtering/searching
   - **Indexing**: Can create GIN indexes on JSONB paths for performance
   - **No Schema Changes**: Adding new client formats doesn't require schema migration

5. **Example Storage**:

   **ONDC Request** (stored in `request_payload` JSONB):
   ```json
   {
     "context": {
       "domain": "nic2004:60221",
       "action": "confirm",
       "bap_id": "buyer.example.com",
       "bap_uri": "https://buyer.example.com",
       "transaction_id": "txn_123"
     },
     "message": {
       "order": {
         "id": "ondc_order_123",
         "state": "Created",
         "items": [...]
       }
     }
   }
   ```

   **Order Reference** (stores only IDs):
   ```
   mapping_id: uuid-123
   ondc_order_id: "ondc_order_123"
   dispatch_order_id: "dispatch_order_789"
   protocol_type: "ONDC"
   ```

6. **Response Reconstruction**:
   - When generating callbacks/webhooks, UOIS Gateway:
     1. Fetches order data from Order Service (normalized format)
     2. Retrieves original request format from audit log (`request_payload`)
     3. Transforms normalized data back to client's original format
     4. Stores translated response in audit log (`response_payload`)
     5. Sends to client in their expected format

**Key Principle**: 
- **Audit Database**: Stores original formats (JSONB) for non-repudiation
- **Order Service Database**: Stores normalized order data (single schema)
- **UOIS Gateway**: Transforms between formats, stores references and audit trail

### 10.4 Database Technology Choice: PostgreSQL vs NoSQL

**Consideration**: Given the flexible JSON schemas (ONDC/Beckn/client-specific formats) and write-heavy audit workload, NoSQL databases (e.g., DynamoDB, MongoDB, AWS DocumentDB) could be considered.

**Trade-off Analysis**:

| Factor | PostgreSQL (JSONB) | NoSQL (DynamoDB/DocumentDB) |
|--------|-------------------|------------------------------|
| **ACID Guarantees** | ✅ Full ACID transactions | ⚠️ Limited (eventual consistency) |
| **Non-Repudiation** | ✅ Strong (immutable logs with constraints) | ⚠️ Weaker (no built-in immutability) |
| **JSON/Document Storage** | ✅ JSONB (mature, performant) | ✅ Native (optimized for documents) |
| **Query Flexibility** | ✅ Complex queries, joins, aggregations | ⚠️ Limited (key-value or simple queries) |
| **Compliance/Audit** | ✅ Better tooling, audit trails | ⚠️ Less mature audit capabilities |
| **Write Scalability** | ⚠️ Vertical scaling (read replicas) | ✅ Horizontal scaling (auto-scaling) |
| **Cost (7-year retention)** | ⚠️ Higher (storage grows linearly) | ✅ Lower (pay-per-use, archival tiers) |
| **Dispute Resolution Queries** | ✅ Complex queries across time ranges | ⚠️ Requires careful index design |
| **Schema Evolution** | ✅ JSONB flexible, no migration needed | ✅ Native flexibility |
| **Backup/Recovery** | ✅ Mature (PITR, snapshots) | ⚠️ Varies by service |

**Recommendation**: **PostgreSQL with JSONB** for v0, with **NoSQL as future consideration** if:
- Write volume exceeds PostgreSQL capacity (>10K req/sec sustained)
- Cost becomes prohibitive for 7-year retention
- Query patterns remain simple (key lookups only)

**Hybrid Approach** (Future):
- **PostgreSQL**: Critical audit logs (ACID, non-repudiation, complex queries)
- **DynamoDB/DocumentDB**: High-volume operational logs (write-heavy, simple queries)
- **S3**: Long-term archival (>1 year old logs)

### 10.5 Log Storage Location

**Per Container Diagram & Security Architecture**:

1. **PostgreSQL-E (Audit Database)** - **Separate Database Instance** (Recommended for v0):
   - **Location**: **Dedicated PostgreSQL RDS instance** (separate from Postgres-A/B/C/D)
   - **Database Name**: `postgres_audit` (or `postgres_e`)
   - **Rationale for Separate Database**: 
     - **Complete Isolation**: No risk of affecting other service databases
     - **Security Isolation**: Separate VPC security groups, IAM roles, encryption keys
     - **Compliance**: Independent retention policies (7-year minimum vs 7-35 days for operational databases)
     - **Performance**: Optimized for write-heavy, append-only workload (minimal indexes, partitioning by date)
     - **Backup Strategy**: Long-term archival to S3 (7-year retention), separate from operational backups
     - **Tamper Resistance**: Isolated database with restricted access (write-only for UOIS Gateway, read-only for Admin/Analytics)
     - **ACID Guarantees**: Ensures data integrity for dispute resolution
   - **Access Pattern**: 
     - Write: UOIS Gateway only (append-only, immutable logs)
     - Read: Admin Dashboard (dispute resolution), Analytics Service (metrics)
     - No updates/deletes: Database constraints prevent modification
   - **Backup Strategy**:
     - Daily snapshots (7-day PITR window for operational recovery)
     - Monthly snapshots archived to S3 (7-year retention for compliance)
     - Automated archival process for logs older than 1 year
   - **RTO/RPO**: 30 min RTO, 1 hour RPO (per SLOs/RTO/RPO document)

**Alternative: NoSQL Database** (Future Consideration):

If write volume or cost becomes a concern, consider:
- **AWS DocumentDB** (MongoDB-compatible): 
  - Native JSON document storage
  - PostgreSQL-compatible API
  - Better write scalability
  - Lower cost for large volumes
  - Trade-off: Weaker ACID guarantees
  
- **DynamoDB**:
  - Serverless, auto-scaling
  - Pay-per-use pricing
  - Excellent for high write volumes
  - Trade-off: Limited query flexibility, eventual consistency

**Migration Path**: Start with PostgreSQL, migrate to NoSQL if:
- Write volume > 10K requests/second sustained
- Storage costs exceed budget
- Query patterns remain simple (no complex joins/aggregations)

2. **Redis Cluster** (ElastiCache):
   - **Location**: Shared Redis cluster (ElastiCache)
   - **Rationale**:
     - Sub-millisecond latency for idempotency checks
     - Temporary cache for order references
     - Request context for response reconstruction
   - **TTL Strategy**:
     - Order references: 30 days
     - Idempotency keys: 24 hours
     - Request context: 1 hour
   - **Eviction**: LRU policy when memory pressure

3. **CloudWatch Logs** (Operational Logs):
   - **Location**: AWS CloudWatch Logs
   - **Purpose**: Application logs, errors, metrics (not audit trail)
   - **Retention**: 30 days (operational), 7 years (archived to S3 for compliance)
   - **Access**: Operations team, monitoring tools

### 10.6 Log Integrity & Non-Repudiation

**Per Security Threat Model**:

- **Request Hashing**:
  - Hash canonical request: `SHA-256(method + path + headers + body)`
  - Store hash in audit log for integrity verification
  - Prevent tampering of logged requests

- **Signature Storage**:
  - Store HMAC signature for client requests
  - Store ONDC request signatures
  - Enable cryptographic proof of request authenticity

- **Immutable Logs**:
  - Audit logs are append-only (no updates/deletes)
  - Use database constraints to prevent modification
  - Support tamper-evident audit trail

- **Retention Compliance**:
  - Minimum 7 years for financial transactions (per regulatory requirements)
  - Support log archival to S3 for long-term storage
  - Enable log retrieval for dispute resolution

---

## 11. Observability & Audit

**Purpose**: Enable dispute resolution and operational monitoring.

**Functional Requirements**:
- **Distributed Tracing**:
  - **Trace ID Generation** (Edge Service Responsibility):
    - Generate W3C `traceparent` header when receiving HTTP request (if not present in incoming request)
    - Format: `00-<trace-id>-<span-id>-<flags>` (W3C Trace Context standard)
    - Start root span using OpenTelemetry SDK
    - Extract trace_id from traceparent for logging convenience
  - **Trace Context Propagation**:
    - Include `traceparent` in all Redis Stream events published (`SEARCH_REQUESTED`, `INIT_REQUESTED`, `CONFIRM_REQUESTED`)
    - Propagate trace context in all gRPC calls to Order Service (OpenTelemetry SDK handles this automatically)
    - Include `traceparent` in callback headers if useful for client support (careful re: privacy/security)
  - **Trace Continuity**:
    - Extract `traceparent` from consumed events (`QUOTE_COMPUTED`, `QUOTE_CREATED`, `ORDER_CONFIRMED`)
    - Create child spans when processing events (do NOT generate new trace IDs)
    - Maintain trace continuity across sync + async hops (HTTP → Redis Streams → services → callbacks)
  - **Logging**:
    - Always log both `trace_id` and correlation IDs (`search_id`, `quote_id`, `dispatch_order_id`) together
    - Format: `INFO [service=uois] trace_id=4bf92f3577b34da6... search_id=550e8400-e29b-41d4... msg="published SEARCH_REQUESTED"`
    - Include trace_id in audit logs (Postgres-E) for fast troubleshooting
  - **Sampling**:
    - Default sampling at edge (sample p95/p99 traces to avoid cost explosion)
    - Use adaptive sampling if necessary
    - Ensure spans are created minimally for high-volume endpoints

- **ID Stack & Ownership**:
  - **UOIS Gateway Responsibilities**:
    - **Generates**: `trace_id` (via W3C `traceparent` header at edge)
    - **Extracts from auth**: `client_id` (from JWT/API key)
    - **Generates**: `search_id` (Serviceability ID for `/search` requests)
    - **Passes downstream**: `traceparent`, `client_id`, business IDs (`search_id`, `quote_id`)
    - **Never generates or uses**: `correlation_id` (WebSocket Gateway responsibility only)
  
  - **ID Stack Summary**:
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
  
  - **One-line meaning**:
    * **trace_id** → *What happened across services* (generated by UOIS Gateway)
    * **correlation_id** → *What belongs to one UI/session* (WebSocket Gateway only)
    * **client_id** → *Who owns this business* (extracted from auth)
    * **business IDs** → *What the business object is* (search_id/quote_id/dispatch_order_id)
    * **event_id** → *Did we already process this event* (for deduplication)
    * **message_id** → *Where is this message in the stream* (Redis Streams, ACK only)
  
  - **Strict Rules**:
    - **`trace_id`**: Generated by UOIS Gateway, propagated everywhere, logs + spans only, ❌ never business logic
    - **`correlation_id`**: Generated by WebSocket Gateway, ❌ never stored in DB, ❌ never enters core services
    - **`client_id`**: Extracted from auth, passed to all core services, used for pricing/billing/multi-tenancy
    - **`search_id/quote_id/dispatch_order_id`**: Pure business lifecycle identifiers (co-exist on the same order record; different APIs select different IDs)
    - **`event_id`**: Generated by event publisher, used only for deduplication, TTL-based storage
    - **`message_id`**: Generated by Redis Streams, used only for ACK/replay/lag monitoring, ❌ never stored in business tables
  
  - **One-Line Law**:
    > **UOIS Gateway generates `trace_id`, extracts `client_id`, generates `search_id`, passes business IDs downstream, uses `event_id` for deduplication, and NEVER generates or uses `correlation_id` (WebSocket Gateway responsibility exclusively).**

- **Request Logging**:
  - Log all incoming requests with:
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
    - Order ID (ONDC, seller-generated) and dispatch_order_id (internal-only, for `/confirm` and post-confirmation requests)

- **Event Logging**:
  - Log all events published (SEARCH_REQUESTED, INIT_REQUESTED, CONFIRM_REQUESTED)
  - Log all events consumed (QUOTE_COMPUTED, QUOTE_CREATED, ORDER_CONFIRMED)
  - Include event correlation IDs (search_id, quote_id, dispatch_order_id)
  - Include **traceparent** and **trace_id** in all event logs
  - Include timing and status for event processing

- **Callback Delivery Logging**:
  - Log all callback delivery attempts for all ONDC callbacks:
    - Order callbacks: `/on_search`, `/on_init`, `/on_confirm`, `/on_status`, `/on_cancel`, `/on_update`, `/on_track`
    - IGM callbacks: `/on_issue`, `/on_issue_status`
  - Include timing and status
  - Track retry attempts and failures
  - Store callback URL, ONDC correlation IDs (`transaction_id`, ONDC `message_id`), and delivery status
  - **Note**: ONDC `message_id` is from request context, NOT Redis Stream message_id (which is only used for ACK)

- **IGM Logging**:
  - Log all IGM requests (`/issue`, `/issue_status`) and callbacks (`/on_issue`, `/on_issue_status`)
  - Include issue_id, transaction_id, message_id for correlation
  - Log Zendesk webhook events (ticket updates)
  - Track issue lifecycle (created, in-progress, resolved, closed)
  - Log financial resolution updates
  - Store full ONDC payload for callback reconstruction

- **Audit Trail**:
  - Tamper-resistant storage of request/response pairs in Postgres-E (`audit` schema)
  - Request hashing for integrity verification
  - Support dispute resolution with complete audit trail
  - Link requests to events via search_id, quote_id, dispatch_order_id
  - Include **trace_id** in audit logs for end-to-end correlation

- **Data Retention**:
  - Retain audit logs per regulatory requirements (minimum 7 years for financial transactions)
  - Support log archival and retrieval
  - Maintain log integrity and non-repudiation
  - Store in Postgres-E (`audit.request_response_logs`) with 7-year retention

- **Metrics & Monitoring**:
  
  **Version Scope**:
  - **v1 (Current)**: Business Metrics, Latency Metrics, Error Metrics, Service Health Metrics, Cache Metrics, Idempotency Metrics, ONDC-Specific Metrics, IGM Metrics, Database Metrics, SLO/SLI Metrics
  - **v2 (Future)**: Event Processing Metrics, Client Registry Metrics, Alerting Thresholds
  
  - **Business Metrics** (Counters) - **v1**:
    - `uois.requests.total` - Total incoming requests by endpoint (`/search`, `/init`, `/confirm`, `/status`, `/track`, `/cancel`, `/update`, `/rto`), client_id, protocol (ONDC/BECKN), status (success/error)
    - `uois.orders.created.total` - Orders created via `/confirm` (by client_id, status)
    - `uois.quotes.computed.total` - Quotes computed for `/search` (by client_id, serviceable/non-serviceable)
    - `uois.quotes.created.total` - Quotes created for `/init` (by client_id, status)
    - `uois.callbacks.delivered.total` - Callbacks delivered successfully (by callback type: `/on_search`, `/on_init`, `/on_confirm`, etc., client_id)
    - `uois.callbacks.failed.total` - Callback delivery failures (by callback type, error_code, client_id)
    - `uois.issues.created.total` - IGM issues created (by client_id, category, type)
    - `uois.issues.resolved.total` - IGM issues resolved (by client_id, resolution_time_bucket)
  
  - **Latency Metrics** (Histograms/Distributions) - **v1**:
    - `uois.request.duration` - Request processing time by endpoint, client_id, status (p50, p95, p99)
    - `uois.callback.delivery.duration` - Callback delivery time by callback type, client_id (p50, p95, p99)
    - `uois.event.processing.duration` - Event processing time by event type (`QUOTE_COMPUTED`, `QUOTE_CREATED`, `ORDER_CONFIRMED`, etc.)
    - `uois.db.query.duration` - Database query time by query type (audit_log_insert, client_lookup, order_lookup)
    - `uois.grpc.call.duration` - gRPC call duration to Order Service by method (`GetOrder`, `CancelOrder`, etc.)
    - `uois.auth.duration` - Authentication/authorization time (client lookup, credential validation)
  
  - **Error Metrics** (Counters) - **v1**:
    - `uois.errors.total` - Total errors by error_code (`65001`, `65002`, `65003`, etc.), endpoint, client_id
    - `uois.errors.by_category` - Errors by category (validation, authentication, dependency, internal)
    - `uois.timeouts.total` - Request timeouts by endpoint, dependency (Order Service, Quote Service, Location Service)
    - `uois.rate_limit.exceeded.total` - Rate limit violations by client_id, endpoint
    - `uois.callback.retries.total` - Callback retry attempts by callback type, attempt_number
  
  - **Service Health Metrics** (Gauges) - **v1**:
    - `uois.service.availability` - Service availability (1 = healthy, 0 = unhealthy)
    - `uois.dependencies.health` - Dependency health status (Order Service, Quote Service, Location Service, Admin Service, Postgres-E, Redis) - 1 = healthy, 0 = unhealthy
    - `uois.dependencies.latency` - Dependency latency by service (p95, p99)
    - `uois.circuit_breaker.state` - Circuit breaker state by dependency (closed/open/half-open)
    - `uois.db.connection.pool.active` - Active database connections
    - `uois.db.connection.pool.idle` - Idle database connections
    - `uois.redis.connection.pool.active` - Active Redis connections
  
  - **Cache Metrics** (Counters/Gauges) - **v1**:
    - `uois.cache.hits.total` - Cache hits by cache type (client_registry, order_reference, client_config)
    - `uois.cache.misses.total` - Cache misses by cache type
    - `uois.cache.hit_rate` - Cache hit rate by cache type (hits / (hits + misses))
    - `uois.cache.size` - Cache size by cache type (number of entries)
    - `uois.cache.evictions.total` - Cache evictions by cache type (LRU evictions)
  
  - **Idempotency Metrics** (Counters) - **v1**:
    - `uois.idempotency.duplicate_requests.total` - Duplicate requests detected by endpoint, idempotency_key_type
    - `uois.idempotency.replays.total` - Idempotent request replays (returned cached response)
  
  - **ONDC-Specific Metrics** (Counters) - **v1**:
    - `uois.ondc.signature.verifications.total` - ONDC signature verifications by status (success/failure), reason
    - `uois.ondc.signature.generation.total` - ONDC signature generations by status (success/failure)
    - `uois.ondc.registry.lookups.total` - ONDC network registry lookups by status (success/failure, cache_hit/cache_miss)
    - `uois.ondc.timestamp.validations.total` - Timestamp validations by status (valid/invalid/stale)
  
  - **IGM Metrics** (Counters/Gauges) - **v1**:
    - `uois.igm.zendesk.tickets.created.total` - Zendesk tickets created (by issue_type, priority)
    - `uois.igm.zendesk.tickets.updated.total` - Zendesk tickets updated (by status_change)
    - `uois.igm.zendesk.webhooks.received.total` - Zendesk webhooks received (by event_type)
    - `uois.igm.zendesk.sync.lag` - Time between Zendesk update and ONDC callback (seconds)
    - `uois.igm.issues.by_status` - Current issue count by status (OPEN, CLOSED, IN_PROGRESS)
    - `uois.igm.issues.resolution_time` - Issue resolution time by issue_type (histogram)
  
  - **Database Metrics** (Gauges) - **v1**:
    - `uois.db.audit_logs.written.total` - Audit log writes by status (success/failure)
    - `uois.db.audit_logs.size` - Total audit log entries (approximate count)
    - `uois.db.query.errors.total` - Database query errors by query_type, error_code
    - `uois.db.transaction.duration` - Database transaction duration by operation_type
  
  - **SLO/SLI Metrics** (Derived from above) - **v1**:
    - **Availability SLI**: `(total_requests - errors_5xx) / total_requests` (target: 99.9%)
    - **Latency SLI**: `p95_latency` by endpoint (targets: `/search` < 500ms, `/confirm` < 1s, `/status` < 200ms, callbacks < 2s)
    - **Error Rate SLI**: `errors_total / total_requests` (target: < 0.1%)
    - **Callback Success Rate SLI**: `callbacks_delivered / (callbacks_delivered + callbacks_failed)` (target: > 99%)
    - **Event Processing Lag SLI**: `event_processing_lag` (target: < 5 seconds p95) - **v2**
  
  - **Metric Labels/Tags**:
    - **Required labels**: `service=uois-gateway`, `environment` (sandbox/production), `instance_id`
    - **Optional labels**: `client_id`, `endpoint`, `protocol` (ONDC/BECKN), `error_code`, `event_type`, `callback_type`, `dependency_name`
    - **Cardinality consideration**: Limit high-cardinality labels (e.g., `client_id` only for business metrics, not technical metrics)
  
  - **Metric Export**:
    - Export metrics in Prometheus format (OpenMetrics)
    - Expose metrics endpoint: `/metrics` (standard Prometheus scrape endpoint)
    - Push metrics to CloudWatch Metrics (AWS) or equivalent
    - Support metric aggregation and retention policies (1-minute granularity for 7 days, 5-minute for 30 days, 1-hour for 1 year)
  
  - **Event Processing Metrics** (Counters/Gauges) - **v2**:
    - `uois.events.published.total` - Events published by event type (`SEARCH_REQUESTED`, `INIT_REQUESTED`, `CONFIRM_REQUESTED`), status (success/failure)
    - `uois.events.consumed.total` - Events consumed by event type (`QUOTE_COMPUTED`, `QUOTE_CREATED`, `ORDER_CONFIRMED`), status (success/failure)
    - `uois.events.processing.lag` - Event processing lag by stream (time between event published and consumed)
    - `uois.events.consumer_group.lag` - Redis Stream consumer group lag by stream (`quote:computed`, `stream.uois.quote_created`, etc.)
    - `uois.events.ack.total` - Events ACKed by consumer group, stream
    - `uois.events.failed.total` - Event processing failures by event type, error_code
    - `uois.events.publish.rate` - Event publish rate (events/second) by event type
    - `uois.events.consume.rate` - Event consume rate (events/second) by event type
  
  - **Client Registry Metrics** (Counters/Gauges) - **v2**:
    - `uois.client.registry.lookups.total` - Client registry lookups by source (cache/db/fallback)
    - `uois.client.registry.sync.total` - Client registry sync events processed (by event_type: `client.created`, `client.updated`, `client.suspended`, `client.revoked`, `client.api_key_rotated`)
    - `uois.client.registry.size` - Total number of active clients in registry
    - `uois.client.registry.sync.lag` - Time between Admin Service event and local registry update (seconds)
  
  - **Alerting Thresholds** (Derived from SLOs) - **v2**:
    - **Availability**: Alert if availability < 99.9% over 5-minute window
    - **Latency**: Alert if p95 latency exceeds targets (`/search` > 500ms, `/confirm` > 1s, `/status` > 200ms) over 5-minute window
    - **Error Rate**: Alert if error rate > 1% over 5-minute window
    - **Dependency Health**: Alert if any dependency (Order Service, Quote Service, Postgres-E, Redis) is unhealthy
    - **Callback Failure Rate**: Alert if callback failure rate > 5% over 5-minute window
    - **Event Processing Lag**: Alert if event processing lag > 30 seconds (p95) over 5-minute window
    - **Database Connection Pool**: Alert if connection pool utilization > 80%
    - **Circuit Breaker**: Alert if circuit breaker opens (dependency failure)
    - **Client Registry Sync Lag**: Alert if sync lag > 60 seconds over 5-minute window

---

## 12. Non-Functional Requirements

### 12.1 Performance

**Latency Requirements**:
- Quote requests: < 500ms (p95)
- Order creation: < 1s (p95)
- Status queries: < 200ms (p95)
- Webhook relay: < 2s (p95)

**Throughput**:
- Support minimum 1000 requests/second
- Scale horizontally for peak loads

### 12.2 Availability

**SLO Requirements**:
- Availability: 99.9% uptime
- Graceful degradation when Order Service or Quote Service is unavailable
- Circuit breaker pattern for external service calls

### 12.3 Reliability

**Error Handling**:
- Handle Order Service and Quote Service timeouts gracefully
- Retry transient failures automatically
- Dead Letter Queue for persistent failures
- Alert on service degradation

**Idempotency**:
- All mutation operations must be idempotent
- Support request replay without side effects

**Configuration Validation**:
- **Config.Validate() Requirements**: Application MUST validate all critical configuration at startup and fail early if any required configuration is missing or invalid
- **Required Validations**:
  - Postgres-E connection string and database accessibility
  - Redis connection and cluster accessibility
  - Order Service gRPC endpoint (`ORDER_SERVICE_GRPC_*` configuration)
  - Admin Service gRPC endpoint (`ADMIN_SERVICE_GRPC_*` configuration)
  - ONDC key paths and signing configuration
  - All mandatory environment variables must be present and valid
- **Failure Behavior**: Application MUST NOT start if any critical configuration validation fails
- **Error Reporting**: Validation failures MUST include clear error messages indicating which configuration is missing or invalid

---

## 13. Compliance & Data Retention

### 13.1 Audit Log Retention

**Functional Requirements**:
- Retain all request/response logs for minimum 7 years
- Support log archival and retrieval
- Maintain log integrity (hashing, tamper-resistant storage)
- Enable dispute resolution with complete audit trail

### 13.2 Data Privacy

**Functional Requirements**:
- Sanitize sensitive data in logs (PII, payment details)
- Support data retention policies
- Comply with data protection regulations

### 13.3 ONDC Compliance

**Functional Requirements**:
- Comply with ONDC network security requirements
- Maintain ONDC protocol version compatibility
- Support ONDC network registry integration
- Follow ONDC dispute resolution guidelines

---

## 14. Out of Scope (v0)

The following are explicitly out of scope for v0:

- **Business Logic**: Pricing, capacity management, routing decisions (Quote Service, Location Service, DroneAI own)
- **Order Lifecycle**: Order state management, fulfillment orchestration (Order Service owns)
- **Grievance Resolution**: Issue resolution, customer support (External Helpdesk owns)
- **Client Configuration Management**: Client onboarding, configuration updates (Admin Service owns)
- **Payment Processing**: Payment gateway integration (Payment Service owns)
- **Event Stream Management**: Event stream infrastructure and routing (infrastructure layer owns)

---

## 15. Data Contracts (Schemas)

### 15.1 Events Published by UOIS Gateway

| Event Type | Stream | Consumer(s) | Purpose | Contract |
|-----------|-------|------------|---------|----------|
| SEARCH_REQUESTED | `stream.location.search` | **Location Service** | Trigger serviceability computation for `/search` flow | [Contract](/docs/04_DispatchContracts/06_location_service/03_consumed_events/01_search-requested) |
| INIT_REQUESTED | `stream.uois.init_requested` | **Order Service** | Trigger quote validation and creation for `/init` flow | `Order-Service-Dispatch/contracts/events/consumed/uois/init_requested.json` |
| CONFIRM_REQUESTED | `stream.uois.confirm_requested` | **Order Service** | Trigger order creation and rider assignment for `/confirm` flow | `Order-Service-Dispatch/contracts/events/consumed/uois/confirm_requested.json` |

**Schema Format**: JSON Schema 2020-12 (events)
**Versioning**: Semantic Versioning (v1.0.0)
**Transport**: Redis Streams (with consumer groups and ACK)

**Key Integration Notes:**
- **Location Service** consumes `SEARCH_REQUESTED` from stream `stream.location.search` to provide rider availability, distances, and ETAs
- **Quote Service** consumes `SERVICEABILITY_FOUND` from stream `location:serviceability:found` and publishes `QUOTE_COMPUTED` to stream `quote:computed` with all fields included (pass-through of serviceability fields)
- UOIS Gateway publishes events **after** validating client requests (authentication, schema validation)
- Events include `search_id` as correlation key for tracking across services
- All events include `traceparent` (W3C format) for distributed tracing
- All events include `event_id` (UUID v4) for event-level deduplication (v2.0.0)

### 15.2 Events Consumed by UOIS Gateway

| Event Type | Stream | Producer | Purpose | Contract |
|-----------|-------|----------|---------|----------|
| QUOTE_COMPUTED | `quote:computed` | Quote Service | Receive quote for `/search` response to client | `Order-Service-Dispatch/contracts/events/consumed/quote/quote_computed.json` |
| QUOTE_CREATED | `stream.uois.quote_created` | Order Service | Receive validated quote for `/init` response to client | `Order-Service-Dispatch/contracts/events/produced/confirmation/order_confirmation_accepted.json` (Note: QUOTE_CREATED schema TBD) |
| QUOTE_INVALIDATED | `stream.uois.quote_invalidated` | Order Service | Receive quote validation failure for `/init` response to client | TBD |
| ORDER_CONFIRMED | `stream.uois.order_confirmed` | Order Service | Receive order confirmation for `/confirm` response to client | `Order-Service-Dispatch/contracts/events/produced/confirmation/order_confirmed.json` |
| ORDER_CONFIRM_FAILED | `stream.uois.order_confirm_failed` | Order Service | Receive order confirmation failure for `/confirm` response to client | `Order-Service-Dispatch/contracts/events/produced/confirmation/order_confirm_failed.json` |

---

## Related Documents

For detailed technical specifications, refer to:
- **API Contract**: Endpoint specifications, request/response schemas, error codes
- **ONDC Integration Guide**: ONDC protocol details, signing requirements, network registry
- **Security Architecture**: Authentication, authorization, key management
- **Architecture Document**: System design, database schema, caching strategy
- **Location Service Contracts**: `/docs/04_DispatchContracts/06_location_service/` - Serviceability events and APIs