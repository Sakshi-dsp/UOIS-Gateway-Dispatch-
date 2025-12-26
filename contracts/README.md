# Order Service - API & Event Contracts

## Overview

This directory contains all **data contracts** (API schemas and event schemas) for the **Order Service**.

Order Service is the **order lifecycle orchestrator** responsible for:
- Quote validation and persistence in `/init` flow
- Order creation and state management in `/confirm` flow
- Order lifecycle tracking through 22 canonical states
- Event-driven coordination with Location Service, Quote Service, and downstream consumers
- **Rider App payload management** following [Rider Payload Contract v1.0](../../docs/production-docs/RIDER_PAYLOAD_SPEC_V1.0.md)
- Single source of truth for order state in Postgres-C (`orders` schema)

**Database:** Postgres-C (`orders` schema) - canonical order state, quotes, and order events

---

## Rider Payload Contract v1.0 Implementation

Order Service event contracts have been updated to implement the Rider Payload Contract v1.0 specification:

### Payload Categories
- **FULL PAYLOADS** (`order_details` REQUIRED): `order.accepted`, `order.rto_initiated`
- **PARTIAL PAYLOADS** (`order_details` RECOMMENDED): `order.assigned`, `order.en_route_to_origin`, `order.picked_up`, `order.en_route_to_destination`
- **SIGNAL-ONLY PAYLOADS** (`order_details` EXCLUDED): `order.delivery_attempted`, `order.payment_collected`, `order.delivered`, `order.completed`, `order.rto_delivered`

### Key Changes
- Added `state` field (canonical state machine value)
- Added `order_details` structure for FULL/PARTIAL payloads
- Created shared `order_details.json` schema for consistency
- Updated event descriptions with payload category information
- Maintained backward compatibility for non-Rider App consumers

See [Rider Payload Contract v1.0](../../docs/production-docs/RIDER_PAYLOAD_SPEC_V1.0.md) for complete specification.

---

## Contract Organization

### 1. Events Consumed (`events/consumed/`)

Events consumed by Order Service from upstream producers.

| Event | Producer | Purpose | Status |
|-------|----------|---------|--------|
| `INIT_REQUESTED` | UOIS Gateway | Triggers `/init` flow quote validation | ✅ Defined |
| `CONFIRM_REQUESTED` | UOIS Gateway | Triggers `/confirm` flow order creation | ✅ Defined |
| `REVALIDATION_QUOTE_COMPUTED` | Quote Service | Updated pricing for revalidation scenarios | ✅ Exists |
| `REVALIDATION_FOUND` | Location Service | Updated ETAs/serviceability for revalidation | ✅ Exists |
| `RIDER_ASSIGNED` | DroneAI | Rider assignment success notification | ✅ Defined |
| `RIDER_ASSIGNMENT_FAILED` | DroneAI | Rider assignment failure notification | ✅ Defined |
| `soft_arrived` | Location Service | Auto-transition to SOFT_ARRIVED states | ✅ Defined |
| `geofence.entered` | Location Service | Zone-based notifications and transitions | ✅ Defined |

### 2. Events Produced (`events/produced/`)

Events published by Order Service for downstream consumption.

| Event | Consumer(s) | Purpose | Status |
|-------|-------------|---------|--------|
| `REVALIDATION_REQUESTED` | Location Service, Quote Service | Triggers coordinated revalidation workflow | ❌ Missing |
| `ORDER_QUOTE_ATTACHED` | UOIS Gateway | Quote validation success (renamed to avoid Quote Service conflict) | ❌ Missing |
| `ORDER_QUOTE_INVALIDATED` | UOIS Gateway | Quote validation failure (renamed to avoid Quote Service conflict) | ❌ Missing |
| `ORDER_CONFIRMATION_ACCEPTED` | DroneAI | Order created, ready for rider assignment | ❌ Missing |
| `ORDER_CONFIRMATION_REJECTED` | UOIS Gateway | Order creation failed due to quote issues | ❌ Missing |
| `ORDER_CONFIRMED` | UOIS Gateway | Rider assignment success notification | ❌ Missing |
| `ORDER_CONFIRM_FAILED` | UOIS Gateway | Order created but rider assignment failed | ❌ Missing |
| `order.lifecycle` | Location Service, Invoicing Service | Consolidated order state transition events | ❌ Missing |
| `order.assigned` | WebSocket Gateway, Location Service | Rider assigned notification (PARTIAL payload) | ✅ Updated v1.0 |
| `order.accepted` | WebSocket Gateway | Rider accepted assignment (FULL payload) | ✅ Updated v1.0 |
| `order.picked_up` | WebSocket Gateway, Location Service | Package collected from origin (PARTIAL payload) | ✅ Updated v1.0 |
| `order.arrived_at_destination` | WebSocket Gateway | Rider arrived at destination | ❌ Needs Update |
| `order.delivered` | Location Service, Invoicing Service, WebSocket Gateway | Order delivered (SIGNAL-ONLY for Rider App, FULL for others) | ✅ Updated v1.0 |
| `order.cancelled` | Location Service, WebSocket Gateway | Order cancelled | ✅ Defined |
| `order.rto_initiated` | Location Service, WebSocket Gateway | RTO started | ✅ Defined |
| `order.rto_arrived_at_location` | WebSocket Gateway | Rider arrived at RTO location | ✅ Defined |
| `order.rto_delivered` | Location Service, Invoicing Service | RTO completed | ✅ Defined |
| `order.snapshot` | WebSocket Gateway, Admin Dashboard | Canonical order state for hydration/replay | ❌ Missing |

### 3. APIs (`apis/`)

Currently no gRPC APIs defined. Order Service is event-first architecture.

---

## Schema Standards

### Event Schemas
- **Format**: JSON Schema 2020-12 (primary format for Redis Streams)
- **Transport**: Redis Streams (with consumer groups and ACK)
- **Envelope**: Standard event envelope with `event_type`, `event_id`, `timestamp`, `traceparent`
- **Versioning**: Semantic Versioning (v1.0.0)
  - **MAJOR**: Breaking changes (required field removal, type change)
  - **MINOR**: Backward-compatible additions (optional fields)
  - **PATCH**: Documentation clarifications only

### Required Event Fields

All Order Service events **MUST** include:
- `event_id`: UUID v4 for event-level deduplication
- `event_version`: Integer starting at 1 for contract evolution
- `traceparent`: W3C Trace Context for distributed tracing
- `timestamp`: ISO 8601 UTC timestamp

### Field Naming Conventions
- `order_id`: 
  - **Rider-facing events**: Include both `dispatch_order_id` (required) and `client_order_id` (required)
  - **System-facing events**: Use `dispatch_order_id` only (human-readable format: `{CLIENT_CODE_PREFIX}{INCREMENTAL_NUMBER}`)
  - `dispatch_order_id`: Dispatch-generated order identifier (canonical internal ID)
  - `client_order_id`: Client-provided order identifier (what CLIENT sends TO Dispatch during order creation)
- `quote_id`: UUID reference to validated quote
- `rider_id`: Assigned rider identifier
- `client_id`: Client identifier for billing/context

---

## Ownership Model

### Order Service Owned Contracts

**Location:** `contracts/events/produced/`

Order Service **owns and defines** all outbound event schemas:
- `revalidation_requested.json` - Coordination trigger for revalidation
- `order.lifecycle.json` - Consolidated order state transitions
- `order.snapshot.json` - Order state projections
- All confirmation and quote attachment events

**Ownership Rules:**
- Order Service defines initial schemas (v1.0.0)
- Breaking changes require Order Service team approval
- Consumers provide feedback but do not own the contracts

### Consumed Contracts (Not Owned by Order Service)

**Location:** `contracts/events/consumed/`

Order Service **consumes** but does **not own** these contracts:
- UOIS Gateway events: `INIT_REQUESTED`, `CONFIRM_REQUESTED`
- DroneAI events: `RIDER_ASSIGNED`, `RIDER_ASSIGNMENT_FAILED`
- Location Service events: `REVALIDATION_FOUND`, `soft_arrived`, `geofence.entered`
- Quote Service events: `REVALIDATION_QUOTE_COMPUTED`

**Consumption Rules:**
- Order Service validates against upstream schemas
- Schema changes require upstream team coordination
- Order Service maintains copies for validation/testing

### Referenced Contracts (External Ownership)

Order Service **references** but does **not own**:
- Location Service: `SERVICEABILITY_FOUND`, `QUOTE_COMPUTED`
- WebSocket: Event envelope schemas
- Admin Service: Pricing configurations

---

## Structure

```
contracts/
├── events/
│   ├── consumed/           # Events Order Service subscribes to
│   │   ├── OWNERS          # Upstream team ownership
│   │   ├── init_requested.json
│   │   ├── confirm_requested.json
│   │   ├── rider_assigned.json
│   │   ├── rider_assignment_failed.json
│   │   ├── soft_arrived.json
│   │   └── geofence_entered.json
│   └── produced/           # Events Order Service publishes
│       ├── OWNERS          # Order Service team ownership
│       ├── revalidation_requested.json
│       ├── order_quote_attached.json
│       ├── order_quote_invalidated.json
│       ├── order_confirmation_accepted.json
│       ├── order_confirmation_rejected.json
│       ├── order_confirmed.json
│       ├── order_confirm_failed.json
│       ├── order.lifecycle.json
│       └── order.snapshot.json
└── apis/                   # Future gRPC contracts (if needed)
    └── README.md           # Placeholder
```

---

## Critical Requirements

### Dual Event Publishing Strategy

**Order Service publishes two types of order events:**

1. **Consolidated `order.lifecycle` events** for analytics and state synchronization:
```json
{
  "event_type": "order.lifecycle",
  "dispatch_order_id": "ABC0000001",
  "client_order_id": "SWG-ORD-982734",
  "state": "DELIVERED",
  "previous_state": "EN_ROUTE_TO_DESTINATION",
  "reason": "rider_marked_delivered",
  "metadata": { /* state-specific fields */ }
}
```

2. **Separate rider-facing events** for real-time WebSocket delivery:
- `order.assigned` → WebSocket Gateway → Rider App
- `order.accepted` → WebSocket Gateway → Rider App
- `order.picked_up` → WebSocket Gateway → Rider App
- `order.arrived_at_destination` → WebSocket Gateway → Rider App
- `order.delivered` → WebSocket Gateway → Rider App
- `order.cancelled` → WebSocket Gateway → Rider App
- `order.rto_initiated` → WebSocket Gateway → Rider App
- `order.rto_arrived_at_location` → WebSocket Gateway → Rider App
- `order.rto_delivered` → WebSocket Gateway → Rider App

**WebSocket Gateway Responsibility:**
- Pure transport layer: forwards event payloads as-is (both IDs included)
- No ID interpretation, filtering, or mapping
- Forwards complete event payload to Rider App

**Location Service and Invoicing Service consume `order.lifecycle` events.**
**WebSocket Gateway consumes rider-facing events for real-time delivery to Rider Apps.**

### Order Snapshot Schema

**Required for state hydration and replay:**
```json
{
  "dispatch_order_id": "ABC0000001",
  "client_order_id": "SWG-ORD-982734",
  "state": "DELIVERED",
  "quote_snapshot": { /* validated quote data */ },
  "rider_id": "rider_123",
  "timestamps": { /* all state transition timestamps */ }
}
```

### Naming Conflict Avoidance

**Quote-related events use `ORDER_QUOTE_*` prefix** to avoid confusion with Quote Service `QUOTE_*` events:
- `ORDER_QUOTE_ATTACHED` (not `QUOTE_CREATED`)
- `ORDER_QUOTE_INVALIDATED` (not `QUOTE_INVALIDATED`)

### State Machine Integration

All events must align with **22 canonical order states**:
- States defined in OrderService_FR.md
- Transitions validated before publishing
- Event publishing occurs **after** database commit

---

## Operational Requirements

1. **Idempotency:** Order Service ensures idempotent processing using `event_id` deduplication
2. **Timestamps:** All timestamps in UTC ISO8601 only
3. **Distributed Tracing:** All events include `traceparent` for end-to-end observability
4. **Event Versioning:** All events include `event_version` starting at 1
5. **State Consistency:** Events published only after canonical state commit in Postgres-C
6. **Consumer Groups:** Per-stream consumer groups to avoid backpressure coupling

### Stream Configuration

| Stream | Consumer Group | Purpose |
|--------|----------------|---------|
| `stream.uois.init_requested` | `order-init-cg` | Quote validation triggers |
| `stream.uois.confirm_requested` | `order-confirm-cg` | Order creation triggers |
| `quote:revalidation:computed` | `order-revalidation-quote-cg` | Revalidation pricing |
| `location:revalidation:found` | `order-revalidation-location-cg` | Revalidation ETAs |
| `stream.droneai.rider_assigned` | `order-rider-assigned-cg` | Assignment success |
| `stream.location.soft_arrived` | `order-soft-arrived-cg` | Auto-transitions |
| `stream.location.geofence.entered` | `order-geofence-cg` | Zone transitions |

### Publishing Streams

| Stream | Event Types | Consumers |
|--------|-------------|-----------|
| `stream.location.revalidation` | `REVALIDATION_REQUESTED` | Location + Quote Services |
| `stream.uois.quote_events` | `ORDER_QUOTE_*` | UOIS Gateway |
| `stream.droneai.confirmation` | `ORDER_CONFIRMATION_*` | DroneAI |
| `stream.order.lifecycle` | `order.lifecycle` | Location Service, Invoicing Service |
| `stream.ws.push.{rider_id}` | `order.*` (rider-facing events) | WebSocket Gateway |
| `stream.order.snapshots` | `order.snapshot` | WebSocket, Admin |

---

## Consumer Responsibilities

### For Event Consumers
- Register consumer group name with Order Service team
- Implement idempotent processing using `event_id`
- ACK events only after successful processing
- Handle schema version mismatches gracefully
- Process events in order within partition (Redis Stream ordering guarantee)
- Store processed `event_id` values (TTL: 24-48 hours) for at-least-once deduplication

### For Event Producers
- Validate events against schemas before publishing
- Include all required fields
- Propagate `traceparent` unchanged
- Publish events only after state transitions are committed

---

## Integration Dependencies

### Upstream Services (Order Service Consumes From)

| Service | Required Contracts | Status |
|---------|-------------------|--------|
| UOIS Gateway | `INIT_REQUESTED`, `CONFIRM_REQUESTED` | ✅ Defined |
| Quote Service | `REVALIDATION_QUOTE_COMPUTED` | ✅ Exists |
| Location Service | `REVALIDATION_FOUND`, `soft_arrived`, `geofence.entered` | ✅ All Available |
| DroneAI | `RIDER_ASSIGNED`, `RIDER_ASSIGNMENT_FAILED` | ✅ Defined |

### Downstream Services (Order Service Publishes To)

| Service | Consumes Events | Status |
|---------|-----------------|--------|
| UOIS Gateway | `ORDER_QUOTE_*`, `ORDER_CONFIRMATION_*`, `ORDER_CONFIRMED`, `ORDER_CONFIRM_FAILED` | ❌ Missing |
| DroneAI | `ORDER_CONFIRMATION_ACCEPTED` | ❌ Missing |
| Location Service | `REVALIDATION_REQUESTED`, `order.lifecycle` | ❌ Missing |
| WebSocket Gateway | `order.assigned`, `order.accepted`, `order.picked_up`, `order.arrived_at_destination`, `order.delivered`, `order.cancelled`, `order.rto_initiated`, `order.rto_arrived_at_location`, `order.rto_delivered` | ✅ Complete (9/9 defined) |
| Invoicing Service | `order.lifecycle` | ❌ Missing |

---

## Implementation Blocking Issues

### Critical Gaps (Must Resolve Before Implementation)

1. **8 Downstream Event Schemas Missing** - Cannot coordinate with downstream services
2. **Order Lifecycle Consolidation** - Requires domain-wide agreement on single schema approach
3. **Cross-Service Coordination** - Requires agreement from 5+ service teams

### Recommended Implementation Sequence

1. **Phase 1: Upstream Dependencies**
   - Define UOIS Gateway events (`INIT_REQUESTED`, `CONFIRM_REQUESTED`)
   - Define DroneAI events (`RIDER_ASSIGNED`, `RIDER_ASSIGNMENT_FAILED`)
   - Complete Location Service events (`soft_arrived`, `geofence.entered`)

2. **Phase 2: Order Service Contracts**
   - Define consolidated `order.lifecycle` schema
   - Define `order.snapshot` schema
   - Define revalidation and confirmation events

3. **Phase 3: Integration Testing**
   - Contract validation across all services
   - Event flow testing with consumer groups
   - Idempotency and error scenario testing

---

## Related Documentation

- **Functional Requirements**: `OrderService_FR.md`
- **Non-Functional Requirements**: `OrderService_NFR.md`
- **Integration Boundary**: `INTEGRATION_BOUNDARY.md`
- **Location Service Contracts**: `@Location-Service-Dispatch-/Contracts`
- **Quote Service Contracts**: `@Quote-Service-Dispatch/Quote_contracts`
- **Dispatch Architecture**: `DispatchArchitecture/Plan_v2.md`

---

## Change Log

| Version | Date | Changes | Author |
|---------|------|---------|--------|
| v1.0.0 | 2025-12-16 | Initial contract requirements analysis and gaps identification | System |

---

## Support

For questions about Order Service contracts:
- **Schema Issues**: Open issue in Order Service repository
- **Integration Help**: Contact Order Service team
- **Breaking Changes**: Coordinate with all consuming services

## Contact

- **Order Service Team:** order-service-team@dispatch.arch
- **Location Service Team:** location-service-team@dispatch.arch
- **Quote Service Team:** quote-service-team@dispatch.arch
- **UOIS Gateway Team:** uois-team@dispatch.arch
- **DroneAI Team:** droneai-team@dispatch.arch

---

**Status:** ⚠️ **PARTIALLY READY** - Consumed event contracts defined, but downstream contracts and cross-service coordination still required.
