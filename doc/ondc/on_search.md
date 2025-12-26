# on_search API (Seller NP)

## 1. Overview
- Purpose of this API: This API allows Seller NP to provide catalog response with available logistics services based on Buyer's search intent
- Who calls it (Buyer NP / Seller NP): Called by Seller NP (BPP) as callback response to /search
- Sync vs Async behavior: Asynchronous - This is the callback response sent after ACK to /search request
- Callback expectations: Buyer NP acknowledges receipt with ACK/NACK
- TTL behavior (if applicable): Response must be sent within TTL specified in original /search request

## 2. Role Perspective
We are Seller NP (BPP / Logistics Service Provider). In this API:
- We provide our catalog of logistics services that match the buyer's search intent
- We include pricing, TAT, and static terms
- We send this as asynchronous callback to the buyer's /on_search endpoint
- We must ensure all catalog data complies with ONDC standards
- We are responsible for accurate pricing representation
- Note: Serviceability is determined at /search stage; /on_search is only sent if locations are serviceable

## 3. Endpoint Details
- HTTP Method: POST
- Endpoint path: /on_search (at Buyer NP's bap_uri)
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
    "action": "on_search",
    "core_version": "1.2.0",
    "bap_id": "logistics_buyer.com",
    "bap_uri": "https://logistics_buyer.com/ondc",
    "bpp_id": "lsp.com",
    "bpp_uri": "https://lsp.com/ondc",
    "transaction_id": "T1",
    "message_id": "M1",
    "timestamp": "2023-06-06T21:00:30.000Z"
  },
  "message": {
    "catalog": {
      "bpp/descriptor": {
        "name": "LSP Aggregator Inc",
        "tags": [
          {
            "code": "bpp_terms",
            "list": [
              {
                "code": "static_terms",
                "value": ""
              },
              {
                "code": "static_terms_new",
                "value": "https://github.com/ONDC-Official/NP-Static-Terms/lspNP_LSP/1.0/tc.pdf"
              },
              {
                "code": "effective_date",
                "value": "2023-10-01T00:00:00.000Z"
              }
            ]
          }
        ]
      },
      "bpp/providers": [
        {
          "id": "P1",
          "descriptor": {
            "name": "LSP Courier Inc",
            "short_desc": "LSP Courier Inc",
            "long_desc": "LSP Courier Inc"
          },
          "categories": [
            {
              "id": "Immediate Delivery",
              "time": {
                "label": "TAT",
                "duration": "PT60M",
                "timestamp": "2023-06-06"
              }
            }
          ],
          "fulfillments": [
            {
              "id": "1",
              "type": "Delivery",
              "start": {
                "time": {
                  "duration": "PT15M"
                }
              },
              "tags": [
                {
                  "code": "distance",
                  "list": [
                    {
                      "code": "motorable_distance_type",
                      "value": "kilometer"
                    },
                    {
                      "code": "motorable_distance",
                      "value": "1.8"
                    }
                  ]
                }
              ]
            },
            {
              "id": "2",
              "type": "RTO"
            }
          ],
          "locations": [
            {
              "id": "L1",
              "gps": "12.967555,77.749666",
              "address": {
                "street": "Jayanagar 4th Block",
                "city": "Bengaluru",
                "area_code": "560076",
                "state": "KA"
              }
            }
          ],
          "items": [
            {
              "id": "I1",
              "parent_item_id": "",
              "category_id": "Immediate Delivery",
              "fulfillment_id": "1",
              "descriptor": {
                "code": "P2P",
                "name": "60 min delivery",
                "short_desc": "60 min delivery for F&B",
                "long_desc": "60 min delivery for F&B"
              },
              "price": {
                "currency": "INR",
                "value": "59.00"
              },
              "time": {
                "label": "TAT",
                "duration": "PT45M",
                "timestamp": "2023-06-06"
              }
            },
            {
              "id": "I2",
              "parent_item_id": "I1",
              "category_id": "Immediate Delivery",
              "fulfillment_id": "2",
              "descriptor": {
                "code": "P2P",
                "name": "RTO quote",
                "short_desc": "RTO quote",
                "long_desc": "RTO quote"
              },
              "price": {
                "currency": "INR",
                "value": "23.60"
              },
              "time": {
                "label": "TAT",
                "duration": "PT60M",
                "timestamp": "2023-06-06"
              }
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
| context.domain | string | Y | Domain identifier for logistics (nic2004:60232) | Seller NP |
| context.country | string | Y | Country code (IND) | Seller NP |
| context.city | string | Y | City code (std:080) | Seller NP |
| context.action | string | Y | Action name (on_search) | Seller NP |
| context.core_version | string | Y | ONDC core version (1.2.0) | Seller NP |
| context.bap_id | string | Y | Buyer NP subscriber ID | Seller NP |
| context.bap_uri | string | Y | Buyer NP callback URI | Seller NP |
| context.bpp_id | string | Y | Seller NP subscriber ID | Seller NP |
| context.bpp_uri | string | Y | Seller NP callback URI | Seller NP |
| context.transaction_id | string | Y | Transaction ID from original search request | Seller NP |
| context.message_id | string | Y | Message ID from original search request | Seller NP |
| context.timestamp | string | Y | Response timestamp in RFC3339 format | Seller NP |
| message.catalog.bpp/descriptor.name | string | Y | LSP aggregator/provider name | Seller NP |
| message.catalog.bpp/descriptor.tags | array | Y | Static terms and compliance information | Seller NP |
| message.catalog.bpp/providers[].id | string | Y | Unique provider identifier | Seller NP |
| message.catalog.bpp/providers[].descriptor.name | string | Y | Provider display name | Seller NP |
| message.catalog.bpp/providers[].descriptor.short_desc | string | Y | Provider short description | Seller NP |
| message.catalog.bpp/providers[].descriptor.long_desc | string | Y | Provider detailed description | Seller NP |
| message.catalog.bpp/providers[].categories[].id | string | Y | Delivery category (Immediate Delivery, etc.) | Seller NP |
| message.catalog.bpp/providers[].categories[].time.label | string | Y | Time label (TAT) | Seller NP |
| message.catalog.bpp/providers[].categories[].time.duration | string | Y | Category TAT duration (ISO8601) | Seller NP |
| message.catalog.bpp/providers[].categories[].time.timestamp | string | N | Category TAT date | Seller NP |
| message.catalog.bpp/providers[].fulfillments[].id | string | Y | Fulfillment unique identifier | Seller NP |
| message.catalog.bpp/providers[].fulfillments[].type | string | Y | Fulfillment type (Delivery/RTO) | Seller NP |
| message.catalog.bpp/providers[].fulfillments[].start.time.duration | string | N | Dispatch or handover preparation time for P2P (if applicable) | Seller NP |
| message.catalog.bpp/providers[].fulfillments[].tags | array | N | Distance and other fulfillment metadata (motorable distance optional for P2P) | Seller NP |
| message.catalog.bpp/providers[].locations[].id | string | N | Location identifier (not required for P2P, only for drop-off at LSP location) | Seller NP |
| message.catalog.bpp/providers[].locations[].gps | string | N | GPS coordinates (not required for P2P) | Seller NP |
| message.catalog.bpp/providers[].locations[].address.street | string | N | Street address (not required for P2P) | Seller NP |
| message.catalog.bpp/providers[].locations[].address.city | string | N | City name (not required for P2P) | Seller NP |
| message.catalog.bpp/providers[].locations[].address.area_code | string | N | Area pincode (not required for P2P) | Seller NP |
| message.catalog.bpp/providers[].locations[].address.state | string | N | State code (not required for P2P) | Seller NP |
| message.catalog.bpp/providers[].items[].id | string | Y | Item unique identifier | Seller NP |
| message.catalog.bpp/providers[].items[].parent_item_id | string | Y | Parent item ID (empty for main items) | Seller NP |
| message.catalog.bpp/providers[].items[].category_id | string | Y | Associated category | Seller NP |
| message.catalog.bpp/providers[].items[].fulfillment_id | string | Y | Associated fulfillment | Seller NP |
| message.catalog.bpp/providers[].items[].descriptor.code | string | Y | Item code (P2P) | Seller NP |
| message.catalog.bpp/providers[].items[].descriptor.name | string | Y | Item display name | Seller NP |
| message.catalog.bpp/providers[].items[].descriptor.short_desc | string | Y | Item short description | Seller NP |
| message.catalog.bpp/providers[].items[].descriptor.long_desc | string | Y | Item detailed description | Seller NP |
| message.catalog.bpp/providers[].items[].price.currency | string | Y | Currency code (INR) | Seller NP |
| message.catalog.bpp/providers[].items[].price.value | string | Y | Item price (tax inclusive) | Seller NP |
| message.catalog.bpp/providers[].items[].time.label | string | Y | Time label | Seller NP |
| message.catalog.bpp/providers[].items[].time.duration | string | Y | Item TAT duration | Seller NP |
| message.catalog.bpp/providers[].items[].time.timestamp | string | N | Item TAT date | Seller NP |

## 5. Synchronous Response (ACK/NACK)

### 5.1 ACK Example
```json
{
  "message": {
    "ack": {
      "status": "ACK",
      "tags": [
        {
          "code": "bap_terms",
          "list": [
            {
              "code": "accept_bpp_terms",
              "value": "Y"
            }
          ]
        }
      ]
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
      "message": "Invalid catalog format"
    }
  }
}
```

## 6. Asynchronous Callback (if applicable)
This API does not have an asynchronous callback - it is itself the callback response to /search.

## 7. State & Correlation
- transaction_id: Must match the transaction_id from the original /search request
- message_id: Must match the message_id from the original /search request
- Correlation: Buyer NP correlates this callback to the original search using transaction_id + message_id
- No order_id exists at this stage as no order has been created yet

## 8. Validation Rules
- Mandatory field checks: All fields marked as Required=Y in the schema must be present
- Enum validations: fulfillment.type must be Delivery/RTO, descriptor.code must be P2P
- Timestamp & TTL validation: context.timestamp must be valid RFC3339 format and within TTL window from original request
- Stale request handling: If timestamp is older than acceptable window, Buyer NP may NACK with error code 65003
- Signing verification: Buyer NP must verify Seller NP's signature using public key from registry
- Catalog validation: Items must have valid pricing, TAT must be reasonable, locations must be properly formatted
- Terms validation: Static terms links must be valid and accessible

## 9. Error Scenarios

| Scenario | Error Code | When it occurs | Buyer NP Action |
|----------|------------|----------------|------------------|
| Invalid catalog structure | 40001 | JSON schema validation fails | Buyer NP sends NACK |
| Incompatible terms | 62505 | Buyer NP doesn't accept LSP terms | Buyer NP sends NACK |
| Service temporarily unavailable | 50001 | Internal LSP system issues | Buyer NP sends NACK |
| Invalid pricing | 40002 | Price format or values invalid | Buyer NP sends NACK |
| Stale callback | 65003 | Callback sent after TTL expired | Buyer NP sends NACK |

## 10. Important Notes (Seller NP)
- Seller NP MUST send catalog directly to Buyer NP via /on_search callback, not through ONDC network
- Seller NP MUST include both current and new static terms links in bpp/descriptor/tags
- Seller NP MUST include RTO items alongside forward delivery items
- Seller NP MUST ensure all pricing is tax-inclusive in /on_search (tax itemization appears in /on_init)
- Seller NP MUST ensure catalog reflects serviceability results (serviceability is determined at /search stage)
- Seller NP MAY provide motorable distance for P2P pricing (optional, distance computation is internal and not required to be surfaced in payload)
- Seller NP MAY include dispatch or handover preparation time if applicable
- Seller NP MUST ensure catalog matches the original search intent before sending
- Common mistake to avoid: Sending incomplete item information or catalog that doesn't match search intent
