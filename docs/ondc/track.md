# track API (Seller NP)

## 1. Overview
- Purpose of this API: This API allows Buyer NP to request real-time tracking information for active deliveries
- Who calls it (Buyer NP / Seller NP): Called by Buyer NP (BAP)
- Sync vs Async behavior: Asynchronous - Buyer NP sends request, Seller NP responds with ACK/NACK synchronously, then sends callback via /on_track
- Callback expectations: Seller NP MUST send /on_track callback with current tracking information within TTL window
- TTL behavior (if applicable): TTL is specified in context.ttl (default PT30S)

## 2. Role Perspective
We are Seller NP (BPP / Logistics Service Provider). In this API:
- We receive tracking requests for active orders
- We validate that the order exists and has tracking enabled
- We return ACK/NACK synchronously
- We provide current GPS location and tracking details asynchronously via /on_track callback
- We must ensure tracking is enabled and rider location is available

## 3. Endpoint Details
- HTTP Method: POST
- Endpoint path: /track
- Content-Type: application/json
- Authentication & Signing requirement: Request must be signed by Buyer NP using their private key. We must verify the signature using the public key from registry lookup.

## 4. Request Payload

### 4.1 Full JSON Example
```json
{
  "context": {
    "domain": "nic2004:60232",
    "country": "IND",
    "city": "std:080",
    "action": "track",
    "core_version": "1.2.0",
    "bap_id": "logistics_buyer.com",
    "bap_uri": "https://logistics_buyer.com/ondc",
    "bpp_id": "lsp.com",
    "bpp_uri": "https://lsp.com/ondc",
    "transaction_id": "T1",
    "message_id": "M6",
    "timestamp": "2023-06-06T23:30:00.000Z",
    "ttl": "PT30S"
  },
  "message": {
    "order_id": "O2"
  }
}
```

### 4.2 Field-Level Schema (Table)

| Field Path | Data Type | Required (Y/N) | Description | Source of Truth |
|------------|-----------|----------------|-------------|-----------------|
| context.domain | string | Y | Domain identifier for logistics (nic2004:60232) | Buyer NP |
| context.country | string | Y | Country code (IND) | Buyer NP |
| context.city | string | Y | City code (std:080) | Buyer NP |
| context.action | string | Y | Action name (track) | Buyer NP |
| context.core_version | string | Y | ONDC core version (1.2.0) | Buyer NP |
| context.bap_id | string | Y | Buyer NP subscriber ID | Buyer NP |
| context.bap_uri | string | Y | Buyer NP callback URI | Buyer NP |
| context.bpp_id | string | Y | Seller NP subscriber ID | Buyer NP |
| context.bpp_uri | string | Y | Seller NP callback URI | Buyer NP |
| context.transaction_id | string | Y | Transaction ID from order creation | Buyer NP |
| context.message_id | string | Y | Unique message identifier | Buyer NP |
| context.timestamp | string | Y | Request timestamp in RFC3339 format | Buyer NP |
| context.ttl | string | Y | Time to live for response (PT30S) | Buyer NP |
| message.order_id | string | Y | Order identifier to track | Buyer NP |

## 5. Synchronous Response (ACK/NACK)

### 5.1 ACK Example
```json
{
  "message": {
    "ack": {
      "status": "ACK"
    }
  },
  "context": {
    "domain": "nic2004:60232",
    "country": "IND",
    "city": "std:080",
    "action": "track",
    "core_version": "1.2.0",
    "bap_id": "logistics_buyer.com",
    "bap_uri": "https://logistics_buyer.com/ondc",
    "bpp_id": "lsp.com",
    "bpp_uri": "https://lsp.com/ondc",
    "transaction_id": "T1",
    "message_id": "M6",
    "timestamp": "2023-06-06T23:30:30.000Z"
  }
}
```

### 5.2 NACK Example
```json
{
  "message": {
    "ack": {
      "status": "NACK"
    },
    "error": {
      "type": "DOMAIN-ERROR",
      "code": "60012",
      "message": "Tracking not available for this order"
    }
  },
  "context": {
    "domain": "nic2004:60232",
    "country": "IND",
    "city": "std:080",
    "action": "track",
    "core_version": "1.2.0",
    "bap_id": "logistics_buyer.com",
    "bap_uri": "https://logistics_buyer.com/ondc",
    "bpp_id": "lsp.com",
    "bpp_uri": "https://lsp.com/ondc",
    "transaction_id": "T1",
    "message_id": "M6",
    "timestamp": "2023-06-06T23:30:30.000Z"
  }
}
```

## 6. Asynchronous Callback (if applicable)

### 6.1 Callback Endpoint
- /on_track

### 6.2 Full Callback Payload Example
```json
{
  "context": {
    "domain": "nic2004:60232",
    "country": "IND",
    "city": "std:080",
    "action": "on_track",
    "core_version": "1.2.0",
    "bap_id": "logistics_buyer.com",
    "bap_uri": "https://logistics_buyer.com/ondc",
    "bpp_id": "lsp.com",
    "bpp_uri": "https://lsp.com/ondc",
    "transaction_id": "T1",
    "message_id": "M6",
    "timestamp": "2023-06-06T23:30:30.000Z",
    "ttl": "PT30S"
  },
  "message": {
    "tracking": {
      "id": "F1",
      "url": "https://lsp.com/ondc/track/F1",
      "location": {
        "gps": "12.974002,77.613458",
        "time": {
          "timestamp": "2023-06-06T23:30:00.000Z"
        },
        "updated_at": "2023-06-06T23:31:00.000Z"
      },
      "status": "active",
      "tags": [
        {
          "code": "order",
          "list": [
            {
              "code": "id",
              "value": "O2"
            }
          ]
        },
        {
          "code": "config",
          "list": [
            {
              "code": "attr",
              "value": "tracking.location.gps"
            },
            {
              "code": "type",
              "value": "live_poll"
            }
          ]
        },
        {
          "code": "path",
          "list": [
            {
              "code": "lat_lng",
              "value": "12.974002,77.613458"
            },
            {
              "code": "sequence",
              "value": "1"
            }
          ]
        },
        {
          "code": "path",
          "list": [
            {
              "code": "lat_lng",
              "value": "12.974077,77.613600"
            },
            {
              "code": "sequence",
              "value": "2"
            }
          ]
        },
        {
          "code": "path",
          "list": [
            {
              "code": "lat_lng",
              "value": "12.974098,77.613699"
            },
            {
              "code": "sequence",
              "value": "3"
            }
          ]
        }
      ]
    }
  }
}
```

### 6.3 Callback Field Schema
| Field Path | Data Type | Required (Y/N) | Description | Source |
|------------|-----------|----------------|-------------|--------|
| context.domain | string | Y | Domain identifier for logistics | Seller NP |
| context.country | string | Y | Country code | Seller NP |
| context.city | string | Y | City code | Seller NP |
| context.action | string | Y | Action name (on_track) | Seller NP |
| context.core_version | string | Y | ONDC core version | Seller NP |
| context.bap_id | string | Y | Buyer NP subscriber ID | Seller NP |
| context.bap_uri | string | Y | Buyer NP callback URI | Seller NP |
| context.bpp_id | string | Y | Seller NP subscriber ID | Seller NP |
| context.bpp_uri | string | Y | Seller NP callback URI | Seller NP |
| context.transaction_id | string | Y | Transaction ID from request | Seller NP |
| context.message_id | string | Y | Message ID from request | Seller NP |
| context.timestamp | string | Y | Response timestamp | Seller NP |
| context.ttl | string | Y | TTL for tracking data validity | Seller NP |
| message.tracking.id | string | Y | Fulfillment ID being tracked | Seller NP |
| message.tracking.url | string | N | Tracking URL (optional, for P2P GPS-based tracking is used) | Seller NP |
| message.tracking.location.gps | string | Y | Current GPS coordinates | Seller NP |
| message.tracking.location.time.timestamp | string | Y | Location timestamp | Seller NP |
| message.tracking.location.updated_at | string | Y | Location update timestamp | Seller NP |
| message.tracking.status | string | Y | Tracking status (active/inactive) | Seller NP |
| message.tracking.tags | array | Y | Tracking configuration and path data | Seller NP |

## 7. State & Correlation
- transaction_id: Must match the transaction_id from the original order creation
- message_id: Unique identifier for this track request
- order_id: The order identifier to track
- Correlation: Seller NP correlates the callback using transaction_id + message_id + order_id

## 8. Validation Rules
- Mandatory field checks: order_id must be present and valid
- Tracking validation: Order must have tracking enabled and be in active delivery state
- Order validation: Order must exist and belong to requesting buyer NP
- Timestamp & TTL validation: context.timestamp must be current time, response within TTL
- Stale request handling: If timestamp too old, respond with NACK error code 65003
- Signing verification: Verify Buyer's signature using registry public key

## 9. Error Scenarios

| Scenario | Error Code | When it occurs | Seller NP Action |
|----------|------------|----------------|------------------|
| Tracking not available | 60012 | Order not picked up or tracking disabled | Send NACK with LSP-ERROR |
| Order not found | 66004 | Order doesn't exist | Send NACK with LSP-ERROR |
| Service temporarily unavailable | 50001 | Internal system issues | Send NACK with LSP-ERROR |
| Stale request | 65003 | Request timestamp too old | Send NACK with PROTOCOL-ERROR |
| Invalid signature | 401 | Signature verification fails | Return HTTP 401 |

## 10. Important Notes (Seller NP)
- Seller NP MUST validate that tracking is enabled for the order
- Seller NP MUST ensure the order is in active delivery state (picked up)
- Seller NP MUST provide current GPS coordinates from the assigned rider
- Seller NP MUST include path data with sequence numbers for route tracking
- Seller NP MAY provide tracking URL if applicable (for P2P, GPS-based tracking is used)
- Seller NP MUST set status to "active" when tracking is available
- Common mistake to avoid: Sending tracking data for orders not yet picked up or with disabled tracking
