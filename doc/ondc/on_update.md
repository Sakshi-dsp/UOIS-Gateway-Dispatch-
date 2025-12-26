# on_update API (Seller NP)

## 1. Overview
- Purpose of this API: This API allows Seller NP to confirm order updates and provide updated order details including differential charges if applicable
- Who calls it (Buyer NP / Seller NP): Called by Seller NP (BPP) as callback response to /update
- Sync vs Async behavior: Asynchronous - This is the callback response sent after ACK to /update request
- Callback expectations: Buyer NP acknowledges receipt with ACK/NACK
- TTL behavior (if applicable): Response must be sent within TTL specified in original /update request

## 2. Role Perspective
We are Seller NP (BPP / Logistics Service Provider). In this API:
- We confirm the order update has been processed
- We provide updated order details and fulfillment status
- We include differential charges if weight/dimensions changed
- We provide proof of changes and updated schedules
- We send this as asynchronous callback to the buyer's /on_update endpoint
- We must ensure all update information is accurate and complete

## 3. Endpoint Details
- HTTP Method: POST
- Endpoint path: /on_update (at Buyer NP's bap_uri)
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
    "action": "on_update",
    "core_version": "1.2.0",
    "bap_id": "logistics_buyer.com",
    "bap_uri": "https://logistics_buyer.com/ondc",
    "bpp_id": "lsp.com",
    "bpp_uri": "https://lsp.com/ondc",
    "transaction_id": "T1",
    "message_id": "M4",
    "timestamp": "2023-06-06T22:30:30.000Z"
  },
  "message": {
    "order": {
      "id": "O2",
      "state": "In-progress",
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
        }
      ],
      "quote": {
        "price": {
          "currency": "INR",
          "value": "88.50"
        },
        "breakup": [
          {
            "@ondc/org/item_id": "I1",
            "@ondc/org/title_type": "delivery",
            "price": {
              "currency": "INR",
              "value": "50.0"
            }
          },
          {
            "@ondc/org/item_id": "I1",
            "@ondc/org/title_type": "tax",
            "price": {
              "currency": "INR",
              "value": "9.0"
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
          "state": {
            "descriptor": {
              "code": "Order-picked-up"
            }
          },
          "@ondc/org/awb_no": "1227262193237777",
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
            "instructions": {
              "code": "2",
              "short_desc": "value of PCC",
              "long_desc": "additional instructions for pickup",
              "images": [
                "link to downloadable shipping label (optional for P2P)",
                "https://lsp.com/pickup_image.png",
                "https://lsp.com/rider_location.png"
              ],
              "additional_desc": {
                "content_type": "text/html",
                "url": "https://reverse_qc_sop_form.htm"
              }
            },
            "time": {
              "duration": "PT15M",
              "range": {
                "start": "2023-06-06T23:45:00.000Z",
                "end": "2023-06-07T00:00:00.000Z"
              },
              "timestamp": "2023-06-07T00:00:00.000Z"
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
            "instructions": {
              "code": "3",
              "short_desc": "value of DCC",
              "long_desc": "additional instructions for delivery",
              "images": [
                "https://lsp.com/delivery_image.png",
                "https://lsp.com/rider_location.png"
              ]
            },
            "time": {
              "range": {
                "start": "2023-06-07T02:00:00.000Z",
                "end": "2023-06-07T02:15:00.000Z"
              }
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
          "@ondc/org/ebnexpirydate": "2023-06-30T12:00:00.000Z"
        }
      ],
      "billing": {
        "name": "ONDC sellerNP",
        "address": {
          "name": "My building no",
          "building": "My building name",
          "locality": "My street name",
          "city": "Bengaluru",
          "state": "Karnataka",
          "country": "India",
          "area_code": "560076"
        },
        "tax_number": "XXXXXXXXXXXXXXX",
        "phone": "9886098860",
        "email": "abcd.efgh@gmail.com",
        "created_at": "2023-06-06T21:30:00.000Z",
        "updated_at": "2023-06-06T21:30:00.000Z"
      },
      "payment": {
        "@ondc/org/collection_amount": "300.00",
        "collected_by": "BPP",
        "type": "ON-FULFILLMENT",
        "@ondc/org/settlement_details": [
          {
            "settlement_counterparty": "buyer-app",
            "settlement_type": "upi",
            "upi_address": "gft@oksbi",
            "settlement_bank_account_no": "XXXXXXXXXX",
            "settlement_ifsc_code": "XXXXXXXXX"
          }
        ]
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
      ],
      "updated_at": "2023-06-07T23:00:30.000Z"
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
| context.action | string | Y | Action name (on_update) | Seller NP |
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
| message.order.provider | object | Y | Provider information | Seller NP |
| message.order.items | array | Y | Order items | Seller NP |
| message.order.quote | object | Y | Updated quote (may include differential charges) | Seller NP |
| message.order.fulfillments | array | Y | Updated fulfillment details with current status | Seller NP |
| message.order.billing | object | Y | Billing information | Seller NP |
| message.order.payment | object | Y | Payment information | Seller NP |
| message.order.@ondc/org/linked_order | object | Y | Linked order details | Seller NP |
| message.order.tags | array | N | Order-level tags (differential charges, etc.) | Seller NP |
| message.order.updated_at | string | Y | Order update timestamp | Seller NP |

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
      "code": "62504",
      "message": "Differential charges not acceptable"
    }
  }
}
```

## 6. Asynchronous Callback (if applicable)
This API does not have an asynchronous callback - it is itself the callback response to /update.

## 7. State & Correlation
- transaction_id: Must match the transaction_id from the original /update request
- message_id: Must match the message_id from the original /update request
- order_id: Must match the order_id from the /update request
- Correlation: Buyer NP correlates this callback using transaction_id + message_id + order_id

## 8. Validation Rules
- Mandatory field checks: All fields marked as Required=Y in the schema must be present
- Update validation: Order details must reflect the processed update
- Differential charges validation: If included, must be reasonable and accurately calculated
- Timestamp & TTL validation: context.timestamp must be current time and within TTL window
- Stale request handling: If timestamp is older than acceptable window, Buyer NP may NACK
- Signing verification: Buyer NP must verify Seller NP's signature using registry public key

## 9. Error Scenarios

| Scenario | Error Code | When it occurs | Seller NP Action |
|----------|------------|----------------|------------------|
| Differential charges not acceptable | 62504 | Buyer NP doesn't accept additional charges | Buyer NP sends NACK |
| Invalid quote breakup | 40001 | Quote structure invalid | Buyer NP sends NACK |
| Service temporarily unavailable | 50001 | Internal system issues | Buyer NP sends NACK |
| Stale callback | 65003 | Callback sent after TTL expired | Buyer NP sends NACK |

## 10. Important Notes (Seller NP)
- Seller NP MUST include differential charges in quote breakup when weight/dimensions differ
- Seller NP MUST provide proof images for weight/dimension differences
- Seller NP MUST update fulfillment state accurately after processing updates
- Seller NP MUST include updated time ranges and agent assignments
- Seller NP SHOULD include diff_dim, diff_weight, and diff_proof tags when applicable
- Seller NP MUST ensure quote.total matches sum of all breakup items
- Common mistake to avoid: Sending incorrect differential calculations or missing proof documentation
