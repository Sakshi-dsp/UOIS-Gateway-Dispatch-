# on_confirm API (Seller NP)

## 1. Overview
- Purpose of this API: This API allows Seller NP to communicate order acceptance/rejection and provide final order details including fulfillment schedules and tracking information
- Who calls it (Buyer NP / Seller NP): Called by Seller NP (BPP) as callback response to /confirm
- Sync vs Async behavior: Asynchronous - This is the callback response sent after ACK to /confirm request
- Callback expectations: Buyer NP acknowledges receipt with ACK/NACK
- TTL behavior (if applicable): Response must be sent within TTL specified in original /confirm request

## 2. Role Perspective
We are Seller NP (BPP / Logistics Service Provider). In this API:
- We communicate final order acceptance or rejection
- We provide detailed fulfillment schedules with time slots
- We assign agents, vehicles, and tracking information
- We confirm all order terms and conditions
- We send this as asynchronous callback to the buyer's /on_confirm endpoint
- We must ensure order is ready for execution or provide clear rejection reasons

## 3. Endpoint Details
- HTTP Method: POST
- Endpoint path: /on_confirm (at Buyer NP's bap_uri)
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
    "action": "on_confirm",
    "core_version": "1.2.0",
    "bap_id": "logistics_buyer.com",
    "bap_uri": "https://logistics_buyer.com/ondc",
    "bpp_id": "lsp.com",
    "bpp_uri": "https://lsp.com/ondc",
    "transaction_id": "T1",
    "message_id": "M3",
    "timestamp": "2023-06-06T22:00:30.000Z"
  },
  "message": {
    "order": {
      "id": "O2",
      "state": "Accepted",
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
          "value": "59.00"
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
                "name": "Store name",
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
              "images": [
                "link to downloadable shipping label (optional for P2P)"
              ],
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
              "code": "weather_check",
              "list": [
                {
                  "code": "raining",
                  "value": "yes"
                }
              ]
            },
            {
              "code": "state",
              "list": [
                {
                  "code": "ready_to_ship",
                  "value": "yes"
                }
              ]
            },
            {
              "code": "rto_action",
              "list": [
                {
                  "code": "return_to_origin",
                  "value": "no"
                }
              ]
            },
            {
              "code": "reverseqc_input",
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
                  "value": ""
                }
              ]
            }
          ]
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
      "cancellation_terms": [
        {
          "fulfillment_state": {
            "descriptor": {
              "code": "Pending",
              "short_desc": "008"
            }
          },
          "cancellation_fee": {
            "percentage": "0.00",
            "amount": {
              "currency": "INR",
              "value": "0.00"
            }
          }
        },
        {
          "fulfillment_state": {
            "descriptor": {
              "code": "Agent-assigned",
              "short_desc": "001,003"
            }
          },
          "cancellation_fee": {
            "percentage": "100.00",
            "amount": {
              "currency": "INR",
              "value": "50.00"
            }
          }
        },
        {
          "fulfillment_state": {
            "descriptor": {
              "code": "Order-picked-up",
              "short_desc": "001,003"
            }
          },
          "cancellation_fee": {
            "percentage": "100.00",
            "amount": {
              "currency": "INR",
              "value": "50.00"
            }
          }
        },
        {
          "fulfillment_state": {
            "descriptor": {
              "code": "Out-for-delivery",
              "short_desc": "011,012,013,014,015"
            }
          },
          "cancellation_fee": {
            "percentage": "100.00",
            "amount": {
              "currency": "INR",
              "value": "50.00"
            }
          }
        }
      ],
      "tags": [
        {
          "code": "bpp_terms",
          "list": [
            {
              "code": "max_liability",
              "value": "2"
            },
            {
              "code": "max_liability_cap",
              "value": "10000"
            },
            {
              "code": "mandatory_arbitration",
              "value": "false"
            },
            {
              "code": "court_jurisdiction",
              "value": "Bengaluru"
            },
            {
              "code": "delay_interest",
              "value": "1000"
            },
            {
              "code": "static_terms",
              "value": "https://github.com/ONDC-Official/protocol-network-extension/discussions/79"
            }
          ]
        },
        {
          "code": "bap_terms",
          "list": [
            {
              "code": "accept_bpp_terms",
              "value": "Y"
            }
          ]
        }
      ],
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
| context.action | string | Y | Action name (on_confirm) | Seller NP |
| context.core_version | string | Y | ONDC core version | Seller NP |
| context.bap_id | string | Y | Buyer NP subscriber ID | Seller NP |
| context.bap_uri | string | Y | Buyer NP callback URI | Seller NP |
| context.bpp_id | string | Y | Seller NP subscriber ID | Seller NP |
| context.bpp_uri | string | Y | Seller NP callback URI | Seller NP |
| context.transaction_id | string | Y | Transaction ID from original confirm request | Seller NP |
| context.message_id | string | Y | Message ID from original confirm request | Seller NP |
| context.timestamp | string | Y | Response timestamp in RFC3339 format | Seller NP |
| message.order.id | string | Y | Order ID from confirm request | Seller NP |
| message.order.state | string | Y | Order state (Accepted) | Seller NP |
| message.order.provider.id | string | Y | Provider ID | Seller NP |
| message.order.provider.locations[].id | string | Y | Location ID | Seller NP |
| message.order.items[].id | string | Y | Item ID | Seller NP |
| message.order.items[].fulfillment_id | string | Y | Fulfillment ID | Seller NP |
| message.order.items[].category_id | string | Y | Item category | Seller NP |
| message.order.items[].descriptor.code | string | Y | Item code (P2P) | Seller NP |
| message.order.items[].time.label | string | Y | Time label (TAT) | Seller NP |
| message.order.items[].time.duration | string | Y | TAT duration | Seller NP |
| message.order.items[].time.timestamp | string | Y | TAT date | Seller NP |
| message.order.quote.price.currency | string | Y | Currency code | Seller NP |
| message.order.quote.price.value | string | Y | Total quote value | Seller NP |
| message.order.quote.breakup[].@ondc/org/item_id | string | Y | Item ID for breakup | Seller NP |
| message.order.quote.breakup[].@ondc/org/title_type | string | Y | Breakup type | Seller NP |
| message.order.quote.breakup[].price.currency | string | Y | Currency for breakup | Seller NP |
| message.order.quote.breakup[].price.value | string | Y | Value for breakup | Seller NP |
| message.order.fulfillments[].id | string | Y | Fulfillment ID | Seller NP |
| message.order.fulfillments[].type | string | Y | Fulfillment type (Delivery) | Seller NP |
| message.order.fulfillments[].state.descriptor.code | string | Y | Fulfillment state (Pending) | Seller NP |
| message.order.fulfillments[].@ondc/org/awb_no | string | N | AWB number (optional for P2P) | Seller NP |
| message.order.fulfillments[].tracking | boolean | N | Tracking enabled flag | Seller NP |
| message.order.fulfillments[].start.person.name | string | N | Pickup person name | Seller NP |
| message.order.fulfillments[].start.location.gps | string | Y | Pickup GPS coordinates | Seller NP |
| message.order.fulfillments[].start.location.address.name | string | Y | Pickup location name | Seller NP |
| message.order.fulfillments[].start.location.address.building | string | Y | Pickup building | Seller NP |
| message.order.fulfillments[].start.location.address.locality | string | Y | Pickup locality | Seller NP |
| message.order.fulfillments[].start.location.address.city | string | Y | Pickup city | Seller NP |
| message.order.fulfillments[].start.location.address.state | string | Y | Pickup state | Seller NP |
| message.order.fulfillments[].start.location.address.country | string | Y | Pickup country | Seller NP |
| message.order.fulfillments[].start.location.address.area_code | string | Y | Pickup pincode | Seller NP |
| message.order.fulfillments[].start.contact.phone | string | Y | Pickup phone | Seller NP |
| message.order.fulfillments[].start.contact.email | string | N | Pickup email | Seller NP |
| message.order.fulfillments[].start.instructions.code | string | N | Pickup instruction code | Seller NP |
| message.order.fulfillments[].start.instructions.short_desc | string | N | Pickup instruction description | Seller NP |
| message.order.fulfillments[].start.instructions.long_desc | string | N | Detailed pickup instructions | Seller NP |
| message.order.fulfillments[].start.instructions.images | array | N | Pickup instruction images | Seller NP |
| message.order.fulfillments[].start.instructions.additional_desc.content_type | string | N | Content type for additional desc | Seller NP |
| message.order.fulfillments[].start.instructions.additional_desc.url | string | N | URL for reverse QC form | Seller NP |
| message.order.fulfillments[].start.time.duration | string | Y | Pickup time duration | Seller NP |
| message.order.fulfillments[].start.time.range.start | string | Y | Pickup time start | Seller NP |
| message.order.fulfillments[].start.time.range.end | string | Y | Pickup time end | Seller NP |
| message.order.fulfillments[].end.person.name | string | N | Delivery person name | Seller NP |
| message.order.fulfillments[].end.location.gps | string | Y | Dropoff GPS coordinates | Seller NP |
| message.order.fulfillments[].end.location.address.name | string | Y | Dropoff location name | Seller NP |
| message.order.fulfillments[].end.location.address.building | string | Y | Dropoff building | Seller NP |
| message.order.fulfillments[].end.location.address.locality | string | Y | Dropoff locality | Seller NP |
| message.order.fulfillments[].end.location.address.city | string | Y | Dropoff city | Seller NP |
| message.order.fulfillments[].end.location.address.state | string | Y | Dropoff state | Seller NP |
| message.order.fulfillments[].end.location.address.country | string | Y | Dropoff country | Seller NP |
| message.order.fulfillments[].end.location.address.area_code | string | Y | Dropoff pincode | Seller NP |
| message.order.fulfillments[].end.contact.phone | string | Y | Dropoff phone | Seller NP |
| message.order.fulfillments[].end.contact.email | string | N | Dropoff email | Seller NP |
| message.order.fulfillments[].end.instructions.code | string | N | Delivery instruction code | Seller NP |
| message.order.fulfillments[].end.instructions.short_desc | string | N | Delivery instruction description | Seller NP |
| message.order.fulfillments[].end.time.range.start | string | Y | Delivery time start | Seller NP |
| message.order.fulfillments[].end.time.range.end | string | Y | Delivery time end | Seller NP |
| message.order.fulfillments[].agent.name | string | N | Agent name | Seller NP |
| message.order.fulfillments[].agent.phone | string | N | Agent phone | Seller NP |
| message.order.fulfillments[].vehicle.registration | string | N | Vehicle registration | Seller NP |
| message.order.fulfillments[].tags | array | N | Fulfillment tags | Seller NP |
| message.order.billing.name | string | Y | Billing entity name | Seller NP |
| message.order.billing.address.name | string | Y | Billing address name | Seller NP |
| message.order.billing.address.building | string | Y | Billing building | Seller NP |
| message.order.billing.address.locality | string | Y | Billing locality | Seller NP |
| message.order.billing.address.city | string | Y | Billing city | Seller NP |
| message.order.billing.address.state | string | Y | Billing state | Seller NP |
| message.order.billing.address.country | string | Y | Billing country | Seller NP |
| message.order.billing.address.area_code | string | Y | Billing pincode | Seller NP |
| message.order.billing.tax_number | string | Y | GST number | Seller NP |
| message.order.billing.phone | string | Y | Billing phone | Seller NP |
| message.order.billing.email | string | Y | Billing email | Seller NP |
| message.order.billing.created_at | string | Y | Billing record creation timestamp | Seller NP |
| message.order.billing.updated_at | string | Y | Billing record update timestamp | Seller NP |
| message.order.payment.@ondc/org/collection_amount | string | N | Collection amount for CoD | Seller NP |
| message.order.payment.collected_by | string | Y | Payment collector | Seller NP |
| message.order.payment.type | string | Y | Payment type | Seller NP |
| message.order.payment.@ondc/org/settlement_details | array | N | Settlement details | Seller NP |
| message.order.@ondc/org/linked_order.items[].category_id | string | Y | Linked order item category | Seller NP |
| message.order.@ondc/org/linked_order.items[].descriptor.name | string | Y | Linked order item name | Seller NP |
| message.order.@ondc/org/linked_order.items[].quantity.count | number | Y | Item quantity count | Seller NP |
| message.order.@ondc/org/linked_order.items[].quantity.measure.unit | string | Y | Quantity unit | Seller NP |
| message.order.@ondc/org/linked_order.items[].quantity.measure.value | number | Y | Quantity value | Seller NP |
| message.order.@ondc/org/linked_order.items[].price.currency | string | Y | Item currency | Seller NP |
| message.order.@ondc/org/linked_order.items[].price.value | string | Y | Item price | Seller NP |
| message.order.@ondc/org/linked_order.provider.descriptor.name | string | Y | Retail provider name | Seller NP |
| message.order.@ondc/org/linked_order.provider.address.name | string | Y | Retail provider address | Seller NP |
| message.order.@ondc/org/linked_order.provider.address.building | string | Y | Retail provider building | Seller NP |
| message.order.@ondc/org/linked_order.provider.address.locality | string | Y | Retail provider locality | Seller NP |
| message.order.@ondc/org/linked_order.provider.address.city | string | Y | Retail provider city | Seller NP |
| message.order.@ondc/org/linked_order.provider.address.state | string | Y | Retail provider state | Seller NP |
| message.order.@ondc/org/linked_order.provider.address.area_code | string | Y | Retail provider pincode | Seller NP |
| message.order.@ondc/org/linked_order.order.id | string | Y | Linked retail order ID | Seller NP |
| message.order.@ondc/org/linked_order.order.weight.unit | string | Y | Package weight unit | Seller NP |
| message.order.@ondc/org/linked_order.order.weight.value | number | Y | Package weight value | Seller NP |
| message.order.@ondc/org/linked_order.order.dimensions.length.unit | string | N | Package length unit | Seller NP |
| message.order.@ondc/org/linked_order.order.dimensions.length.value | number | N | Package length | Seller NP |
| message.order.@ondc/org/linked_order.order.dimensions.breadth.unit | string | N | Package breadth unit | Seller NP |
| message.order.@ondc/org/linked_order.order.dimensions.breadth.value | number | N | Package breadth | Seller NP |
| message.order.@ondc/org/linked_order.order.dimensions.height.unit | string | N | Package height unit | Seller NP |
| message.order.@ondc/org/linked_order.order.dimensions.height.value | number | N | Package height | Seller NP |
| message.order.cancellation_terms[].fulfillment_state.descriptor.code | string | Y | Fulfillment state for cancellation | Seller NP |
| message.order.cancellation_terms[].fulfillment_state.descriptor.short_desc | string | Y | Reason codes | Seller NP |
| message.order.cancellation_terms[].cancellation_fee.percentage | string | Y | Cancellation fee percentage | Seller NP |
| message.order.cancellation_terms[].cancellation_fee.amount.currency | string | Y | Fee currency | Seller NP |
| message.order.cancellation_terms[].cancellation_fee.amount.value | string | Y | Fee amount | Seller NP |
| message.order.tags | array | Y | Terms acceptance tags | Seller NP |
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
      "code": "63002",
      "message": "Order validation failed"
    }
  }
}
```

## 6. Asynchronous Callback (if applicable)
This API does not have an asynchronous callback - it is itself the callback response to /confirm.

## 7. State & Correlation
- transaction_id: Must match the transaction_id from the original /confirm request
- message_id: Must match the message_id from the original /confirm request
- order_id: Must match the order_id from the /confirm request
- Correlation: Buyer NP correlates this callback to the original confirm using transaction_id + message_id + order_id

## 8. Validation Rules
- Mandatory field checks: All fields marked as Required=Y in the schema must be present
- Enum validations: order.state must be Accepted, fulfillment.state.descriptor.code must be valid
- Timestamp & TTL validation: context.timestamp must be current time and within TTL window from original request
- Stale request handling: If timestamp is older than acceptable window, Buyer NP may NACK with error code 65003
- Signing verification: Buyer NP must verify Seller NP's signature using public key from registry
- Order validation: All order details must match the /confirm request, fulfillment schedules must be reasonable

## 9. Error Scenarios

| Scenario | Error Code | When it occurs | Seller NP Action |
|----------|------------|----------------|------------------|
| Order validation failed | 63002 | Order details don't match confirm | Buyer NP sends NACK |
| Invalid time slots | 63003 | Pickup/delivery times unreasonable | Buyer NP sends NACK |
| Service temporarily unavailable | 50001 | Internal LSP system issues | Buyer NP sends NACK |
| Invalid fulfillment state | 40001 | Fulfillment state invalid | Buyer NP sends NACK |
| Stale callback | 65003 | Callback sent after TTL expired | Buyer NP sends NACK |

## 10. Important Notes (Seller NP)
- Seller NP MUST set order.state to "Accepted" when accepting the order
- Seller NP MUST provide realistic pickup and delivery time slots
- Seller NP SHOULD assign agent and vehicle details if available at confirmation time (for P2P, rider assignment typically happens post-confirm)
- Seller NP MAY include weather conditions and fulfillment tags as applicable
- Seller NP MUST ensure fulfillment.state is set to "Pending" for accepted orders
- Seller NP MAY assign AWB number if internally tracking shipments (optional for P2P, not mandatory)
- Seller NP MAY include shipping labels in instructions.images if applicable (optional for P2P)
- Common mistake to avoid: Sending incorrect time ranges or missing agent assignment when available
