# search API (Seller NP)

## 1. Overview
- Purpose of this API: This API allows Buyer NP to specify search intent for logistics services including fulfillment type, locations, package details, and payment preferences
- Who calls it (Buyer NP / Seller NP): Called by Buyer NP (BAP)
- Sync vs Async behavior: Asynchronous - Buyer NP sends request, Seller NP responds with ACK/NACK synchronously, then sends callback via /on_search
- Callback expectations: Seller NP MUST send /on_search callback with catalog data within TTL window
- TTL behavior (if applicable): TTL is specified in context.ttl (default PT30S)

## 2. Role Perspective
We are Seller NP (BPP / Logistics Service Provider). In this API:
- We receive the buyer's search intent
- We validate request schema, signature, and TTL synchronously
- We return ACK/NACK immediately (within 1 second) based on syntactic validation only
- We publish SEARCH_REQUESTED event after ACK
- We perform serviceability check asynchronously via Location Service
- We send catalog response asynchronously via /on_search callback within TTL window

## 3. Endpoint Details
- HTTP Method: POST
- Endpoint path: /search
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
    "action": "search",
    "core_version": "1.2.0",
    "bap_id": "logistics_buyer.com",
    "bap_uri": "https://logistics_buyer.com/ondc",
    "transaction_id": "T1",
    "message_id": "M1",
    "timestamp": "2023-06-06T21:00:00.000Z",
    "ttl": "PT30S"
  },
  "message": {
    "intent": {
      "category": {
        "id": "Immediate Delivery"
      },
      "provider": {
        "time": {
          "days": "1,2,3,4,5,6,7",
          "schedule": {
            "holidays": [
              "2023-06-29",
              "2023-08-15"
            ]
          },
          "duration": "PT30M",
          "range": {
            "start": "1100",
            "end": "2100"
          }
        }
      },
      "fulfillment": {
        "type": "Delivery",
        "start": {
          "location": {
            "gps": "12.453544,77.928379",
            "address": {
              "area_code": "560041"
            }
          },
          "authorization": {
            "type": "OTP"
          }
        },
        "end": {
          "location": {
            "gps": "12.453544,77.928379",
            "address": {
              "area_code": "560001"
            }
          },
          "authorization": {
            "type": "OTP"
          }
        }
      },
      "payment": {
        "type": "POST-FULFILLMENT",
        "@ondc/org/collection_amount": "300.00"
      },
      "@ondc/org/payload_details": {
        "weight": {
          "unit": "kilogram",
          "value": 1
        },
        "dimensions": {
          "length": {
            "unit": "centimeter",
            "value": 1
          },
          "breadth": {
            "unit": "centimeter",
            "value": 1
          },
          "height": {
            "unit": "centimeter",
            "value": 1
          }
        },
        "category": "Grocery",
        "value": {
          "currency": "INR",
          "value": "300.00"
        },
        "dangerous_goods": false
      }
    }
  }
}
```

### 4.2 Field-Level Schema (Table)

| Field Path | Data Type | Required (Y/N) | Description | Source of Truth |
|------------|-----------|----------------|-------------|-----------------|
| context.domain | string | Y | Domain identifier for logistics (nic2004:60232) | Buyer NP |
| context.country | string | Y | Country code (IND) | Buyer NP |
| context.city | string | Y | City code (std:080) | Buyer NP |
| context.action | string | Y | Action name (search) | Buyer NP |
| context.core_version | string | Y | ONDC core version (1.2.0) | Buyer NP |
| context.bap_id | string | Y | Buyer NP subscriber ID | Buyer NP |
| context.bap_uri | string | Y | Buyer NP callback URI | Buyer NP |
| context.transaction_id | string | Y | Unique transaction identifier | Buyer NP |
| context.message_id | string | Y | Unique message identifier | Buyer NP |
| context.timestamp | string | Y | Request timestamp in RFC3339 format | Buyer NP |
| context.ttl | string | Y | Time to live for response (PT30S) | Buyer NP |
| message.intent.category.id | string | Y | Delivery category (Immediate Delivery, Same Day Delivery, etc.) | Buyer NP |
| message.intent.provider.time.days | string | Y | Operating days (comma-separated: 1,2,3,4,5,6,7) | Buyer NP |
| message.intent.provider.time.schedule.holidays | array | N | List of holiday dates | Buyer NP |
| message.intent.provider.time.duration | string | N | Order preparation time (ISO8601 duration) | Buyer NP |
| message.intent.provider.time.range.start | string | Y | Start time in HHMM format | Buyer NP |
| message.intent.provider.time.range.end | string | Y | End time in HHMM format | Buyer NP |
| message.intent.fulfillment.type | string | Y | Fulfillment type (Delivery/Return) | Buyer NP |
| message.intent.fulfillment.start.location.gps | string | Y | Pickup GPS coordinates (latitude,longitude) | Buyer NP |
| message.intent.fulfillment.start.location.address.area_code | string | Y | Pickup area pincode | Buyer NP |
| message.intent.fulfillment.start.authorization.type | string | N | Pickup authorization type (OTP) | Buyer NP |
| message.intent.fulfillment.end.location.gps | string | Y | Dropoff GPS coordinates (latitude,longitude) | Buyer NP |
| message.intent.fulfillment.end.location.address.area_code | string | Y | Dropoff area pincode | Buyer NP |
| message.intent.fulfillment.end.authorization.type | string | N | Dropoff authorization type (OTP) | Buyer NP |
| message.intent.payment.type | string | Y | Payment type (ON-ORDER/ON-FULFILLMENT/POST-FULFILLMENT) | Buyer NP |
| message.intent.payment.@ondc/org/collection_amount | string | N | Collection amount for ON-FULFILLMENT | Buyer NP |
| message.intent.@ondc/org/payload_details.weight.unit | string | Y | Weight unit (kilogram/gram) | Buyer NP |
| message.intent.@ondc/org/payload_details.weight.value | number | Y | Package weight value | Buyer NP |
| message.intent.@ondc/org/payload_details.dimensions.length.unit | string | N | Length unit (centimeter) | Buyer NP |
| message.intent.@ondc/org/payload_details.dimensions.length.value | number | N | Package length | Buyer NP |
| message.intent.@ondc/org/payload_details.dimensions.breadth.unit | string | N | Breadth unit (centimeter) | Buyer NP |
| message.intent.@ondc/org/payload_details.dimensions.breadth.value | number | N | Package breadth | Buyer NP |
| message.intent.@ondc/org/payload_details.dimensions.height.unit | string | N | Height unit (centimeter) | Buyer NP |
| message.intent.@ondc/org/payload_details.dimensions.height.value | number | N | Package height | Buyer NP |
| message.intent.@ondc/org/payload_details.category | string | Y | Package category (Grocery, Fashion, etc.) | Buyer NP |
| message.intent.@ondc/org/payload_details.value.currency | string | Y | Currency code (INR) | Buyer NP |
| message.intent.@ondc/org/payload_details.value.value | string | Y | Package value | Buyer NP |
| message.intent.@ondc/org/payload_details.dangerous_goods | boolean | N | Whether package contains dangerous goods | Buyer NP |

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
    "action": "search",
    "core_version": "1.2.0",
    "bap_id": "logistics_buyer.com",
    "bap_uri": "https://logistics_buyer.com/ondc",
    "bpp_id": "lsp.com",
    "bpp_uri": "https://lsp.com/ondc",
    "transaction_id": "T1",
    "message_id": "M1",
    "timestamp": "2023-06-06T21:00:30.000Z"
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
      "message": "Invalid request"
    }
  },
  "context": {
    "domain": "nic2004:60232",
    "country": "IND",
    "city": "std:080",
    "action": "search",
    "core_version": "1.2.0",
    "bap_id": "logistics_buyer.com",
    "bap_uri": "https://logistics_buyer.com/ondc",
    "bpp_id": "lsp.com",
    "bpp_uri": "https://lsp.com/ondc",
    "transaction_id": "T1",
    "message_id": "M1",
    "timestamp": "2023-06-06T21:00:30.000Z"
  }
}
```

## 6. Asynchronous Callback (if applicable)

### 6.1 Callback Endpoint
- /on_search

### 6.2 Full Callback Payload Example
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

### 6.3 Callback Field Schema
| Field Path | Data Type | Required (Y/N) | Description | Source |
|------------|-----------|----------------|-------------|--------|
| context.domain | string | Y | Domain identifier for logistics | Seller NP |
| context.country | string | Y | Country code | Seller NP |
| context.city | string | Y | City code | Seller NP |
| context.action | string | Y | Action name (on_search) | Seller NP |
| context.core_version | string | Y | ONDC core version | Seller NP |
| context.bap_id | string | Y | Buyer NP subscriber ID | Seller NP |
| context.bap_uri | string | Y | Buyer NP callback URI | Seller NP |
| context.bpp_id | string | Y | Seller NP subscriber ID | Seller NP |
| context.bpp_uri | string | Y | Seller NP callback URI | Seller NP |
| context.transaction_id | string | Y | Transaction ID from request | Seller NP |
| context.message_id | string | Y | Message ID from request | Seller NP |
| context.timestamp | string | Y | Response timestamp | Seller NP |
| message.catalog.bpp/descriptor.name | string | Y | LSP name | Seller NP |
| message.catalog.bpp/descriptor.tags | array | Y | Static terms information | Seller NP |
| message.catalog.bpp/providers[].id | string | Y | Provider unique identifier | Seller NP |
| message.catalog.bpp/providers[].descriptor.name | string | Y | Provider display name | Seller NP |
| message.catalog.bpp/providers[].descriptor.short_desc | string | Y | Provider short description | Seller NP |
| message.catalog.bpp/providers[].descriptor.long_desc | string | Y | Provider long description | Seller NP |
| message.catalog.bpp/providers[].categories[].id | string | Y | Delivery category | Seller NP |
| message.catalog.bpp/providers[].categories[].time.label | string | Y | Time label (TAT) | Seller NP |
| message.catalog.bpp/providers[].categories[].time.duration | string | Y | TAT duration (ISO8601) | Seller NP |
| message.catalog.bpp/providers[].categories[].time.timestamp | string | N | TAT date | Seller NP |
| message.catalog.bpp/providers[].fulfillments[].id | string | Y | Fulfillment identifier | Seller NP |
| message.catalog.bpp/providers[].fulfillments[].type | string | Y | Fulfillment type (Delivery/RTO) | Seller NP |
| message.catalog.bpp/providers[].fulfillments[].start.time.duration | string | N | Pickup SLA duration for P2P delivery (part of end-to-end TAT) | Seller NP |
| message.catalog.bpp/providers[].fulfillments[].tags | array | N | Distance and other fulfillment tags | Seller NP |
| message.catalog.bpp/providers[].locations[].id | string | N | Location identifier (not required for P2P, only for drop-off at LSP location) | Seller NP |
| message.catalog.bpp/providers[].locations[].gps | string | N | GPS coordinates (not required for P2P) | Seller NP |
| message.catalog.bpp/providers[].locations[].address.street | string | N | Street address (not required for P2P) | Seller NP |
| message.catalog.bpp/providers[].locations[].address.city | string | N | City name (not required for P2P) | Seller NP |
| message.catalog.bpp/providers[].locations[].address.area_code | string | N | Area pincode (not required for P2P) | Seller NP |
| message.catalog.bpp/providers[].locations[].address.state | string | N | State code (not required for P2P) | Seller NP |
| message.catalog.bpp/providers[].items[].id | string | Y | Item unique identifier | Seller NP |
| message.catalog.bpp/providers[].items[].parent_item_id | string | Y | Parent item ID (empty for main items) | Seller NP |
| message.catalog.bpp/providers[].items[].category_id | string | Y | Category identifier | Seller NP |
| message.catalog.bpp/providers[].items[].fulfillment_id | string | Y | Associated fulfillment ID | Seller NP |
| message.catalog.bpp/providers[].items[].descriptor.code | string | Y | Item code (P2P) | Seller NP |
| message.catalog.bpp/providers[].items[].descriptor.name | string | Y | Item display name | Seller NP |
| message.catalog.bpp/providers[].items[].descriptor.short_desc | string | Y | Item short description | Seller NP |
| message.catalog.bpp/providers[].items[].descriptor.long_desc | string | Y | Item long description | Seller NP |
| message.catalog.bpp/providers[].items[].price.currency | string | Y | Currency code | Seller NP |
| message.catalog.bpp/providers[].items[].price.value | string | Y | Item price (tax inclusive) | Seller NP |
| message.catalog.bpp/providers[].items[].time.label | string | Y | Time label | Seller NP |
| message.catalog.bpp/providers[].items[].time.duration | string | Y | Item TAT duration | Seller NP |
| message.catalog.bpp/providers[].items[].time.timestamp | string | N | Item TAT date | Seller NP |

## 7. State & Correlation
- transaction_id: Unique identifier for the entire search transaction, set by Buyer NP. Seller NP must use the same transaction_id in ACK, NACK, and callback responses
- message_id: Unique identifier for this specific search request message, set by Buyer NP. Seller NP must echo this in ACK/NACK responses
- Correlation: Seller NP correlates the callback to the original request using transaction_id + message_id. The callback should be sent to bap_uri endpoint
- No order_id exists at this stage as no order has been created yet

## 8. Validation Rules

### 8.1 Pre-ACK Validation (Synchronous, < 1 second)
- Mandatory field checks: All fields marked as Required=Y in the schema must be present
- Enum validations: category.id must be valid delivery categories, fulfillment.type must be Delivery/Return, payment.type must be valid payment types
- Timestamp & TTL validation: context.timestamp must be current time, response must be sent within TTL window (PT30S)
- Stale request handling: If timestamp is older than acceptable window, respond with NACK error code 65003
- Signing verification: Verify request signature using Buyer's public key from registry
- Payload validation: Validate GPS coordinates format, pincode validity, weight/dimensions ranges

### 8.2 Post-ACK Validation (Asynchronous)
- Serviceability validation: Check if pickup and dropoff locations are serviceable via Location Service
- Distance calculation: Calculate route distance for P2P pricing (performed during serviceability check, motorable distance optional)
- Pricing computation: Compute quote prices based on serviceability and distance (performed by Quote Service)

## 9. Error Scenarios

| Scenario | Error Code | When it occurs | Seller NP Action |
|----------|------------|----------------|------------------|
| Invalid request payload | 40001 | JSON schema validation fails | Send NACK with DOMAIN-ERROR |
| Invalid signature | 401 | Signature verification fails | Return HTTP 401 (before ACK) |
| Stale request | 65003 | Request timestamp too old | Send NACK with PROTOCOL-ERROR |
| Rate limit exceeded | 429 | Too many requests | Return HTTP 429 (before ACK) |
| Service not available | 50001 | Locations not serviceable (after ACK) | Send /on_search with empty catalog OR error in callback |
| Internal server error | 50002 | Unexpected system failure (after ACK) | Send /on_search with error in callback |

## 10. Important Notes (Seller NP)

### Seller NP MUST:
- Validate request schema, signature, TTL before ACK
- Return ACK within 1 second (synchronous response)
- Publish SEARCH_REQUESTED event after ACK
- Perform serviceability asynchronously via Location Service
- Use Location Service for serviceability check (distance calculation is internal and optional for P2P pricing)
- Send /on_search callback within TTL window
- Include both forward delivery items (fulfillment_id: "1") and RTO items (fulfillment_id: "2") in catalog when serviceable
- Include static terms in bpp/descriptor/tags with current and new version links
- Use transaction_id + message_id for correlation between request and callback

### Seller NP MUST NOT:
- Block ACK on serviceability computation
- Perform pricing computation before ACK (distance calculation is internal and optional)
- NACK valid requests due to non-serviceability (use empty catalog or error in /on_search instead)
- Send catalog after TTL expiry

### Common mistakes to avoid:
- Not validating GPS coordinates format or pincode validity before ACK
- Waiting for serviceability check before sending ACK
- Sending NACK for non-serviceable locations (should send empty catalog or error in callback)
