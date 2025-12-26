# on_status API (Seller NP)

## 1. Overview
- Purpose of this API: This API allows Seller NP to provide comprehensive order status including fulfillment updates, payment status, and proof of delivery
- Who calls it (Buyer NP / Seller NP): Called by Seller NP (BPP) as callback response to /status
- Sync vs Async behavior: Asynchronous - This is the callback response sent after ACK to /status request
- Callback expectations: Buyer NP acknowledges receipt with ACK/NACK
- TTL behavior (if applicable): Response must be sent within TTL specified in original /status request

## 2. Role Perspective
We are Seller NP (BPP / Logistics Service Provider). In this API:
- We provide complete order status information
- We include current fulfillment states and timestamps
- We provide proof of pickup/delivery when available
- We update payment status and settlement information
- We send this as asynchronous callback to the buyer's /on_status endpoint
- We must ensure all status information is current and accurate

## 3. Endpoint Details
- HTTP Method: POST
- Endpoint path: /on_status (at Buyer NP's bap_uri)
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

### 4.2 Field-Level Schema (Table)

| Field Path | Data Type | Required (Y/N) | Description | Source of Truth |
|------------|-----------|----------------|-------------|-----------------|
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
| message.order.cancellation | object | N | Cancellation details if applicable | Seller NP |
| message.order.provider.id | string | Y | Provider ID | Seller NP |
| message.order.provider.locations | array | Y | Provider locations | Seller NP |
| message.order.items | array | Y | Order items with current status | Seller NP |
| message.order.quote | object | Y | Current quote (may include differential charges) | Seller NP |
| message.order.fulfillments | array | Y | Fulfillment details with current state | Seller NP |
| message.order.payment | object | Y | Payment status and settlement details | Seller NP |
| message.order.billing | object | Y | Billing information | Seller NP |
| message.order.@ondc/org/linked_order | object | Y | Linked retail order details | Seller NP |
| message.order.tags | array | N | Order-level tags (differential charges, etc.) | Seller NP |

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
      "code": "63002",
      "message": "Status update validation failed"
    }
  }
}
```

## 6. Asynchronous Callback (if applicable)
This API does not have an asynchronous callback - it is itself the callback response to /status.

## 7. State & Correlation
- transaction_id: Must match the transaction_id from the original /status request
- message_id: Must match the message_id from the original /status request
- order_id: Must match the order_id from the /status request
- Correlation: Buyer NP correlates this callback using transaction_id + message_id + order_id

## 8. Validation Rules
- Mandatory field checks: All fields marked as Required=Y in the schema must be present
- Order validation: Order details must match the confirmed order
- Timestamp & TTL validation: context.timestamp must be current time and within TTL window
- Stale request handling: If timestamp is older than acceptable window, Buyer NP may NACK
- Signing verification: Buyer NP must verify Seller NP's signature using registry public key

## 9. Error Scenarios

| Scenario | Error Code | When it occurs | Seller NP Action |
|----------|------------|----------------|------------------|
| Status validation failed | 63002 | Order details don't match records | Buyer NP sends NACK |
| Invalid fulfillment state | 40001 | Fulfillment state invalid | Buyer NP sends NACK |
| Service temporarily unavailable | 50001 | Internal system issues | Buyer NP sends NACK |
| Stale callback | 65003 | Callback sent after TTL expired | Buyer NP sends NACK |

## 10. Important Notes (Seller NP)
- Seller NP MUST provide current, accurate fulfillment states and timestamps
- Seller NP MUST include proof of pickup/delivery images when order is completed
- Seller NP MUST update payment status and settlement details when payment is processed
- Seller NP MUST include authorization details if pickup/delivery authorization was used
- Seller NP SHOULD include fulfillment delay information with reason codes
- Seller NP MUST include differential charges in quote breakup when applicable
- Common mistake to avoid: Sending incomplete fulfillment information or outdated timestamps
