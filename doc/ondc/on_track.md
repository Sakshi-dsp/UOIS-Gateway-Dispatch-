# on_track API (Seller NP)

## 1. Overview
- Purpose of this API: This API allows Seller NP to provide real-time tracking information including GPS coordinates and route path for active deliveries
- Who calls it (Buyer NP / Seller NP): Called by Seller NP (BPP) as callback response to /track
- Sync vs Async behavior: Asynchronous - This is the callback response sent after ACK to /track request
- Callback expectations: Buyer NP acknowledges receipt with ACK/NACK
- TTL behavior (if applicable): Response must be sent within TTL specified in original /track request

## 2. Role Perspective
We are Seller NP (BPP / Logistics Service Provider). In this API:
- We provide real-time GPS tracking data for active deliveries
- We include route path information with sequence numbers
- We specify tracking configuration and status
- We send this as asynchronous callback to the buyer's /on_track endpoint
- We must ensure GPS data is current and accurate

## 3. Endpoint Details
- HTTP Method: POST
- Endpoint path: /on_track (at Buyer NP's bap_uri)
- Content-Type: application/json
- Authentication & Signing requirement: Request must be signed by Seller NP using their private key. Buyer NP must verify the signature.

## 4. Request Payload

### 4.1 Full JSON Example
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

### 4.2 Field-Level Schema (Table)

| Field Path | Data Type | Required (Y/N) | Description | Source of Truth |
|------------|-----------|----------------|-------------|-----------------|
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
| message.tracking.location.gps | string | Y | Current GPS coordinates (lat,lng) | Seller NP |
| message.tracking.location.time.timestamp | string | Y | GPS location timestamp | Seller NP |
| message.tracking.location.updated_at | string | Y | Location data update timestamp | Seller NP |
| message.tracking.status | string | Y | Tracking status (active/inactive) | Seller NP |
| message.tracking.tags | array | Y | Tracking metadata and path information | Seller NP |

## 5. Synchronous Response (ACK/NACK)

### 5.1 ACK Example
```json
{
  "message": {
    "ack": {
      "status": "ACK"
    }
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
      "code": "40001",
      "message": "Invalid tracking data format"
    }
  }
}
```

## 6. Asynchronous Callback (if applicable)
This API does not have an asynchronous callback - it is itself the callback response to /track.

## 7. State & Correlation
- transaction_id: Must match the transaction_id from the original /track request
- message_id: Must match the message_id from the original /track request
- order_id: Must match the order_id from the /track request
- Correlation: Buyer NP correlates this callback using transaction_id + message_id + order_id

## 8. Validation Rules
- Mandatory field checks: All fields marked as Required=Y in the schema must be present
- Tracking validation: GPS coordinates must be valid format, timestamps must be current
- Path validation: Sequence numbers must be in order, coordinates must form valid path
- Timestamp & TTL validation: context.timestamp must be current time and within TTL window
- Stale request handling: If timestamp is older than acceptable window, Buyer NP may NACK
- Signing verification: Buyer NP must verify Seller NP's signature using registry public key

## 9. Error Scenarios

| Scenario | Error Code | When it occurs | Seller NP Action |
|----------|------------|----------------|------------------|
| Invalid GPS format | 40001 | GPS coordinates malformed | Buyer NP sends NACK |
| Stale tracking data | 40002 | Location data too old | Buyer NP sends NACK |
| Service temporarily unavailable | 50001 | Internal system issues | Buyer NP sends NACK |
| Stale callback | 65003 | Callback sent after TTL expired | Buyer NP sends NACK |

## 10. Important Notes (Seller NP)
- Seller NP MUST provide current GPS coordinates from the active rider
- Seller NP MUST include accurate timestamps for location data
- Seller NP MUST provide path data with sequential coordinates when available
- Seller NP MUST set status to "active" when rider is actively delivering
- Seller NP MAY include tracking URL if applicable (for P2P, GPS-based tracking is used)
- Seller NP MUST ensure GPS coordinates are in latitude,longitude format
- Common mistake to avoid: Sending stale GPS data or incorrect coordinate formats
