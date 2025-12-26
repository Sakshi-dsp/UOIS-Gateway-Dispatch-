# on_cancel API (Seller NP)

## 1. Overview
- Purpose of this API: This API allows Seller NP to communicate order cancellation confirmation and handle RTO scenarios
- Who calls it (Buyer NP / Seller NP): Called by Seller NP (BPP) as callback response to /cancel, or unsolicited when LSP initiates cancellation
- Sync vs Async behavior: Asynchronous - This is the callback response sent after ACK to /cancel request, or unsolicited for LSP-initiated cancellations
- Callback expectations: Buyer NP acknowledges receipt with ACK/NACK
- TTL behavior (if applicable): Response must be sent within TTL specified in original /cancel request (for solicited), or reasonable time for unsolicited

## 2. Role Perspective
We are Seller NP (BPP / Logistics Service Provider). In this API:
- We confirm order cancellation and provide final order details
- We handle RTO scenarios by creating new RTO fulfillment records
- We provide updated quotes including RTO charges when applicable
- We update order state to "Cancelled"
- We send this as asynchronous callback to the buyer's /on_cancel endpoint
- We must provide accurate cancellation details and any applicable charges

## 3. Endpoint Details
- HTTP Method: POST
- Endpoint path: /on_cancel (at Buyer NP's bap_uri)
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
    "action": "on_cancel",
    "core_version": "1.2.0",
    "bap_id": "logistics_buyer.com",
    "bap_uri": "https://logistics_buyer.com/ondc",
    "bpp_id": "lsp.com",
    "bpp_uri": "https://lsp.com/ondc",
    "transaction_id": "T1",
    "message_id": "M5",
    "timestamp": "2023-06-06T23:00:30.000Z"
  },
  "message": {
    "order": {
      "id": "O2",
      "state": "Cancelled",
      "cancellation": {
        "cancelled_by": "lsp.com",
        "reason": {
          "id": "013"
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
          "value": "82.60"
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
            "@ondc/org/item_id": "I2",
            "@ondc/org/title_type": "rto",
            "price": {
              "currency": "INR",
              "value": "20.0"
            }
          },
          {
            "@ondc/org/item_id": "I2",
            "@ondc/org/title_type": "tax",
            "price": {
              "currency": "INR",
              "value": "3.60"
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
              "code": "Pending"
            }
          },
          "@ondc/org/awb_no": "1227262193237777",
          "tracking": false,
          "start": {
            "person": {
              "name": "person_name"
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
              "long_desc": "QR code will be attached to package",
              "additional_desc": {
                "content_type": "text/html",
                "url": "https://reverse_qc_sop_form.htm"
              }
            },
            "time": {
              "duration": "PT15M",
              "range": {
                "start": "2023-06-06T22:30:00.000Z",
                "end": "2023-06-06T22:45:00.000Z"
              }
            }
          },
          "end": {
            "person": {
              "name": "person_name"
            },
            "location": {
              "gps": "12.4535445,77.9283792",
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
              "short_desc": "value of DCC"
            },
            "time": {
              "range": {
                "start": "2023-06-06T23:00:00.000Z",
                "end": "2023-06-06T23:15:00.000Z"
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
          "tags": [
            {
              "code": "rto_event",
              "list": [
                {
                  "code": "retry_count",
                  "value": "3"
                },
                {
                  "code": "rto_id",
                  "value": "F1-RTO"
                },
                {
                  "code": "cancellation_reason_id",
                  "value": "013"
                },
                {
                  "code": "sub_reason_id",
                  "value": "004"
                },
                {
                  "code": "cancelled_by",
                  "value": "lsp.com"
                }
              ]
            },
            {
              "code": "igm_request",
              "list": [
                {
                  "code": "id",
                  "value": "Issue1"
                }
              ]
            },
            {
              "code": "precancel_state",
              "list": [
                {
                  "code": "fulfillment_state",
                  "value": "Order-picked-up"
                },
                {
                  "code": "updated_at",
                  "value": "2023-06-06T23:15:00.000Z"
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
              "timestamp": "2023-06-07T14:00:00.000Z"
            }
          }
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
      "created_at": "2023-06-06T22:00:00.000Z",
      "updated_at": "2023-06-06T22:00:30.000Z"
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
| context.action | string | Y | Action name (on_cancel) | Seller NP |
| context.core_version | string | Y | ONDC core version | Seller NP |
| context.bap_id | string | Y | Buyer NP subscriber ID | Seller NP |
| context.bap_uri | string | Y | Buyer NP callback URI | Seller NP |
| context.bpp_id | string | Y | Seller NP subscriber ID | Seller NP |
| context.bpp_uri | string | Y | Seller NP callback URI | Seller NP |
| context.transaction_id | string | Y | Transaction ID from request | Seller NP |
| context.message_id | string | Y | Message ID from request | Seller NP |
| context.timestamp | string | Y | Response timestamp | Seller NP |
| message.order.id | string | Y | Order ID | Seller NP |
| message.order.state | string | Y | Order state (Cancelled) | Seller NP |
| message.order.cancellation.cancelled_by | string | Y | Who initiated cancellation | Seller NP |
| message.order.cancellation.reason.id | string | Y | Cancellation reason code | Seller NP |
| message.order.provider.id | string | Y | Provider ID | Seller NP |
| message.order.provider.locations | array | Y | Provider locations | Seller NP |
| message.order.items | array | Y | Order items (including RTO items if applicable) | Seller NP |
| message.order.quote | object | Y | Updated quote with RTO charges if applicable | Seller NP |
| message.order.fulfillments | array | Y | Fulfillments (including RTO fulfillment if applicable) | Seller NP |
| message.order.billing | object | Y | Billing information | Seller NP |
| message.order.payment | object | Y | Payment information | Seller NP |
| message.order.@ondc/org/linked_order | object | Y | Linked retail order details | Seller NP |
| message.order.created_at | string | Y | Order creation timestamp | Seller NP |
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
      "code": "62503",
      "message": "RTO quote not acceptable"
    }
  }
}
```

## 6. Asynchronous Callback (if applicable)
This API does not have an asynchronous callback - it is itself the callback response to /cancel.

## 7. State & Correlation
- transaction_id: Must match the transaction_id from the original /cancel request
- message_id: Must match the message_id from the original /cancel request
- order_id: Must match the order_id from the /cancel request
- Correlation: Buyer NP correlates this callback using transaction_id + message_id + order_id

## 8. Validation Rules
- Mandatory field checks: All fields marked as Required=Y in the schema must be present
- Cancellation validation: Cancellation details must be accurate and consistent
- RTO validation: If RTO is initiated, new fulfillment and items must be created correctly
- Quote validation: RTO charges must match catalog pricing
- Timestamp & TTL validation: context.timestamp must be current time and within TTL window
- Stale request handling: If timestamp is older than acceptable window, Buyer NP may NACK
- Signing verification: Buyer NP must verify Seller NP's signature using registry public key

## 9. Error Scenarios

| Scenario | Error Code | When it occurs | Seller NP Action |
|----------|------------|----------------|------------------|
| RTO quote not acceptable | 62503 | Buyer NP doesn't accept RTO pricing | Buyer NP sends NACK |
| Invalid RTO fulfillment | 40001 | RTO fulfillment structure invalid | Buyer NP sends NACK |
| Service temporarily unavailable | 50001 | Internal system issues | Buyer NP sends NACK |
| Stale callback | 65003 | Callback sent after TTL expired | Buyer NP sends NACK |

## 10. Important Notes (Seller NP)
- Seller NP MUST set order.state to "Cancelled" in callback response
- Seller NP MUST include cancellation.cancelled_by indicating who initiated cancellation
- Seller NP MUST create RTO fulfillment and items when RTO is triggered
- Seller NP MUST include rto_event tags with retry count and reason details
- Seller NP MUST include precancel_state in fulfillment tags
- Seller NP SHOULD include IGM request reference if applicable
- Seller NP MUST ensure RTO quote matches the catalog pricing
- Common mistake to avoid: Missing RTO fulfillment creation or incorrect cancellation attribution
