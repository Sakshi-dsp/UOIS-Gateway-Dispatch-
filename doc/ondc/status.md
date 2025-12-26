# status API (Seller NP)

## 1. Overview
- Purpose of this API: This API allows Buyer NP to request the current status of an existing logistics order
- Who calls it (Buyer NP / Seller NP): Called by Buyer NP (BAP)
- Sync vs Async behavior: Asynchronous - Buyer NP sends request, Seller NP responds with ACK/NACK synchronously, then sends callback via /on_status
- Callback expectations: Seller NP MUST send /on_status callback with current order status within TTL window
- TTL behavior (if applicable): TTL is specified in context.ttl (default PT30S)

## 2. Role Perspective
We are Seller NP (BPP / Logistics Service Provider). In this API:
- We receive status requests for existing orders
- We validate that the order exists and belongs to the buyer
- We return ACK/NACK synchronously
- We send current order status, fulfillment updates, and proof of delivery asynchronously via /on_status callback
- We must provide accurate, real-time order status information

## 3. Endpoint Details
- HTTP Method: POST
- Endpoint path: /status
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
    "action": "status",
    "core_version": "1.2.0",
    "bap_id": "logistics_buyer.com",
    "bap_uri": "https://logistics_buyer.com/ondc",
    "bpp_id": "lsp.com",
    "bpp_uri": "https://lsp.com/ondc",
    "transaction_id": "T1",
    "message_id": "M7",
    "timestamp": "2023-06-07T00:00:00.000Z",
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
| context.action | string | Y | Action name (status) | Buyer NP |
| context.core_version | string | Y | ONDC core version (1.2.0) | Buyer NP |
| context.bap_id | string | Y | Buyer NP subscriber ID | Buyer NP |
| context.bap_uri | string | Y | Buyer NP callback URI | Buyer NP |
| context.bpp_id | string | Y | Seller NP subscriber ID | Buyer NP |
| context.bpp_uri | string | Y | Seller NP callback URI | Buyer NP |
| context.transaction_id | string | Y | Transaction ID from order creation | Buyer NP |
| context.message_id | string | Y | Unique message identifier | Buyer NP |
| context.timestamp | string | Y | Request timestamp in RFC3339 format | Buyer NP |
| context.ttl | string | Y | Time to live for response (PT30S) | Buyer NP |
| message.order_id | string | Y | Order identifier to check status for | Buyer NP |

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
    "action": "status",
    "core_version": "1.2.0",
    "bap_id": "logistics_buyer.com",
    "bap_uri": "https://logistics_buyer.com/ondc",
    "bpp_id": "lsp.com",
    "bpp_uri": "https://lsp.com/ondc",
    "transaction_id": "T1",
    "message_id": "M7",
    "timestamp": "2023-06-07T00:00:30.000Z"
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
      "code": "66004",
      "message": "Order not found"
    }
  },
  "context": {
    "domain": "nic2004:60232",
    "country": "IND",
    "city": "std:080",
    "action": "status",
    "core_version": "1.2.0",
    "bap_id": "logistics_buyer.com",
    "bap_uri": "https://logistics_buyer.com/ondc",
    "bpp_id": "lsp.com",
    "bpp_uri": "https://lsp.com/ondc",
    "transaction_id": "T1",
    "message_id": "M7",
    "timestamp": "2023-06-07T00:00:30.000Z"
  }
}
```

## 6. Asynchronous Callback (if applicable)

### 6.1 Callback Endpoint
- /on_status

### 6.2 Full Callback Payload Example
```json
{
  "context": {
    "domain": "nic2004:60232",
    "country": "IND",
    "city": "std:080",
    "action": "on_status",
    "core_version": "1.2.0",
    "bap_id": "logistics_buyer.com",
    "bap_uri": "https://logistics_buyer.com/ondc",
    "bpp_id": "lsp.com",
    "bpp_uri": "https://lsp.com/ondc",
    "transaction_id": "T1",
    "message_id": "M7",
    "timestamp": "2023-06-07T00:00:30.000Z"
  },
  "message": {
    "order": {
      "id": "O2",
      "state": "Completed",
      "cancellation": {
        "cancelled_by": "buyerNP.com",
        "reason": {
          "id": "011"
        }
      },
      "provider": {
        "id": "P1",
        "locations": [
          {
            "id": "L1"
          }
        ]
      },
      "items": [
        {
          "id": "I1",
          "fulfillment_id": "1",
          "category_id": "Same Day Delivery",
          "descriptor": {
            "code": "P2P"
          },
          "time": {
            "label": "TAT",
            "duration": "PT45M",
            "timestamp": "2023-06-06"
          }
        },
        {
          "id": "I2",
          "fulfillment_id": "1-RTO",
          "category_id": "Same Day Delivery",
          "descriptor": {
            "code": "P2P"
          },
          "time": {
            "label": "TAT",
            "duration": "PT45M",
            "timestamp": "2023-06-06"
          }
        }
      ],
      "quote": {
        "price": {
          "currency": "INR",
          "value": "108.50"
        },
        "breakup": [
          {
            "@ondc/org/item_id": "I1",
            "@ondc/org/title_type": "delivery",
            "price": {
              "currency": "INR",
              "value": "50.00"
            }
          },
          {
            "@ondc/org/item_id": "I1",
            "@ondc/org/title_type": "tax",
            "price": {
              "currency": "INR",
              "value": "9.00"
            }
          },
          {
            "@ondc/org/item_id": "I1",
            "@ondc/org/title_type": "diff",
            "price": {
              "currency": "INR",
              "value": "25.0"
            }
          },
          {
            "@ondc/org/item_id": "I1",
            "@ondc/org/title_type": "tax_diff",
            "price": {
              "currency": "INR",
              "value": "4.5"
            }
          }
        ]
      },
      "fulfillments": [
        {
          "id": "1",
          "type": "Delivery",
          "@ondc/org/awb_no": "1227262193237777",
          "state": {
            "descriptor": {
              "code": "Order-picked-up",
              "short_desc": "pickup or delivery failed reason code"
            }
          },
          "tracking": false,
          "start": {
            "person": {
              "name": "Ramu"
            },
            "location": {
              "gps": "12.453544,77.928379",
              "address": {
                "name": "name",
                "building": "My house or building name",
                "locality": "My street name",
                "city": "Bengaluru",
                "state": "Karnataka",
                "country": "India",
                "area_code": "560041"
              }
            },
            "contact": {
              "phone": "9886098860",
              "email": "abcd.efgh@gmail.com"
            },
            "time": {
              "duration": "PT15M",
              "range": {
                "start": "2023-06-06T22:30:00.000Z",
                "end": "2023-06-06T22:45:00.000Z"
              },
              "timestamp": "2023-06-06T22:30:00.000Z"
            },
            "instructions": {
              "code": "2",
              "short_desc": "value of PCC",
              "long_desc": "additional instructions for pickup",
              "images": [
                "link to downloadable shipping label",
                "https://lsp.com/pickup_image.png",
                "https://lsp.com/rider_location.png"
              ],
              "additional_desc": {
                "content_type": "text/html",
                "url": "https://reverse_qc_sop_form.htm"
              }
            },
            "authorization": {
              "type": "OTP",
              "token": "OTP code",
              "valid_from": "2023-06-07T12:00:00.000Z",
              "valid_to": "2023-06-07T14:00:00.000Z"
            }
          },
          "end": {
            "person": {
              "name": "person_name"
            },
            "location": {
              "gps": "12.453544,77.928379",
              "address": {
                "name": "My house or building #",
                "building": "My house or building name",
                "locality": "My street name",
                "city": "Bengaluru",
                "state": "Karnataka",
                "country": "India",
                "area_code": "560076"
              }
            },
            "contact": {
              "phone": "9886098860",
              "email": "abcd.efgh@gmail.com"
            },
            "time": {
              "range": {
                "start": "2023-06-06T23:00:00.000Z",
                "end": "2023-06-06T23:15:00.000Z"
              },
              "timestamp": "2023-06-06T23:00:00.000Z"
            },
            "instructions": {
              "code": "3",
              "short_desc": "value of DCC",
              "long_desc": "additional instructions for delivery",
              "images": [
                "https://lsp.com/delivery_image.png",
                "https://lsp.com/rider_location.png"
              ]
            },
            "authorization": {
              "type": "OTP",
              "token": "OTP code",
              "valid_from": "2023-06-07T12:00:00.000Z",
              "valid_to": "2023-06-07T14:00:00.000Z"
            }
          },
          "agent": {
            "name": "agent_name",
            "phone": "9886098860"
          },
          "vehicle": {
            "registration": "3LVJ945"
          },
          "@ondc/org/ewaybillno": "EBN1",
          "@ondc/org/ebnexpirydate": "2023-06-30T12:00:00.000Z",
          "tags": [
            {
              "code": "reverseqc_output",
              "list": [
                {
                  "code": "P001",
                  "value": "Atta"
                },
                {
                  "code": "P003",
                  "value": "1"
                },
                {
                  "code": "Q001",
                  "value": "Y"
                }
              ]
            },
            {
              "code": "fulfillment_delay",
              "list": [
                {
                  "code": "state",
                  "value": "Order-picked-up"
                },
                {
                  "code": "reason_id",
                  "value": "002"
                },
                {
                  "code": "timestamp",
                  "value": "2023-06-06T22:00:00.000Z"
                }
              ]
            },
            {
              "code": "tracking",
              "list": [
                {
                  "code": "gps_enabled",
                  "value": "yes"
                },
                {
                  "code": "url_enabled",
                  "value": "no"
                },
                {
                  "code": "url",
                  "value": "https://sellerNP.com/ondc/tracking_url"
                }
              ]
            }
          ]
        },
        {
          "id": "1-RTO",
          "type": "RTO",
          "state": {
            "descriptor": {
              "code": "RTO-Initiated"
            }
          },
          "start": {
            "time": {
              "range": {
                "start": "2023-06-06T23:00:00.000Z",
                "end": "2023-06-06T23:00:00.000Z"
              },
              "timestamp": "2023-06-06T23:00:00.000Z"
            }
          },
          "agent": {
            "name": "agent_name",
            "phone": "9886098860"
          }
        }
      ],
      "payment": {
        "@ondc/org/collection_amount": "300.00",
        "collected_by": "BPP",
        "type": "POST-FULFILLMENT",
        "status": "PAID",
        "time": {
          "timestamp": "2023-06-07T10:00:00.000Z"
        },
        "@ondc/org/settlement_details": [
          {
            "settlement_counterparty": "buyer-app",
            "settlement_type": "upi",
            "upi_address": "gft@oksbi",
            "settlement_bank_account_no": "XXXXXXXXXX",
            "settlement_ifsc_code": "XXXXXXXXX",
            "settlement_status": "PAID",
            "settlement_reference": "XXXXXXXXX",
            "settlement_timestamp": "2023-02-10T00:00:00.000Z"
          }
        ]
      },
      "billing": {
        "name": "ONDC Seller NP",
        "address": {
          "name": "My building #",
          "building": "My building name",
          "locality": "My street name",
          "city": "Bengaluru",
          "state": "Karnataka",
          "country": "India",
          "area_code": "560076"
        },
        "tax_number": "XXXXXXXXXXXXXXX",
        "phone": "9886098860",
        "email": "abcd.efgh@gmail.com"
      },
      "@ondc/org/linked_order": {
        "items": [
          {
            "category_id": "Grocery",
            "descriptor": {
              "name": "Atta"
            },
            "quantity": {
              "count": 2,
              "measure": {
                "unit": "kilogram",
                "value": 0.5
              }
            },
            "price": {
              "currency": "INR",
              "value": "150.00"
            }
          }
        ],
        "provider": {
          "descriptor": {
            "name": "Aadishwar Store"
          },
          "address": {
            "name": "KHB Towers",
            "building": "Building or House No",
            "street": "6th Block",
            "locality": "Koramangala",
            "city": "Bengaluru",
            "state": "Karnataka",
            "area_code": "560070"
          }
        },
        "order": {
          "id": "O1",
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
          }
        }
      },
      "tags": [
        {
          "code": "diff_dim",
          "list": [
            {
              "code": "unit",
              "value": "centimeter"
            },
            {
              "code": "length",
              "value": "1.5"
            },
            {
              "code": "breadth",
              "value": "1.5"
            },
            {
              "code": "height",
              "value": "1.5"
            }
          ]
        },
        {
          "code": "diff_weight",
          "list": [
            {
              "code": "unit",
              "value": "kilogram"
            },
            {
              "code": "weight",
              "value": "1.5"
            }
          ]
        },
        {
          "code": "diff_proof",
          "list": [
            {
              "code": "type",
              "value": "image"
            },
            {
              "code": "url",
              "value": "https://lsp.com/sorter/images1.png"
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
| context.action | string | Y | Action name (on_status) | Seller NP |
| context.core_version | string | Y | ONDC core version | Seller NP |
| context.bap_id | string | Y | Buyer NP subscriber ID | Seller NP |
| context.bap_uri | string | Y | Buyer NP callback URI | Seller NP |
| context.bpp_id | string | Y | Seller NP subscriber ID | Seller NP |
| context.bpp_uri | string | Y | Seller NP callback URI | Seller NP |
| context.transaction_id | string | Y | Transaction ID from request | Seller NP |
| context.message_id | string | Y | Message ID from request | Seller NP |
| context.timestamp | string | Y | Response timestamp | Seller NP |
| message.order.id | string | Y | Order ID | Seller NP |
| message.order.state | string | Y | Current order state | Seller NP |
| message.order.cancellation | object | N | Cancellation details if cancelled | Seller NP |
| message.order.provider.id | string | Y | Provider ID | Seller NP |
| message.order.provider.locations | array | Y | Provider locations | Seller NP |
| message.order.items | array | Y | Order items | Seller NP |
| message.order.quote | object | Y | Current quote with updates | Seller NP |
| message.order.fulfillments | array | Y | Fulfillment details with current status | Seller NP |
| message.order.payment | object | Y | Payment status and details | Seller NP |
| message.order.billing | object | Y | Billing information | Seller NP |
| message.order.@ondc/org/linked_order | object | Y | Linked retail order details | Seller NP |
| message.order.tags | array | N | Additional order tags | Seller NP |

## 7. State & Correlation
- transaction_id: Must match the transaction_id from the original order creation
- message_id: Unique identifier for this status request
- order_id: The order identifier to check status for
- Correlation: Seller NP correlates the callback using transaction_id + message_id + order_id

## 8. Validation Rules
- Mandatory field checks: order_id must be present and valid
- Order validation: Order must exist and belong to the requesting buyer NP
- Timestamp & TTL validation: context.timestamp must be current time, response within TTL
- Stale request handling: If timestamp too old, respond with NACK error code 65003
- Signing verification: Verify Buyer's signature using registry public key

## 9. Error Scenarios

| Scenario | Error Code | When it occurs | Seller NP Action |
|----------|------------|----------------|------------------|
| Order not found | 66004 | Order doesn't exist or doesn't belong to buyer | Send NACK with LSP-ERROR |
| Invalid order ID | 40001 | Order ID format invalid | Send NACK with DOMAIN-ERROR |
| Service temporarily unavailable | 50001 | Internal system issues | Send NACK with LSP-ERROR |
| Stale request | 65003 | Request timestamp too old | Send NACK with PROTOCOL-ERROR |
| Invalid signature | 401 | Signature verification fails | Return HTTP 401 |

## 10. Important Notes (Seller NP)
- Seller NP MUST validate that the order belongs to the requesting buyer NP
- Seller NP MUST provide current, accurate order and fulfillment status
- Seller NP MUST include proof of pickup/delivery images when available
- Seller NP MUST update fulfillment timestamps accurately
- Seller NP SHOULD include authorization tokens for pickup/delivery if applicable
- Seller NP MUST include settlement details if payment has been processed
- Common mistake to avoid: Sending stale status information or missing fulfillment state updates
