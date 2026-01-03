# Local Integration Plan: Order + Quote + Location + UOIS Gateway Services

**Purpose:** Executable plan to run Order Service, Quote Service, Location Service, and UOIS Gateway together locally using Docker, Redis Streams, PostgreSQL, and **per-service observability stacks** (Option B).

**Architecture Choice:** Option B - Per-Service Observability  
Each service owns its complete observability stack (Prometheus + Grafana). This mirrors production topology with per-team ownership and namespace-level isolation.

**Last Updated:** Updated to include UOIS Gateway integration

---

## 📌 Component Version Pinning

**Recommended versions for consistent local development:**

| Component | Version | Notes |
|-----------|---------|-------|
| **Redis** | 7.x (7-alpine) | Required for Streams support (XGROUP command). Minimum 5.0+ |
| **PostgreSQL** | 15 | Order DB, Location DB (PostGIS 15-3.4-alpine), UOIS Gateway Audit DB |
| **Go** | 1.21+ | Required for all services (Order, Quote, Location, UOIS Gateway) |
| **OSRM** | latest | Routing engine (osrm/osrm-backend:latest) |


---

## 1️⃣ Local Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                         Docker Network: dispatch-net                │
└─────────────────────────────────────────────────────────────────────┘

┌─────────────────┐         ┌───────────────-──┐        ┌─────────────────┐          ┌──────────────────┐
│   Order Service │         │  Quote Service   │        │ Location Service │         │  UOIS Gateway    │
│   Port: 8082    │         │   Port: 8080     │        │   Port: 8081     │         │   Port: 8083     │
│   DB: order_db  │         │   (no DB)        │        │   DB: location_db│         │DB: postgres_audit│
└────────┬────────┘         └────────┬─────-───┘        └────────┬─────-───┘         └─────────┬────────┘
         │                           │                           │                             │
         │                           │                           │                             │
         └───────────────────────────┴───────────────────────────┴─────────────────────────────┘
                                     │
                         ┌────────────▼────────────┐
                         │   Redis Streams (6380)  │
                         │   - Shared Event Bus    │
                         │ ⚠️ Requires Redis 5.0+ │
                         │     for Streams support │
                         └─────────────────────────┘

┌─────────────────┐         ┌─────────────────┐         ┌─────────────────┐         ┌─────────────────┐
│   PostgreSQL    │         │   PostgreSQL    │         │   PostgreSQL-E  │         │     OSRM        │
│   order_db      │         │   location_db   │         │   postgres_audit│         │   Port: 5000    │
│   Port: 5433    │         │   Port: 5435    │         │   Port: 5436    │         │   (Routing)     │
└─────────────────┘         └─────────────────┘         └─────────────────┘         └─────────────────┘

┌────────────────────────────────────────────────────────────────────────────────────────────────┐
│              Per-Service Observability Stacks (Option B)                                       │
│                                                                                                │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐    ┌──────────────┐                  │
│  │ Quote Obs    │    │ Location Obs │    │ Order Obs    │    │ UOIS Obs     │                  │
│  │ Prom: 9090   │    │ Prom: 9091   │    │ Prom: 9092   │    │ Prom: 9093   │                  │
│  │ Graf: 3000   │    │ Graf: 3001   │    │ Graf: 3002   │    │ Graf: 3003   │                  │
│  └──────────────┘    └──────────────┘    └──────────────┘    └──────────────┘                  │
│       ▲                    ▲                    ▲                    ▲                         │
│       │                    │                    │                    │                         │
│       └────────────────────┴────────────────────┴────────────────────┘                         │
│              Scrapes via host.docker.internal                                                  │
│              - Quote Service:8080/metrics                                                      │
│              - Location Service:8081/metrics                                                   │
│              - Order Service:8082/metrics                                                      │
│              - UOIS Gateway:8083/metrics                                                       │
└────────────────────────────────────────────────────────────────────────────────────────────────┘

┌───────────────────────────────────────────────────────────────────────────────────────┐
│                    Mock Services                                                      │
│  Mock UOIS (Script):                                                                  │
│  - Publishes INIT_REQUESTED to stream.uois.init_requested                             │
│  - Publishes CONFIRM_REQUESTED to stream.uois.confirm_requested                       │
│  - Consumes QUOTE_COMPUTED from quote:computed                                        │
│  - Consumes ORDER_CONFIRMED from stream.uois.order_confirmed                          │
│                                                                                       │
│  Mock DroneAI (Docker):                                                               │
│  - Container: dispatch-droneai-mock                                                   │
│  - Consumes ORDER_CONFIRMATION_ACCEPTED from stream.droneai.confirmation_accepted     │
│  - Publishes RIDER_ASSIGNED to stream.droneai.order.assigned                          │
│  - Publishes RIDER_ASSIGNMENT_FAILED to stream.droneai.order.assign_failed            │
└───────────────────────────────────────────────────────────────────────────────────────┘
```

---

### 1.1 Local Execution Model

**Important:** This integration plan uses a **hybrid execution model**:

- **Application Services:** Run on **host** (`go run`) for faster iteration and debugging
- **Infrastructure:** Runs in **Docker** (Redis, PostgreSQL, OSRM, Admin Mock, DroneAI Mock)
- **Observability:** Each service runs its own Prometheus + Grafana stack in Docker
- **Metrics Scraping:** Prometheus uses `host.docker.internal` to scrape metrics from host-running services

**Why this model:**
- Faster development iteration (no container rebuilds for code changes)
- Production-like infrastructure isolation
- Clear separation between application code and infrastructure

**Note:** When dockerizing services later or running in CI, update Prometheus scrape targets to use container names instead of `host.docker.internal`.

---

## 2️⃣ Docker Compose Design

### 2.1 Complete docker-compose.yml (Shared Infrastructure Only)

**Location:** `Order-Service-Dispatch/docker-compose.integration.yml`

**Note:** This docker-compose file provides **shared infrastructure** (Redis, Postgres, OSRM, Admin Mock, DroneAI Mock).  
**Observability stacks are NOT included** - each service runs its own via `docker-compose.observability.yml`.

```yaml
version: '3.8'

services:
  # ============================================================================
  # Shared Infrastructure
  # ============================================================================
  
  redis:
    image: redis:7-alpine
    container_name: dispatch-redis
    ports:
      - "6380:6379"
    volumes:
      - redis_data:/data
    command: redis-server --appendonly yes
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 3s
      retries: 5
    networks:
      - dispatch-net
    # ⚠️ IMPORTANT: Redis 5.0+ required for Streams support (XGROUP command)
    # UOIS Gateway, Order Service, Quote Service, and Location Service all use consumer groups
    # If you see "ERR unknown command 'xgroup'" error, upgrade Redis to 5.0+ (redis:7-alpine is recommended)

  # ============================================================================
  # Order Service Database
  # ============================================================================
  # NOTE: Using existing order-service-postgres-c container (port 5433)
  # This service is commented out to avoid conflicts
  # 
  # order-postgres:
  #   image: postgres:15-alpine
  #   container_name: dispatch-order-postgres
  #   environment:
  #     POSTGRES_USER: order_service
  #     POSTGRES_PASSWORD: password
  #     POSTGRES_DB: order_service
  #   ports:
  #     - "5433:5432"
  #   volumes:
  #     - order_postgres_data:/var/lib/postgresql/data
  #   healthcheck:
  #     test: ["CMD-SHELL", "pg_isready -U order_service"]
  #     interval: 5s
  #     timeout: 3s
  #     retries: 5
  #   networks:
  #     - dispatch-net

  # ============================================================================
  # Location Service Database
  # ============================================================================
  
  location-postgres:
    image: postgis/postgis:15-3.4-alpine
    container_name: dispatch-location-postgres
    environment:
      POSTGRES_USER: location_service
      POSTGRES_PASSWORD: password
      POSTGRES_DB: location_service
    ports:
      - "5435:5432"
    volumes:
      - location_postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U location_service"]
      interval: 5s
      timeout: 3s
      retries: 5
    networks:
      - dispatch-net

  # ============================================================================
  # OSRM (Required by Location Service)
  # ============================================================================
  
  osrm:
    image: osrm/osrm-backend:latest
    container_name: dispatch-osrm
    ports:
      - "5000:5000"
    volumes:
      # Mount OSRM data directory from Location Service repo
      # Note: Update path if Location-Service-Dispatch is in different location
      - ../Location-Service-Dispatch/osrm/data:/data
    command: >
      sh -c "
        if [ ! -f /data/maharashtra-latest.osrm ]; then
          echo 'Preparing OSRM data for Maharashtra...';
          if [ ! -f /data/maharashtra-latest.osm.pbf ]; then
            echo 'ERROR: OSM PBF file not found. Please download maharashtra-latest.osm.pbf to Location-Service-Dispatch/osrm/data/';
            echo 'Download from: https://download.geofabrik.de/asia/india/maharashtra-latest.osm.pbf';
            exit 1;
          fi;
          osrm-extract -p /opt/car.lua /data/maharashtra-latest.osm.pbf &&
          osrm-partition /data/maharashtra-latest.osrm &&
          osrm-customize /data/maharashtra-latest.osrm;
        else
          echo 'OSRM data already prepared. Skipping extract.';
        fi &&
        osrm-routed --algorithm mld /data/maharashtra-latest.osrm --port 5000
      "
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:5000/route/v1/driving/72.8777,19.0760;72.8877,19.0860?overview=false"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 120s
    networks:
      - dispatch-net

  # ============================================================================
  # Admin Service Mock (Required by Quote Service)
  # ============================================================================
  
  admin-service-mock:
    image: moul/grpcbin:latest
    container_name: dispatch-admin-mock
    ports:
      - "50051:9000"
    networks:
      - dispatch-net
    # Note: This is a generic gRPC mock. For production, use actual Admin Service.
    # Future improvement: Replace with a thin Go mock implementing GetPricingConfig
    # to enable versioned contracts, failure injection, and latency simulation.

  # ============================================================================
  # DroneAI Mock Service (Required by Order Service)
  # ============================================================================
  
  droneai-mock:
    build:
      context: .
      dockerfile: scripts/Dockerfile.mock-droneai
    container_name: dispatch-droneai-mock
    environment:
      REDIS_HOST: redis
      REDIS_PORT: "6379"
      CONSUMER_GROUP: droneai-mock-group
      CONSUMER_NAME: droneai-mock-1
    depends_on:
      - redis
    restart: unless-stopped
    networks:
      - dispatch-net
    # Note: This mock service consumes ORDER_CONFIRMATION_ACCEPTED events
    # and publishes RIDER_ASSIGNED or RIDER_ASSIGNMENT_FAILED events

  # ============================================================================
  # Observability Stack
  # ============================================================================
  # NOTE: Each service runs its own Prometheus + Grafana stack
  # - Quote Service: Prometheus :9090, Grafana :3000 (via docker-compose.observability.yml)
  # - Location Service: Prometheus :9091, Grafana :3001 (via observability/docker-compose.observability.yml)
  # - Order Service: Prometheus :9092, Grafana :3002 (via docker-compose.observability.yml)
  # 
  # This integration docker-compose only provides shared infrastructure.
  # Start each service's observability stack separately from their respective repos.

networks:
  dispatch-net:
    driver: bridge

volumes:
  redis_data:
  order_postgres_data:
  location_postgres_data:
  osrm_data:
```

### 2.2 Port Mapping Summary

| Service | Port | Purpose | Access |
|---------|------|---------|--------|
| **Order Service** | 8082 | HTTP (health, metrics) | http://localhost:8082 |
| **Quote Service** | 8080 | HTTP (health, metrics) | http://localhost:8080 |
| **Location Service** | 8081 | HTTP (health, metrics) | http://localhost:8081 |
| **Location Service** | 9081 | gRPC | localhost:9081 |
| **UOIS Gateway** | 8083 | HTTP (health, metrics, ONDC) | http://localhost:8083 |
| **Redis** | 6380 | Streams & Cache (Shared) | localhost:6380 |
| **Order Postgres** | 5433 | Order DB | localhost:5433 |
| **Location Postgres** | 5435 | Location DB (PostGIS) | localhost:5435 |
| **Postgres-E (Audit)** | 5436 | UOIS Gateway Audit DB | localhost:5436 |
| **OSRM** | 5000 | Routing Engine | http://localhost:5000 |
| **Admin Mock** | 50051 | gRPC Mock | localhost:50051 |
| **DroneAI Mock** | - | Docker Container | Runs in Docker, no exposed ports |
| **Quote Prometheus** | 9090 | Quote Metrics | http://localhost:9090 |
| **Quote Grafana** | 3000 | Quote Dashboards | http://localhost:3000 |
| **Location Prometheus** | 9091 | Location Metrics | http://localhost:9091 |
| **Location Grafana** | 3001 | Location Dashboards | http://localhost:3001 |
| **Order Prometheus** | 9092 | Order Metrics | http://localhost:9092 |
| **Order Grafana** | 3002 | Order Dashboards | http://localhost:3002 |
| **UOIS Prometheus** | 9093 | UOIS Gateway Metrics | http://localhost:9093 |
| **UOIS Grafana** | 3003 | UOIS Gateway Dashboards | http://localhost:3003 |

---

## 3️⃣ Redis Streams Contract

### 3.1 Complete Stream Mapping

**Note:** Stream names below match actual implementation from codebase (env.example, config files, and hardcoded constants).

| Stream Name | Producer | Consumer(s) | Event Type(s) | Purpose |
|-------------|----------|-------------|---------------|---------|
| `stream.location.search` | UOIS Gateway | Location Service | `SEARCH_REQUESTED` | `/search` flow trigger |
| `location:serviceability:found` | Location Service | Quote Service | `SERVICEABILITY_FOUND` | Serviceability results |
| `quote:computed` | Quote Service | UOIS Gateway | `QUOTE_COMPUTED` | Quote for `/search` |
| `stream.uois.init_requested` | UOIS Gateway | Order Service | `INIT_REQUESTED` | `/init` flow trigger |
| `stream.location.revalidation` | Order Service | Location Service | `REVALIDATION_REQUESTED` | Revalidation trigger |
| `location:revalidation:found` | Location Service | Order Service, Quote Service | `REVALIDATION_FOUND` | Revalidation results |
| `quote:revalidation:computed` | Quote Service | Order Service | `REVALIDATION_QUOTE_COMPUTED` | Revalidation quote |
| `stream.uois.confirm_requested` | UOIS Gateway | Order Service | `CONFIRM_REQUESTED` | `/confirm` flow trigger |
| `stream.droneai.confirmation_accepted` | Order Service | DroneAI (Mock) | `ORDER_CONFIRMATION_ACCEPTED` | Order creation success |
| `stream.droneai.order.assigned` ⚠️ | DroneAI (Mock) | Order Service | `RIDER_ASSIGNED` | Rider assignment (hardcoded in code) |
| `stream.droneai.order.assign_failed` ⚠️ | DroneAI (Mock) | Order Service | `RIDER_ASSIGNMENT_FAILED` | Rider assignment failure (hardcoded in code) |
| `stream.order.events` | Order Service | Location Service | `order.assigned`, `order.delivered`, `order.cancelled`, `order.rto_delivered` | Order lifecycle |
| `stream.location.soft_arrived` | Location Service | Order Service | `soft_arrived` | Soft arrived detection |
| `stream.location.geofence.entered` | Location Service | Order Service | `geofence.entered` | Geofence entry events |
| `stream.location.geofence.exited` | Location Service | Analytics Service | `geofence.exited` | Geofence exit events |
| `stream.location.zone.exit` | Location Service | Analytics Service | `zone.exit` | Zone exit events |
| `stream.order.distance.completed` | Location Service | Order Service | `distance.completed` | Distance completion |
| `stream.location.order.completion.update` | Location Service | WebSocket Gateway | `completion.update` | Order completion percentage updates |
| `stream.rider.location.update` | WebSocket Gateway | Location Service | `rider.location.update` | Real-time GPS location updates |
| `stream.order.dlq` | Order Service | (Manual) | (Poison messages) | Dead Letter Queue |
| `stream.uois.quote_created` | Order Service | UOIS Gateway | `QUOTE_CREATED` | Quote validation success |
| `stream.uois.quote_invalidated` | Order Service | UOIS Gateway | `QUOTE_INVALIDATED` | Quote validation failure |
| `stream.uois.confirmation_rejected` | Order Service | UOIS Gateway | `ORDER_CONFIRMATION_REJECTED` | Order creation rejection |
| `stream.uois.order_confirmed` | Order Service | UOIS Gateway | `ORDER_CONFIRMED` | Order confirmation success |
| `stream.uois.confirm_failed` | Order Service | UOIS Gateway | `ORDER_CONFIRM_FAILED` | Order confirmation failure |

**⚠️ Implementation Note:** Order Service code uses hardcoded `stream.droneai.order.assigned` and `stream.droneai.order.assign_failed`, but `env.example` has `stream.droneai.rider_assigned` and `stream.droneai.rider_assignment_failed`. The actual implementation uses the `.order.assigned` variant.

### 3.2 Consumer Group Strategy

| Service | Consumer Group | Streams Consumed |
|---------|----------------|------------------|
| **Order Service** | `order-service-group` | `stream.uois.init_requested`, `stream.uois.confirm_requested`, `quote:revalidation:computed`, `location:revalidation:found`, `stream.location.soft_arrived`, `stream.location.geofence.entered`, `stream.order.distance.completed`, `stream.droneai.order.assigned`, `stream.droneai.order.assign_failed` |
| **Quote Service** | `quote-service-group` | `location:serviceability:found`, `location:revalidation:found` |
| **Location Service** | `location-service-consumers` | `stream.location.search`, `stream.location.revalidation`, `stream.order.events`, `stream.rider.location.update` |
| **UOIS Gateway** | `uois-gateway-consumers` | `quote:computed`, `stream.uois.quote_created`, `stream.uois.quote_invalidated`, `stream.uois.order_confirmed`, `stream.uois.order_confirm_failed` |

### 3.3 Stream Ownership Rule

**Critical:** Stream naming follows a strict ownership model to prevent coupling:

- **Stream name prefix indicates producer ownership:**
  - `location:*` → Produced by **Location Service** (e.g., `location:serviceability:found`, `location:revalidation:found`)
  - `quote:*` → Produced by **Quote Service** (e.g., `quote:computed`, `quote:revalidation:computed`)
  - `stream.order.*` → Split by producer:
    - `stream.order.events` → Produced by **Order Service** (order lifecycle events)
    - `stream.order.dlq` → Produced by **Order Service** (dead letter queue)
    - `stream.order.distance.completed` → Produced by **Location Service** (distance tracking completion)
  - `stream.location.*` → Produced by **Location Service** (e.g., `stream.location.search`, `stream.location.revalidation`, `stream.location.soft_arrived`, `stream.location.geofence.entered`, `stream.location.geofence.exited`, `stream.location.zone.exit`, `stream.location.order.completion.update`)
  - `stream.uois.*` → Split by producer:
    - `stream.uois.init_requested`, `stream.uois.confirm_requested` → Produced by **external system (UOIS)**
    - `stream.uois.quote_created`, `stream.uois.quote_invalidated`, `stream.uois.order_confirmed`, `stream.uois.confirm_failed`, `stream.uois.confirmation_rejected` → Produced by **Order Service**
  - `stream.droneai.*` → Split by producer:
    - `stream.droneai.order.assigned`, `stream.droneai.order.assign_failed` → Produced by **external system (DroneAI)**
    - `stream.droneai.confirmation_accepted` → Produced by **Order Service**
  - `stream.rider.*` → Produced by **WebSocket Gateway** (e.g., `stream.rider.location.update`)

- **Consumers must never mutate or re-emit events from streams they don't own**
- **Only the producer service may:**
  - Create the stream
  - Define the event schema
  - Modify stream configuration
  - Archive or delete the stream

**Why:** This ownership model ensures clear boundaries, prevents accidental coupling, and makes stream evolution predictable.

### 3.4 Implementation Notes

**Consumer Group Names (from actual code):**
- **Order Service:** `order-service-group` (from `CONSUMER_GROUP_NAME` env var)
- **Quote Service:** `quote-service-group` (from `CONSUMER_GROUP_NAME` env var)
- **Location Service:** `location-service-consumers` (hardcoded in `internal/consumer/consumer.go`)
- **UOIS Gateway:** `uois-gateway-consumers` (from `CONSUMER_GROUP_NAME` env var, defaults to `uois-gateway-consumers`)

**⚠️ Redis Version Requirement:**
- **All services require Redis 5.0+** for Streams support (XGROUP command)
- **Recommended:** `redis:7-alpine` (used in docker-compose)
- **If you see "ERR unknown command 'xgroup'" error:**
  - Upgrade Redis to 5.0+ (docker-compose uses `redis:7-alpine` which supports Streams)
  - Consumer groups are required for UOIS Gateway, Order Service, Quote Service, and Location Service

**Stream Name Discrepancies:**
- Order Service code uses hardcoded `stream.droneai.order.assigned` and `stream.droneai.order.assign_failed` (in `redis_consumer.go`), but `env.example` has `stream.droneai.rider_assigned` and `stream.droneai.rider_assignment_failed`. **The actual implementation uses the `.order.assigned` variant.**

**Additional Streams (not in main flow but implemented):**
- `stream.rider.location.update` - WebSocket Gateway → Location Service (GPS updates)
- `stream.location.geofence.exited` - Location Service → Analytics Service
- `stream.location.zone.exit` - Location Service → Analytics Service
- `stream.location.order.completion.update` - Location Service → WebSocket Gateway
- `stream.uois.quote_created` - Order Service → UOIS
- `stream.uois.quote_invalidated` - Order Service → UOIS
- `stream.uois.confirmation_rejected` - Order Service → UOIS
- `stream.uois.order_confirmed` - Order Service → UOIS
- `stream.uois.confirm_failed` - Order Service → UOIS

---

## 4️⃣ Database Strategy

### 4.1 Separate Databases Per Service

**Rationale:**
- **Order Service:** Owns `orders` schema with `orders.orders`, `orders.quotes`, `orders.order_events`
- **Location Service:** Owns `location` schema with location history and tracking data
- **Isolation:** Prevents schema conflicts, allows independent migrations, clear ownership

### 4.2 Migration Execution Order

1. **Start Order Postgres** → Run Order Service migrations (if not already done)
2. **Start Location Postgres** → Run Location Service database setup script
3. **Start Services** → Services connect to respective databases

**Location Service Database Setup:**

Location Service requires PostGIS extension. Use the provided setup script:

```powershell
# From Location-Service-Dispatch root
cd scripts
.\setup-location-db.ps1

# Or if using existing Postgres instance (e.g., Order DB container):
.\setup-location-db.ps1 -UseExistingPostgres -ExistingPostgresUser admin -ExistingPostgresPassword "1234Qwert@18"
```

**Note:** The script will:
- Create Location Service database
- Enable PostGIS extension
- Create required tables (routes, route_milestones, serviceable_zones, gdpr_deletions)
- Verify setup completion

### 4.3 Database Connection Strings

**Order Service:**
```
DATABASE_URL=postgres://admin:simplepass@127.0.0.1:5433/order_service_dev?sslmode=disable
REDIS_HOST=localhost
REDIS_PORT=6380
```

**Location Service:**
```
# Using PostGIS container (port 5435):
POSTGRES_URL=postgres://location_service:password@localhost:5435/location_service?sslmode=disable
REDIS_URL=redis://localhost:6380/0
GRPC_PORT=9081
```

---

## 5️⃣ Quote ↔ Location Integration Flow

### 5.1 `/search` Flow (Step-by-Step)

```
1. Mock UOIS publishes SEARCH_REQUESTED:
   redis-cli XADD stream.location.search * event_type SEARCH_REQUESTED \
     search_id 550e8400-e29b-41d4-a716-446655440000 \
     origin_lat 12.9352 origin_lng 77.6245 \
     destination_lat 12.9716 destination_lng 77.5946 \
     traceparent "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"

2. Location Service consumes SEARCH_REQUESTED
   - Queries GeoRedis for riders in buckets D1-D5
   - Calls OSRM for route distance
   - Computes ETAs
   - Publishes SERVICEABILITY_FOUND to location:serviceability:found

3. Quote Service consumes SERVICEABILITY_FOUND
   - Calls Admin Service gRPC for pricing config
   - Computes price using distance + bucket_counts
   - Publishes QUOTE_COMPUTED to quote:computed

4. Mock UOIS consumes QUOTE_COMPUTED
   - Displays quote to user
```

### 5.2 `/init` Revalidation Flow

```
1. Order Service receives INIT_REQUESTED
   - Checks quote TTL
   - If time_elapsed > 1 minute, publishes REVALIDATION_REQUESTED

2. Location Service consumes REVALIDATION_REQUESTED
   - Recomputes serviceability
   - Publishes REVALIDATION_FOUND to location:revalidation:found

3. Quote Service consumes REVALIDATION_FOUND
   - Recomputes pricing
   - Publishes REVALIDATION_QUOTE_COMPUTED to quote:revalidation:computed

4. Order Service consumes REVALIDATION_QUOTE_COMPUTED
   - Validates price deviation
   - Creates quote_id and persists quote
```

### 5.3 Failure Scenarios

**Location Service Unavailable:**
- Quote Service: No SERVICEABILITY_FOUND events → No quotes computed
- Order Service: Revalidation fails → Falls back to cached quote if valid

**OSRM Unavailable:**
- Location Service: Falls back to straight-line distance (if configured)
- Serviceability may be degraded

**Redis Stream Backlog:**
- Check with: `XPENDING location:serviceability:found quote-service-group`
- Scale consumers or investigate processing delays

---

## 6️⃣ Order Creation Flow (Mocking UOIS)

### 6.1 Mock UOIS Script

**File:** `Order-Service-Dispatch/scripts/mock-uois.ps1` (Windows) or `mock-uois.sh` (Linux)

```powershell
# Mock UOIS - Publishes INIT_REQUESTED and CONFIRM_REQUESTED events

param(
    [string]$RedisHost = "localhost",
    [int]$RedisPort = 6380,
    [string]$SearchID = (New-Guid).ToString()
)

$redis = "redis-cli -h $RedisHost -p $RedisPort"

# Generate traceparent
$traceId = -join ((48..57) + (97..102) | Get-Random -Count 32 | ForEach-Object {[char]$_})
$spanId = -join ((48..57) + (97..102) | Get-Random -Count 16 | ForEach-Object {[char]$_})
$traceparent = "00-$traceId-$spanId-01"

Write-Host "Mock UOIS: Publishing INIT_REQUESTED for search_id=$SearchID"

# Publish INIT_REQUESTED
$initEvent = @{
    event_type = "INIT_REQUESTED"
    event_id = (New-Guid).ToString()
    search_id = $SearchID
    pickup = '{"lat":12.9352,"lng":77.6245,"address":"123 Main St"}'
    drop = '{"lat":12.9716,"lng":77.5946,"address":"456 Oak Ave"}'
    package_info = '{"weight":1.5,"dimensions":{"length":10,"width":5,"height":3}}'
    timestamp = (Get-Date -Format "yyyy-MM-ddTHH:mm:ssZ")
    traceparent = $traceparent
} | ConvertTo-Json -Compress

& $redis XADD stream.uois.init_requested * event_type INIT_REQUESTED `
    event_id $($initEvent | ConvertFrom-Json | Select-Object -ExpandProperty event_id) `
    search_id $SearchID `
    data $initEvent

Write-Host "Mock UOIS: Published INIT_REQUESTED. Waiting 5 seconds..."
Start-Sleep -Seconds 5

# Publish CONFIRM_REQUESTED (assumes quote was created)
Write-Host "Mock UOIS: Publishing CONFIRM_REQUESTED for search_id=$SearchID"

$confirmEvent = @{
    event_type = "CONFIRM_REQUESTED"
    event_id = (New-Guid).ToString()
    quote_id = "quote_$SearchID"  # In real flow, this comes from QUOTE_COMPUTED
    client_id = (New-Guid).ToString()
    service_type = "STANDARD"
    timestamp = (Get-Date -Format "yyyy-MM-ddTHH:mm:ssZ")
    traceparent = $traceparent
} | ConvertTo-Json -Compress

& $redis XADD stream.uois.confirm_requested * event_type CONFIRM_REQUESTED `
    event_id $($confirmEvent | ConvertFrom-Json | Select-Object -ExpandProperty event_id) `
    quote_id "quote_$SearchID" `
    client_id $($confirmEvent | ConvertFrom-Json | Select-Object -ExpandProperty client_id) `
    data $confirmEvent

Write-Host "Mock UOIS: Published CONFIRM_REQUESTED"
```

### 6.2 Payload Examples

**INIT_REQUESTED:**
```json
{
  "event_type": "INIT_REQUESTED",
  "event_id": "550e8400-e29b-41d4-a716-446655440000",
  "search_id": "550e8400-e29b-41d4-a716-446655440001",
  "pickup": {
    "lat": 12.9352,
    "lng": 77.6245,
    "address": "123 Main Street, Bangalore"
  },
  "drop": {
    "lat": 12.9716,
    "lng": 77.5946,
    "address": "456 Oak Avenue, Bangalore"
  },
  "package_info": {
    "weight": 1.5,
    "dimensions": {
      "length": 10,
      "width": 5,
      "height": 3
    }
  },
  "timestamp": "2024-01-15T14:22:30Z",
  "traceparent": "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"
}
```

**CONFIRM_REQUESTED:**
```json
{
  "event_type": "CONFIRM_REQUESTED",
  "event_id": "550e8400-e29b-41d4-a716-446655440002",
  "quote_id": "550e8400-e29b-41d4-a716-446655440003",
  "client_id": "550e8400-e29b-41d4-a716-446655440004",
  "service_type": "STANDARD",
  "timestamp": "2024-01-15T14:25:00Z",
  "traceparent": "00-4bf92f3577b34da6a3ce929d0e0e4736-8f2a1b2c3d4e5f6a-01"
}
```

### 6.3 Redis Stream Injection Method

**Using redis-cli:**
```bash
# Publish INIT_REQUESTED
redis-cli XADD stream.uois.init_requested * \
  event_type INIT_REQUESTED \
  event_id $(uuidgen) \
  search_id 550e8400-e29b-41d4-a716-446655440000 \
  data '{"pickup":{"lat":12.9352,"lng":77.6245},"drop":{"lat":12.9716,"lng":77.5946}}'
```

**Using Python script:**
```python
import redis
import json
import uuid
from datetime import datetime

r = redis.Redis(host='localhost', port=6380, decode_responses=True)

event = {
    "event_type": "INIT_REQUESTED",
    "event_id": str(uuid.uuid4()),
    "search_id": str(uuid.uuid4()),
    "pickup": {"lat": 12.9352, "lng": 77.6245, "address": "123 Main St"},
    "drop": {"lat": 12.9716, "lng": 77.5946, "address": "456 Oak Ave"},
    "timestamp": datetime.utcnow().isoformat() + "Z",
    "traceparent": "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"
}

r.xadd("stream.uois.init_requested", event)
```

---

## 7️⃣ End-to-End Test Plan

### 7.1 Prerequisites

```bash
# 1. Clone all three repositories
git clone <order-service-repo>
git clone <quote-service-repo>
git clone <location-service-repo>

# 2. Install dependencies
cd Order-Service-Dispatch && go mod download
cd Quote-Service-Dispatch && go mod download
cd Location-Service-Dispatch && go mod download

# 3. Ensure Docker Desktop is running
docker ps
```

### 7.2 Step-by-Step Execution

#### Step 1: Start Infrastructure

```bash
# From Order-Service-Dispatch root
# Note: Observability stacks are started separately per-service (see Step 4)
docker-compose -f docker-compose.integration.yml up -d redis location-postgres osrm admin-service-mock droneai-mock

# Wait for health checks (30-60 seconds)
docker ps  # Verify all containers are healthy
# Check logs: docker logs dispatch-droneai-mock
```

#### Step 2: Run Database Migrations

**Order Service (if not already set up):**
```bash
# Order Service migrations
cd Order-Service-Dispatch
# Note: Order DB is already running on port 5433 (order-service-postgres-c container)
# Run migrations if needed:
psql -h localhost -p 5433 -U admin -d order_service_dev -f migrations/001_create_order_sequence_table.sql
```

**Location Service Database Setup:**
```powershell
# From Location-Service-Dispatch root directory
cd scripts

# Setup Location Service database with PostGIS (port 5435)
.\setup-location-db-docker.ps1 -ContainerName dispatch-location-postgres -PostgresUser location_service -PostgresPassword password

# The script will:
# - Create location_service database
# - Enable PostGIS extension
# - Create required tables (routes, route_milestones, serviceable_zones, gdpr_deletions)
# - Verify setup completion
```

**Note:** If you're already in the `Location-Service-Dispatch` directory, just run `cd scripts` (not `cd Location-Service-Dispatch\scripts`).

**Verify Location Service DB:**
```bash
# Test connection (PostGIS container on port 5435)
psql -h localhost -p 5435 -U location_service -d location_service -c "SELECT PostGIS_version();"
```

#### Step 3: Configure Services

**Order Service `.env`:**
```bash
SERVER_PORT=8082
DATABASE_URL=postgres://admin:simplepass@127.0.0.1:5433/order_service_dev?sslmode=disable
REDIS_HOST=localhost
REDIS_PORT=6380
STREAM_INIT_REQUESTED=stream.uois.init_requested
STREAM_CONFIRM_REQUESTED=stream.uois.confirm_requested
STREAM_REVALIDATION_QUOTE_COMPUTED=quote:revalidation:computed
STREAM_REVALIDATION_FOUND=location:revalidation:found
STREAM_REVALIDATION_REQUESTED=stream.location.revalidation
STREAM_ORDER_EVENTS=stream.order.events
METRICS_ENABLED=true
METRICS_PORT=8082
```

**Quote Service `.env`:**
```bash
SERVER_PORT=8080
REDIS_HOST=localhost
REDIS_PORT=6380
STREAM_SERVICEABILITY_FOUND=location:serviceability:found
STREAM_REVALIDATION_FOUND=location:revalidation:found
STREAM_QUOTE_COMPUTED=quote:computed
STREAM_REVALIDATION_QUOTE_COMPUTED=quote:revalidation:computed
ADMIN_SERVICE_GRPC_HOST=localhost
ADMIN_SERVICE_GRPC_PORT=50051
METRICS_ENABLED=true
METRICS_PORT=8080
```

**Location Service `.env`:**
```bash
HTTP_PORT=8081
GRPC_PORT=9081
REDIS_URL=redis://localhost:6380/0
OSRM_BASE_URL=http://localhost:5000
POSTGRES_URL=postgres://location_service:password@localhost:5435/location_service?sslmode=disable
STREAM_SEARCH_REQUESTED=stream.location.search
STREAM_REVALIDATION_REQUESTED=stream.location.revalidation
STREAM_SERVICEABILITY_FOUND=location:serviceability:found
STREAM_REVALIDATION_FOUND=location:revalidation:found
STREAM_ORDER_EVENTS=stream.order.events
METRICS_ENABLED=true
METRICS_PORT=8081
```

**UOIS Gateway `.env`:**
```bash
SERVER_PORT=8083
SERVER_HOST=localhost
REDIS_HOST=localhost
REDIS_PORT=6380  # ⚠️ IMPORTANT: Must be 6380 (Docker Redis), not 6379 (Windows Redis service)
REDIS_DB=0
POSTGRES_E_HOST=localhost
POSTGRES_E_PORT=5436
POSTGRES_E_USER=uois_gateway
POSTGRES_E_PASSWORD=password
POSTGRES_E_DB=uois_gateway
POSTGRES_E_SSL_MODE=disable
STREAM_QUOTE_COMPUTED=quote:computed
STREAM_QUOTE_CREATED=stream.uois.quote_created
STREAM_QUOTE_INVALIDATED=stream.uois.quote_invalidated
STREAM_ORDER_CONFIRMED=stream.uois.order_confirmed
STREAM_ORDER_CONFIRM_FAILED=stream.uois.order_confirm_failed
STREAM_INIT_REQUESTED=stream.uois.init_requested
STREAM_CONFIRM_REQUESTED=stream.uois.confirm_requested
CONSUMER_GROUP_NAME=uois-gateway-consumers
ONDC_SUBSCRIBER_ID=ondc.dispatchos.io
ONDC_UK_ID=2568
ONDC_CITY_CODE=std:020
ONDC_BPP_ID=ondc.dispatchos.io
ONDC_BPP_URI=https://ondc.dispatchos.io
METRICS_ENABLED=true
```

#### Step 4: Start Observability Stacks (Per-Service)

**✅ All port configurations are complete** (see `OBSERVABILITY_PORT_MAPPING.md`)

**Terminal 1 - Quote Service Observability:**
```bash
cd Quote-Service-Dispatch
# Ensure dispatch-net exists (created by integration docker-compose)
docker network create dispatch-net 2>$null
docker-compose -f docker-compose.observability.yml up -d
# Prometheus: http://localhost:9090
# Grafana: http://localhost:3000 (admin/admin)
```

**Terminal 2 - Location Service Observability:**
```bash
cd Location-Service-Dispatch
# Ensure observability/docker-compose.observability.yml uses dispatch-net as external network
docker-compose -f observability/docker-compose.observability.yml up -d
# Prometheus: http://localhost:9091
# Grafana: http://localhost:3001 (admin/admin)
```

**Terminal 3 - Order Service Observability:**
```bash
cd Order-Service-Dispatch
docker-compose -f docker-compose.observability.yml up -d
# Prometheus: http://localhost:9092
# Grafana: http://localhost:3002 (admin/admin)
```

#### Step 5: Start Services

**Terminal 4 - Order Service:**
```bash
cd Order-Service-Dispatch
go run cmd/server/main.go
```

**Terminal 5 - Quote Service:**
```bash
cd Quote-Service-Dispatch
go run cmd/server/main.go
```

**Terminal 6 - Location Service:**
```bash
cd Location-Service-Dispatch
go run cmd/server/main.go
```

**Terminal 7 - UOIS Gateway:**
```bash
cd UOIS-Gateway-Dispatch
go run cmd/server/main.go
# Or use the startup script:
.\scripts\start_uois_gateway.ps1
```

#### Step 6: Verify Services Started

```bash
# Check health endpoints
curl http://localhost:8082/health/live  # Order Service
curl http://localhost:8080/health/live   # Quote Service
curl http://localhost:8081/health/live  # Location Service
curl http://localhost:8083/health       # UOIS Gateway

# Check metrics endpoints
curl http://localhost:8082/metrics  # Order Service
curl http://localhost:8080/metrics # Quote Service
curl http://localhost:8081/metrics # Location Service
curl http://localhost:8083/metrics # UOIS Gateway

# Verify Docker containers (including Mock DroneAI)
docker ps | grep dispatch
# Should show: dispatch-redis, dispatch-location-postgres, dispatch-osrm, 
#              dispatch-admin-mock, dispatch-droneai-mock

# Check Mock DroneAI logs
docker logs dispatch-droneai-mock
# Should show: "=== Mock DroneAI Service ===" and "Listening for ORDER_CONFIRMATION_ACCEPTED events"

# Verify Prometheus targets
curl http://localhost:9092/api/v1/targets  # Order Service Prometheus
curl http://localhost:9090/api/v1/targets  # Quote Service Prometheus
curl http://localhost:9091/api/v1/targets  # Location Service Prometheus
curl http://localhost:9093/api/v1/targets  # UOIS Gateway Prometheus

# Access Grafana dashboards
# Order Service: http://localhost:3002 (admin/admin)
# Quote Service: http://localhost:3000 (admin/admin)
# Location Service: http://localhost:3001 (admin/admin)
# UOIS Gateway: http://localhost:3003 (admin/admin)
```

#### Step 7: Populate Test Data

**Seed Location Service with test riders:**
```bash
# Use Location Service seed script (if available)
cd Location-Service-Dispatch
go run scripts/seed_georedis_test_data.go
```

#### Step 8: Execute Happy Path Test

**8.1 Publish SEARCH_REQUESTED:**
```bash
redis-cli XADD stream.location.search * \
  event_type SEARCH_REQUESTED \
  event_id $(uuidgen) \
  search_id 550e8400-e29b-41d4-a716-446655440000 \
  origin_lat 12.9352 origin_lng 77.6245 \
  destination_lat 12.9716 destination_lng 77.5946 \
  timestamp "2024-01-15T14:22:30Z" \
  traceparent "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"
```

**8.2 Verify SERVICEABILITY_FOUND:**
```bash
# Use $ to read only new messages (avoids re-reading old events in repeated test runs)
redis-cli XREAD STREAMS location:serviceability:found $
# Should see SERVICEABILITY_FOUND event
# Note: First run may need 0 to read existing messages, subsequent runs use $ for new messages only
```

**8.3 Verify QUOTE_COMPUTED:**
```bash
# Use $ to read only new messages
redis-cli XREAD STREAMS quote:computed $
# Should see QUOTE_COMPUTED event with price
```

**8.4 Publish INIT_REQUESTED:**
```bash
redis-cli XADD stream.uois.init_requested * \
  event_type INIT_REQUESTED \
  event_id $(uuidgen) \
  search_id 550e8400-e29b-41d4-a716-446655440000 \
  data '{"pickup":{"lat":12.9352,"lng":77.6245},"drop":{"lat":12.9716,"lng":77.5946}}'
```

**8.5 Verify Order Created:**
```bash
psql -h localhost -U order_service -d order_service -c "SELECT * FROM orders.orders ORDER BY created_at DESC LIMIT 1;"
```

**8.6 Publish CONFIRM_REQUESTED:**
```bash
# Get quote_id from previous step
redis-cli XADD stream.uois.confirm_requested * \
  event_type CONFIRM_REQUESTED \
  event_id $(uuidgen) \
  quote_id <quote_id_from_db> \
  client_id $(uuidgen) \
  service_type STANDARD
```

**8.7 Verify Order Confirmed:**
```bash
psql -h localhost -U order_service -d order_service -c "SELECT dispatch_order_id, state FROM orders.orders ORDER BY created_at DESC LIMIT 1;"
# Should show state = CONFIRMED or ASSIGNED
```

### 7.3 Validation Commands

**Check Redis Streams:**
```bash
# List all streams
redis-cli KEYS "stream.*" "location:*" "quote:*"

# Check pending messages
redis-cli XPENDING location:serviceability:found quote-service-group
redis-cli XPENDING quote:revalidation:computed order-service-group

# Read stream messages (use $ for new messages only, 0 for all messages)
redis-cli XREAD COUNT 10 STREAMS location:serviceability:found $
# For historical messages, use: redis-cli XREAD COUNT 10 STREAMS location:serviceability:found 0
```

**Check Database Records:**
```bash
# Order Service DB
psql -h localhost -U order_service -d order_service -c "SELECT COUNT(*) FROM orders.orders;"
psql -h localhost -U order_service -d order_service -c "SELECT COUNT(*) FROM orders.quotes;"
psql -h localhost -U order_service -d order_service -c "SELECT COUNT(*) FROM orders.order_events;"

# Location Service DB
psql -h localhost -p 5435 -U location_service -d location_service -c "SELECT COUNT(*) FROM routes;"
```

**Check Metrics (Per-Service Prometheus):**
```bash
# Order Service Prometheus (port 9092)
curl http://localhost:9092/api/v1/targets
curl "http://localhost:9092/api/v1/query?query=order_service_orders_created_total"

# Quote Service Prometheus (port 9090)
curl http://localhost:9090/api/v1/targets
curl "http://localhost:9090/api/v1/query?query=quote_service_events_consumed_total"

# Location Service Prometheus (port 9091)
curl http://localhost:9091/api/v1/targets
curl "http://localhost:9091/api/v1/query?query=location_service_events_published_total"
```

---

## 8️⃣ Observability Setup (Per-Service)

### 8.1 Architecture: Option B - Per-Service Observability

**Each service runs its own Prometheus + Grafana stack independently.**

| Service | Prometheus Port | Grafana Port | Config Location |
|---------|----------------|--------------|-----------------|
| **Quote Service** | 9090 | 3000 | `Quote-Service-Dispatch/docker-compose.observability.yml` |
| **Location Service** | 9091 | 3001 | `Location-Service-Dispatch/observability/docker-compose.observability.yml` |
| **Order Service** | 9092 | 3002 | `Order-Service-Dispatch/docker-compose.observability.yml` (to be created) |

### 8.2 Starting Observability Stacks

**✅ All port configurations are complete** (see `OBSERVABILITY_PORT_MAPPING.md`)

**Quote Service:**
```bash
cd Quote-Service-Dispatch
# Ensure dispatch-net exists (created by integration docker-compose)
docker network create dispatch-net 2>$null
docker-compose -f docker-compose.observability.yml up -d
# Prometheus: http://localhost:9090
# Grafana: http://localhost:3000 (admin/admin)
```

**Location Service:**
```bash
cd Location-Service-Dispatch
# Update docker-compose to use external dispatch-net network (if not already)
# Ensure observability/docker-compose.observability.yml uses:
#   networks:
#     dispatch-net:
#       external: true
docker-compose -f observability/docker-compose.observability.yml up -d
# Prometheus: http://localhost:9091
# Grafana: http://localhost:3001 (admin/admin)
```

**Order Service:**
```bash
cd Order-Service-Dispatch
docker-compose -f docker-compose.observability.yml up -d
# Prometheus: http://localhost:9092
# Grafana: http://localhost:3002 (admin/admin)
```

**UOIS Gateway:**
```bash
cd UOIS-Gateway-Dispatch
docker-compose -f docker-compose.observability.yml up -d
# Prometheus: http://localhost:9093
# Grafana: http://localhost:3003 (admin/admin)
```

**Note:** All observability stacks use the shared `dispatch-net` Docker network (created by integration docker-compose).

### 8.3 Port Configuration Status

**✅ All ports configured correctly:**
- Quote Service: Prometheus 9090, Grafana 3000
- Location Service: Prometheus 9091, Grafana 3001
- Order Service: Prometheus 9092, Grafana 3002
- UOIS Gateway: Prometheus 9093, Grafana 3003

**See:** `OBSERVABILITY_PORT_MAPPING.md` for complete port assignment details.

### 8.4 Order Service Observability Setup

**File:** `Order-Service-Dispatch/docker-compose.observability.yml` (already created)

```yaml
version: '3.8'

services:
  prometheus:
    image: prom/prometheus:latest
    container_name: order-service-prometheus
    ports:
      - "9092:9090"
    volumes:
      - ./observability/prometheus/prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus_data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
    networks:
      - dispatch-net
    extra_hosts:
      - "host.docker.internal:host-gateway"

  grafana:
    image: grafana/grafana:latest
    container_name: order-service-grafana
    ports:
      - "3002:3000"
    environment:
      - GF_SECURITY_ADMIN_USER=admin
      - GF_SECURITY_ADMIN_PASSWORD=admin
    volumes:
      - grafana_data:/var/lib/grafana
      - ./observability/grafana/provisioning:/etc/grafana/provisioning
      - ./observability/grafana/dashboards:/var/lib/grafana/dashboards
    networks:
      - dispatch-net
    depends_on:
      - prometheus

networks:
  dispatch-net:
    external: true

volumes:
  prometheus_data:
  grafana_data:
```

**Create:** `Order-Service-Dispatch/observability/prometheus/prometheus.yml`

```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'order-service'
    static_configs:
      - targets: ['host.docker.internal:8082']
        labels:
          service: 'order-service'
          environment: 'local'
```

**Create:** `Order-Service-Dispatch/observability/grafana/provisioning/datasources/prometheus.yml`

```yaml
apiVersion: 1

datasources:
  - name: Prometheus
    type: prometheus
    access: proxy
    url: http://prometheus:9090
    isDefault: true
    editable: true
```

**Create:** `Order-Service-Dispatch/observability/grafana/provisioning/dashboards/dashboards.yml`

```yaml
apiVersion: 1

providers:
  - name: 'Order Service'
    orgId: 1
    folder: 'Order Service'
    type: file
    disableDeletion: false
    updateIntervalSeconds: 10
    allowUiUpdates: true
    options:
      path: /var/lib/grafana/dashboards/order-service
```

### 8.4 Metric Naming Consistency

**All services use service-prefixed metrics:**
- Order Service: `order_service_*`
- Quote Service: `quote_service_*`
- Location Service: `location_service_*`

**Example Queries (per-service Prometheus):**
```promql
# Order Service Prometheus (port 9092)
order_service_orders_created_total
order_service_state_transitions_total
order_service_events_consumed_total

# Quote Service Prometheus (port 9090)
quote_service_events_consumed_total{event_type="SERVICEABILITY_FOUND"}
quote_service_pricing_computation_duration_seconds

# Location Service Prometheus (port 9091)
location_service_events_published_total{event_type="SERVICEABILITY_FOUND"}
location_service_georedis_query_duration_seconds
```

### 8.5 Benefits of Per-Service Observability

✅ **Production Parity:** Mirrors K8s namespace isolation  
✅ **Independent Scaling:** Each service scales observability independently  
✅ **Clean Failure Domains:** Quote Prometheus failure doesn't affect Location  
✅ **Team Ownership:** Each team owns their complete observability stack  
✅ **Benchmark Accuracy:** No cross-service metric masking

---

## 9️⃣ Verification Checklist

### 9.1 Redis Stream Checks

```bash
# ✅ All streams exist
redis-cli KEYS "stream.*" "location:*" "quote:*"

# ✅ Consumer groups created
redis-cli XINFO GROUPS location:serviceability:found
redis-cli XINFO GROUPS quote:revalidation:computed
redis-cli XINFO GROUPS stream.uois.init_requested

# ✅ No pending messages (after processing)
redis-cli XPENDING location:serviceability:found quote-service-group
redis-cli XPENDING quote:revalidation:computed order-service-group

# ✅ Stream lag is zero
redis-cli XINFO STREAM location:serviceability:found
```

### 9.2 Database Record Checks

```sql
-- Order Service DB
SELECT COUNT(*) as order_count FROM orders.orders;
SELECT COUNT(*) as quote_count FROM orders.quotes;
SELECT COUNT(*) as event_count FROM orders.order_events;
SELECT dispatch_order_id, state, created_at FROM orders.orders ORDER BY created_at DESC LIMIT 5;

-- Location Service DB
SELECT COUNT(*) as history_count FROM location.location_history;
```

### 9.3 Metrics Visibility Checks (Per-Service)

```bash
# ✅ Order Service Prometheus (port 9092)
curl http://localhost:9092/api/v1/targets | jq '.data.activeTargets[] | {job: .job, health: .health}'
curl "http://localhost:9092/api/v1/query?query=order_service_orders_created_total" | jq

# ✅ Quote Service Prometheus (port 9090)
curl http://localhost:9090/api/v1/targets | jq '.data.activeTargets[] | {job: .job, health: .health}'
curl "http://localhost:9090/api/v1/query?query=quote_service_events_consumed_total" | jq

# ✅ Location Service Prometheus (port 9091)
curl http://localhost:9091/api/v1/targets | jq '.data.activeTargets[] | {job: .job, health: .health}'
curl "http://localhost:9091/api/v1/query?query=location_service_events_published_total" | jq

# ✅ Grafana dashboards loaded (separate per service)
# Order Service: http://localhost:3002 → Dashboards → Order Service
# Quote Service: http://localhost:3000 → Dashboards → Quote Service
# Location Service: http://localhost:3001 → Dashboards → Location Service
```

### 9.4 Logs & Tracing Sanity

```bash
# ✅ All services logging
# Check service logs for:
# - "Starting HTTP server"
# - "Consumer group created"
# - "Event consumed" / "Event published"
# - No ERROR level logs (except expected failures)

# ✅ Trace context propagation
# Check logs for trace_id fields in structured logs
# Verify traceparent passed through events unchanged
```

---

## 🔟 Complete Happy Path End-to-End

### 10.1 Full Flow Script

**File:** `Order-Service-Dispatch/scripts/test-e2e-flow.ps1`

```powershell
# Complete end-to-end test script

$searchId = (New-Guid).ToString()
$traceId = -join ((48..57) + (97..102) | Get-Random -Count 32 | ForEach-Object {[char]$_})
$spanId = -join ((48..57) + (97..102) | Get-Random -Count 16 | ForEach-Object {[char]$_})
$traceparent = "00-$traceId-$spanId-01"

Write-Host "=== Starting E2E Test ===" -ForegroundColor Green
Write-Host "Search ID: $searchId" -ForegroundColor Cyan
Write-Host "Traceparent: $traceparent" -ForegroundColor Cyan

# Step 1: Publish SEARCH_REQUESTED
Write-Host "`n[1/7] Publishing SEARCH_REQUESTED..." -ForegroundColor Yellow
redis-cli XADD stream.location.search * `
  event_type SEARCH_REQUESTED `
  event_id $(New-Guid) `
  search_id $searchId `
  origin_lat 12.9352 origin_lng 77.6245 `
  destination_lat 12.9716 destination_lng 77.5946 `
  timestamp $(Get-Date -Format "yyyy-MM-ddTHH:mm:ssZ") `
  traceparent $traceparent

Start-Sleep -Seconds 3

# Step 2: Verify SERVICEABILITY_FOUND
Write-Host "[2/7] Checking SERVICEABILITY_FOUND..." -ForegroundColor Yellow
# Use $ for new messages only (avoids re-reading old events in repeated test runs)
$serviceability = redis-cli XREAD COUNT 1 STREAMS location:serviceability:found $
if ($serviceability) {
    Write-Host "✅ SERVICEABILITY_FOUND received" -ForegroundColor Green
} else {
    Write-Host "❌ SERVICEABILITY_FOUND not found" -ForegroundColor Red
    exit 1
}

Start-Sleep -Seconds 2

# Step 3: Verify QUOTE_COMPUTED
Write-Host "[3/7] Checking QUOTE_COMPUTED..." -ForegroundColor Yellow
# Use $ for new messages only
$quote = redis-cli XREAD COUNT 1 STREAMS quote:computed $
if ($quote) {
    Write-Host "✅ QUOTE_COMPUTED received" -ForegroundColor Green
} else {
    Write-Host "❌ QUOTE_COMPUTED not found" -ForegroundColor Red
    exit 1
}

# Step 4: Publish INIT_REQUESTED
Write-Host "[4/7] Publishing INIT_REQUESTED..." -ForegroundColor Yellow
redis-cli XADD stream.uois.init_requested * `
  event_type INIT_REQUESTED `
  event_id $(New-Guid) `
  search_id $searchId `
  data "{`"pickup`":{`"lat`":12.9352,`"lng`":77.6245},`"drop`":{`"lat`":12.9716,`"lng`":77.5946}}"

Start-Sleep -Seconds 5

# Step 5: Verify Order Created
Write-Host "[5/7] Checking Order DB..." -ForegroundColor Yellow
$orderCount = psql -h localhost -U order_service -d order_service -t -c "SELECT COUNT(*) FROM orders.orders WHERE quote_id IS NOT NULL;"
if ([int]$orderCount -gt 0) {
    Write-Host "✅ Order created in DB" -ForegroundColor Green
} else {
    Write-Host "❌ No order found in DB" -ForegroundColor Red
    exit 1
}

# Step 6: Publish CONFIRM_REQUESTED
Write-Host "[6/7] Publishing CONFIRM_REQUESTED..." -ForegroundColor Yellow
$quoteId = psql -h localhost -U order_service -d order_service -t -c "SELECT quote_id FROM orders.quotes ORDER BY created_at DESC LIMIT 1;"
redis-cli XADD stream.uois.confirm_requested * `
  event_type CONFIRM_REQUESTED `
  event_id $(New-Guid) `
  quote_id $quoteId.Trim() `
  client_id $(New-Guid) `
  service_type STANDARD

Start-Sleep -Seconds 3

# Step 7: Verify Order Confirmed
Write-Host "[7/7] Checking Order State..." -ForegroundColor Yellow
$orderState = psql -h localhost -U order_service -d order_service -t -c "SELECT state FROM orders.orders ORDER BY created_at DESC LIMIT 1;"
Write-Host "Order State: $($orderState.Trim())" -ForegroundColor Cyan
if ($orderState.Trim() -eq "CONFIRMED" -or $orderState.Trim() -eq "ASSIGNED") {
    Write-Host "✅ Order confirmed successfully" -ForegroundColor Green
} else {
    Write-Host "⚠️ Order state: $($orderState.Trim())" -ForegroundColor Yellow
}

Write-Host "`n=== E2E Test Complete ===" -ForegroundColor Green
```

### 10.2 Expected Output

```
=== Starting E2E Test ===
Search ID: 550e8400-e29b-41d4-a716-446655440000
Traceparent: 00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01

[1/7] Publishing SEARCH_REQUESTED...
✅ SERVICEABILITY_FOUND received
✅ QUOTE_COMPUTED received
✅ Order created in DB
✅ Order confirmed successfully

=== E2E Test Complete ===
```

---

## 📝 Additional Notes

### Port Conflicts Prevention

- Order Service: 8082 (HTTP)
- Quote Service: 8080 (HTTP)
- Location Service: 8081 (HTTP), 9081 (gRPC)
- UOIS Gateway: 8083 (HTTP) - ONDC endpoints at `/ondc/*`
- All services use same Redis (6380) but different consumer groups
- Separate Postgres databases: Order (5433), Location (5435 with PostGIS), UOIS Gateway Audit (5436)
- **Note:** Redis moved to 6380 to avoid conflict with Windows Redis service on 6379
- **Note:** Location Postgres moved to 5435 to avoid conflict with Order Postgres on 5433
- **Note:** UOIS Gateway Postgres-E on 5436 (separate audit database)
- **⚠️ IMPORTANT:** Redis 5.0+ required for Streams support (all services use consumer groups)

### Windows-Specific Notes

- Use `host.docker.internal` for service-to-Docker communication
- PowerShell scripts provided for Windows
- Use `redis-cli.exe` if installed separately

### Linux-Specific Notes

- Replace `host.docker.internal` with `172.17.0.1` in Prometheus config
- Or add `extra_hosts: ["host.docker.internal:host-gateway"]` to docker-compose
- Use bash scripts instead of PowerShell

### Troubleshooting

**Services not connecting to Redis:**
- Verify Redis container is running: `docker ps | grep redis`
- Check Redis logs: `docker logs dispatch-redis`
- Test connection: `redis-cli -h localhost -p 6380 ping`
- **Note:** Redis is on port 6380 (not 6379) to avoid Windows Redis service conflict

**⚠️ Redis Consumer Groups Error - "ERR unknown command 'xgroup'":**

This error can occur for two reasons:

1. **Wrong Redis Port (Most Common):**
   - **Symptom:** Service connects to Redis but gets "ERR unknown command 'xgroup'"
   - **Root Cause:** Service is connecting to port 6379 (Windows Redis service or old local Redis) instead of port 6380 (Docker Redis container)
   - **Solution:** 
     - Check `.env` file: `REDIS_PORT` should be `6380` (not `6379`)
     - Verify Docker Redis is running: `docker ps | grep dispatch-redis`
     - Test correct Redis: `redis-cli -h localhost -p 6380 XGROUP HELP`
   - **Prevention:** Always use `REDIS_PORT=6380` in all service `.env` files

2. **Old Redis Version:**
   - **Symptom:** Even on port 6380, you get "ERR unknown command 'xgroup'"
   - **Root Cause:** Redis version is too old (requires Redis 5.0+)
   - **Solution:**
     - Upgrade to `redis:7-alpine` in docker-compose (already configured)
     - Restart Redis container: `docker-compose restart redis`
     - Verify version: `redis-cli -h localhost -p 6380 INFO server | grep redis_version`
   - **All services require:** UOIS Gateway, Order Service, Quote Service, and Location Service all use consumer groups and require Redis Streams support

**Prometheus not scraping:**
- Check targets per service:
  - Order Service: http://localhost:9092/targets
  - Quote Service: http://localhost:9090/targets
  - Location Service: http://localhost:9091/targets
- Verify services expose `/metrics` endpoint
- Check Prometheus logs:
  - Order: `docker logs order-service-prometheus`
  - Quote: `docker logs quote-service-prometheus`
  - Location: `docker logs location-service-prometheus`

**Grafana no data:**
- Set time range to "Last 1 hour"
- Wait 1-2 minutes after services start
- Verify Prometheus datasource: Configuration → Data Sources → Test
- **Important:** Each service has its own Grafana instance:
  - Order Service: http://localhost:3002
  - Quote Service: http://localhost:3000
  - Location Service: http://localhost:3001

---

**This plan provides a complete, executable integration setup for local development and testing.**

