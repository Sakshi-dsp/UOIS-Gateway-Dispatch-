# Redis Streams Architecture Alignment Analysis

## Problem Statement

The UOIS Implementation Plan and UOIS Gateway FR documents describe event consumption using "subscribe" terminology, which doesn't align with Redis Streams architecture. Redis Streams uses **consumer groups** with `XREADGROUP`, not pub/sub subscriptions.

## Key Misalignments

### 1. Event Consumption Pattern

**Current Documentation (INCORRECT):**
- Uses "subscribe" terminology: "Subscribe to `QUOTE_COMPUTED` event from stream `quote:computed`"
- Implies synchronous waiting: "waiting for `QUOTE_COMPUTED` event with matching `search_id`"
- Suggests direct event filtering: "correlated by `search_id`"

**Redis Streams Reality (CORRECT):**
- Uses **consumer groups** with `XREADGROUP` command
- Asynchronous event consumption loop
- Events must be filtered after consumption (no built-in filtering by correlation ID)
- Consumer group name: `uois-gateway-consumers`
- Consumer name: unique per instance (e.g., `uois-gateway-instance-1`)

### 2. Event Correlation Pattern

**Current Documentation (INCORRECT):**
- Implies Redis Streams can filter events by `search_id` automatically
- Suggests direct correlation matching in stream

**Redis Streams Reality (CORRECT):**
- Events are consumed sequentially from stream
- Must filter events by `search_id`/`quote_id` after consumption
- Need to maintain pending event map: `pending_events:{search_id}` → event data
- Use Redis Stream message ID for ACK after processing

### 3. Timeout Handling

**Current Documentation (INCORRECT):**
- Implies built-in timeout mechanism: "If `QUOTE_COMPUTED` not received within TTL period"
- Suggests synchronous waiting with timeout

**Redis Streams Reality (CORRECT):**
- Use `BLOCK` parameter in `XREADGROUP` for blocking reads
- Implement timeout logic: `XREADGROUP ... BLOCK 5000` (5 second timeout)
- Track request start time and compare against TTL
- Return timeout response if no matching event found within TTL

### 4. Consumer Group Configuration

**Current Documentation (MISSING):**
- No mention of consumer group names
- No mention of consumer names
- No mention of ACK mechanism
- No mention of pending entry list (PEL) handling

**Redis Streams Reality (REQUIRED):**
- Consumer group: `uois-gateway-consumers`
- Consumer name: unique per instance (e.g., `uois-gateway-{instance-id}`)
- ACK after successful processing: `XACK stream group-name message-id`
- Handle PEL entries on restart (reprocess unacked messages)

## Correct Architecture Pattern

### Event Publishing (Already Correct)

```go
// UOIS Gateway publishes events
XADD stream.location.search * event_type SEARCH_REQUESTED search_id {search_id} ...
```

### Event Consumption (NEEDS CORRECTION)

```go
// 1. Create consumer group (one-time setup)
XGROUP CREATE stream.location.search uois-gateway-consumers 0 MKSTREAM

// 2. Consume events in loop (per request context)
func ConsumeQuoteComputed(ctx context.Context, searchID string, timeout time.Duration) (*QuoteComputedEvent, error) {
    startTime := time.Now()
    pendingKey := fmt.Sprintf("pending:search:%s", searchID)
    
    for {
        // Check timeout
        if time.Since(startTime) > timeout {
            return nil, ErrTimeout
        }
        
        // Blocking read from consumer group
        messages, err := XREADGROUP(
            GROUP "uois-gateway-consumers" "uois-gateway-instance-1",
            BLOCK 5000, // 5 second block
            STREAMS quote:computed ">", // ">" means new messages
            COUNT 10, // Read up to 10 messages
        )
        
        if err != nil {
            // Handle error (timeout, connection, etc.)
            continue
        }
        
        // Filter messages by search_id
        for _, msg := range messages {
            event := parseEvent(msg.Values)
            if event.SearchID == searchID {
                // Found matching event
                XACK quote:computed uois-gateway-consumers msg.ID
                return event, nil
            } else {
                // Not our event - ACK and continue
                // OR: Store in pending map for other requests
                XACK quote:computed uois-gateway-consumers msg.ID
            }
        }
    }
}
```

### Alternative Pattern: Request-Scoped Consumer

For better isolation, use request-scoped consumer names:

```go
func HandleSearchRequest(request *SearchRequest) {
    searchID := generateSearchID()
    consumerName := fmt.Sprintf("uois-gateway-request-%s", searchID)
    
    // Create temporary consumer for this request
    // Consume events until timeout or match found
    // Delete consumer after request completes
}
```

## Required Changes to Documentation

### 1. UOIS Implementation Plan

**Section 5.1 `/search` Flow:**
- Replace "Subscribe to `QUOTE_COMPUTED` event" with "Consume from `quote:computed` stream using consumer group `uois-gateway-consumers`"
- Add consumer group configuration details
- Add ACK mechanism documentation
- Add timeout handling implementation details

**Section 5.2 `/init` Flow:**
- Replace "Subscribe to `QUOTE_CREATED` event" with "Consume from `stream.uois.quote_created` stream using consumer group"
- Add event filtering logic documentation

**Section 5.3 `/confirm` Flow:**
- Replace "Subscribe to `ORDER_CONFIRMED` event" with "Consume from `stream.uois.order_confirmed` stream using consumer group"
- Add event filtering logic documentation

### 2. UOIS Gateway FR

**Section 1.1 Event Publishing Pattern:**
- Add consumer group names for each stream
- Add consumer name pattern (unique per instance)
- Add ACK mechanism requirements

**Section 1.2 Response Composition:**
- Replace "Subscribe to event stream" with "Consume from consumer group"
- Add event filtering by correlation ID
- Add timeout handling implementation

**Section 6.1 Event Consumption for Callbacks:**
- Add consumer group configuration
- Add PEL (Pending Entry List) handling
- Add retry logic for unacked messages

## Implementation Requirements

### 1. Consumer Group Setup

```go
// Required consumer groups
streams := []string{
    "quote:computed",
    "stream.uois.quote_created",
    "stream.uois.quote_invalidated",
    "stream.uois.order_confirmed",
    "stream.uois.order_confirm_failed",
}

for _, stream := range streams {
    XGROUP CREATE stream uois-gateway-consumers 0 MKSTREAM
}
```

### 2. Event Consumer Service

```go
type EventConsumer struct {
    redisClient *redis.Client
    groupName   string
    consumerName string
    handlers    map[string]EventHandler
}

func (ec *EventConsumer) Consume(ctx context.Context, stream string, correlationID string, timeout time.Duration) (Event, error) {
    // Implementation with XREADGROUP, filtering, ACK
}
```

### 3. Request Context Management

```go
type RequestContext struct {
    SearchID    string
    QuoteID     string
    StartTime   time.Time
    Timeout     time.Duration
    PendingEvents map[string]Event // For event filtering
}
```

## Alignment with Other Services

### Location Service Pattern (Reference Implementation)

From `Location-Service-Dispatch/internal/consumer/consumer.go`:
- Uses consumer groups: `location-service-consumers`
- Uses `XREADGROUP` with blocking reads
- ACKs messages after processing
- Handles PEL on restart

### Quote Service Pattern (Reference Implementation)

From `Quote-Service-Dispatch/internal/clients/event_publisher.go`:
- Publishes events using `XADD`
- No consumer group (only publishes)

### Order Service Pattern (Reference Implementation)

From `Order-Service-Dispatch`:
- Uses consumer groups for event consumption
- ACKs messages after processing
- Handles event correlation

## Recommendations

1. **Update UOIS Implementation Plan** to use correct Redis Streams terminology
2. **Update UOIS Gateway FR** to document consumer group architecture
3. **Add consumer group configuration** to configuration section
4. **Document ACK mechanism** for event processing
5. **Document PEL handling** for reliability
6. **Add event filtering pattern** documentation
7. **Add timeout handling** implementation details

## References

- Redis Streams Documentation: https://redis.io/docs/data-types/streams/
- Location Service Consumer Implementation: `Location-Service-Dispatch/internal/consumer/consumer.go`
- Quote Service Publisher Implementation: `Quote-Service-Dispatch/internal/clients/event_publisher.go`
- Main Order Flows: `DispatchArchitecture/website/docs/06_Main_order_flows/`

