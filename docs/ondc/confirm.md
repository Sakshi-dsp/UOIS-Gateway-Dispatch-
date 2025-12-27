# confirm API (Seller NP)

## 1. Overview
- Purpose of this API: This API allows Buyer NP to confirm the logistics order with final details including order ID, payment, and fulfillment instructions
- Who calls it (Buyer NP / Seller NP): Called by Buyer NP (BAP)
- Sync vs Async behavior: Asynchronous - Buyer NP sends request, Seller NP responds with ACK/NACK synchronously, then sends callback via /on_confirm
- Callback expectations: Seller NP MUST send /on_confirm callback with order acceptance/rejection within TTL window
- TTL behavior (if applicable): TTL is specified in context.ttl (default PT30S)

## 2. Role Perspective
We are Seller NP (BPP / Logistics Service Provider). In this API:
- We receive the final order confirmation with order ID and payment details
- We perform final validation of all order details
- We decide whether to accept or reject the order
- We return ACK/NACK synchronously
- We send order acceptance/rejection asynchronously via /on_confirm callback
- We must validate that the order details match the init response

## 3. Endpoint Details
- HTTP Method: POST
- Endpoint path: /confirm
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
    "action": "confirm",
    "core_version": "1.2.0",
    "bap_id": "logistics_buyer.com",
    "bap_uri": "https://logistics_buyer.com/ondc",
    "bpp_id": "lsp.com",
    "bpp_uri": "https://lsp.com/ondc",
    "transaction_id": "T1",
    "message_id": "M3",
    "timestamp": "2023-06-06T22:00:00.000Z",
    "ttl": "PT30S"
  },
  "message": {
    "order": {
      "id": "O2",
      "state": "Created",
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
          "category_id": "Immediate Delivery",
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
          "@ondc/org/awb_no": "1227262193237777",
          "start": {
            "time": {
              "duration": "PT15M"
            },
            "person": {
              "name": "person_name"
            },
            "location": {
              "gps": "12.4535445,77.9283792",
              "address": {
                "name": "Store name",
                "building": "House or building name",
                "locality": "Locality",
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
              "additional_desc": {
                "content_type": "text/html",
                "url": "https://reverse_qc_sop_form.htm"
              }
            }
          },
          "end": {
            "person": {
              "name": "person_name"
            },
            "location": {
              "gps": "12.453544,77.928379",
              "address": {
                "name": "My house #",
                "building": "My house or building name",
                "locality": "locality",
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
              "long_desc": "additional instructions for delivery"
            }
          },
          "tags": [
            {
              "code": "state",
              "list": [
                {
                  "code": "ready_to_ship",
                  "value": "no"
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
        "name": "ONDC Logistics Buyer NP",
        "address": {
          "name": "name",
          "building": "My house or building name",
          "locality": "My street name",
          "city": "Bengaluru",
          "state": "Karnataka",
          "country": "India",
          "area_code": "560076"
        },
        "tax_number": "29AAACU1901H1ZK",
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
      "updated_at": "2023-06-06T22:00:00.000Z"
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
| context.action | string | Y | Action name (confirm) | Buyer NP |
| context.core_version | string | Y | ONDC core version (1.2.0) | Buyer NP |
| context.bap_id | string | Y | Buyer NP subscriber ID | Buyer NP |
| context.bap_uri | string | Y | Buyer NP callback URI | Buyer NP |
| context.bpp_id | string | Y | Seller NP subscriber ID | Buyer NP |
| context.bpp_uri | string | Y | Seller NP callback URI | Buyer NP |
| context.transaction_id | string | Y | Unique transaction identifier | Buyer NP |
| context.message_id | string | Y | Unique message identifier | Buyer NP |
| context.timestamp | string | Y | Request timestamp in RFC3339 format | Buyer NP |
| context.ttl | string | Y | Time to live for response (PT30S) | Buyer NP |
| message.order.id | string | Y | Unique order identifier (alphanumeric, up to 32 chars) | Buyer NP |
| message.order.state | string | Y | Order state (Created) | Buyer NP |
| message.order.provider.id | string | Y | Selected provider ID | Buyer NP |
| message.order.provider.locations[].id | string | Y | Selected location ID | Buyer NP |
| message.order.items[].id | string | Y | Selected item ID | Buyer NP |
| message.order.items[].fulfillment_id | string | Y | Associated fulfillment ID | Buyer NP |
| message.order.items[].category_id | string | Y | Item category | Buyer NP |
| message.order.items[].descriptor.code | string | Y | Item code (P2P) | Buyer NP |
| message.order.items[].time.label | string | Y | Time label (TAT) | Buyer NP |
| message.order.items[].time.duration | string | Y | TAT duration | Buyer NP |
| message.order.items[].time.timestamp | string | Y | TAT date | Buyer NP |
| message.order.quote.price.currency | string | Y | Currency code | Buyer NP |
| message.order.quote.price.value | string | Y | Total quote value | Buyer NP |
| message.order.quote.breakup[].@ondc/org/item_id | string | Y | Item ID for breakup | Buyer NP |
| message.order.quote.breakup[].@ondc/org/title_type | string | Y | Breakup type | Buyer NP |
| message.order.quote.breakup[].price.currency | string | Y | Currency for breakup | Buyer NP |
| message.order.quote.breakup[].price.value | string | Y | Value for breakup | Buyer NP |
| message.order.fulfillments[].id | string | Y | Fulfillment unique identifier | Buyer NP |
| message.order.fulfillments[].type | string | Y | Fulfillment type (Delivery) | Buyer NP |
| message.order.fulfillments[].@ondc/org/awb_no | string | N | AWB number (11-16 digits, optional for P2P) | Buyer NP |
| message.order.fulfillments[].start.time.duration | string | N | Pickup time duration | Buyer NP |
| message.order.fulfillments[].start.person.name | string | N | Pickup person name | Buyer NP |
| message.order.fulfillments[].start.location.gps | string | Y | Pickup GPS coordinates | Buyer NP |
| message.order.fulfillments[].start.location.address.name | string | Y | Pickup location name | Buyer NP |
| message.order.fulfillments[].start.location.address.building | string | Y | Pickup building | Buyer NP |
| message.order.fulfillments[].start.location.address.locality | string | Y | Pickup locality | Buyer NP |
| message.order.fulfillments[].start.location.address.city | string | Y | Pickup city | Buyer NP |
| message.order.fulfillments[].start.location.address.state | string | Y | Pickup state | Buyer NP |
| message.order.fulfillments[].start.location.address.country | string | Y | Pickup country | Buyer NP |
| message.order.fulfillments[].start.location.address.area_code | string | Y | Pickup pincode | Buyer NP |
| message.order.fulfillments[].start.contact.phone | string | Y | Pickup phone | Buyer NP |
| message.order.fulfillments[].start.contact.email | string | N | Pickup email | Buyer NP |
| message.order.fulfillments[].start.instructions.code | string | N | Pickup instruction code | Buyer NP |
| message.order.fulfillments[].start.instructions.short_desc | string | N | Pickup instruction description | Buyer NP |
| message.order.fulfillments[].start.instructions.long_desc | string | N | Detailed pickup instructions | Buyer NP |
| message.order.fulfillments[].start.instructions.additional_desc.content_type | string | N | Content type for additional desc | Buyer NP |
| message.order.fulfillments[].start.instructions.additional_desc.url | string | N | URL for reverse QC form | Buyer NP |
| message.order.fulfillments[].end.person.name | string | N | Delivery person name | Buyer NP |
| message.order.fulfillments[].end.location.gps | string | Y | Dropoff GPS coordinates | Buyer NP |
| message.order.fulfillments[].end.location.address.name | string | Y | Dropoff location name | Buyer NP |
| message.order.fulfillments[].end.location.address.building | string | Y | Dropoff building | Buyer NP |
| message.order.fulfillments[].end.location.address.locality | string | Y | Dropoff locality | Buyer NP |
| message.order.fulfillments[].end.location.address.city | string | Y | Dropoff city | Buyer NP |
| message.order.fulfillments[].end.location.address.state | string | Y | Dropoff state | Buyer NP |
| message.order.fulfillments[].end.location.address.country | string | Y | Dropoff country | Buyer NP |
| message.order.fulfillments[].end.location.address.area_code | string | Y | Dropoff pincode | Buyer NP |
| message.order.fulfillments[].end.contact.phone | string | Y | Dropoff phone | Buyer NP |
| message.order.fulfillments[].end.contact.email | string | N | Dropoff email | Buyer NP |
| message.order.fulfillments[].end.instructions.code | string | N | Delivery instruction code | Buyer NP |
| message.order.fulfillments[].end.instructions.short_desc | string | N | Delivery instruction description | Buyer NP |
| message.order.fulfillments[].end.instructions.long_desc | string | N | Detailed delivery instructions | Buyer NP |
| message.order.fulfillments[].tags | array | N | Fulfillment state and configuration tags | Buyer NP |
| message.order.billing.name | string | Y | Billing entity name | Buyer NP |
| message.order.billing.address.name | string | Y | Billing address name | Buyer NP |
| message.order.billing.address.building | string | Y | Billing building | Buyer NP |
| message.order.billing.address.locality | string | Y | Billing locality | Buyer NP |
| message.order.billing.address.city | string | Y | Billing city | Buyer NP |
| message.order.billing.address.state | string | Y | Billing state | Buyer NP |
| message.order.billing.address.country | string | Y | Billing country | Buyer NP |
| message.order.billing.address.area_code | string | Y | Billing pincode | Buyer NP |
| message.order.billing.tax_number | string | Y | GST number | Buyer NP |
| message.order.billing.phone | string | Y | Billing phone | Buyer NP |
| message.order.billing.email | string | Y | Billing email | Buyer NP |
| message.order.billing.created_at | string | Y | Billing record creation timestamp | Buyer NP |
| message.order.billing.updated_at | string | Y | Billing record update timestamp | Buyer NP |
| message.order.payment.@ondc/org/collection_amount | string | N | Collection amount for CoD | Buyer NP |
| message.order.payment.collected_by | string | Y | Payment collector (BPP/BAP) | Buyer NP |
| message.order.payment.type | string | Y | Payment type | Buyer NP |
| message.order.payment.@ondc/org/settlement_details | array | N | Settlement details | Buyer NP |
| message.order.@ondc/org/linked_order.items[].category_id | string | Y | Linked order item category | Buyer NP |
| message.order.@ondc/org/linked_order.items[].descriptor.name | string | Y | Linked order item name | Buyer NP |
| message.order.@ondc/org/linked_order.items[].quantity.count | number | Y | Item quantity count | Buyer NP |
| message.order.@ondc/org/linked_order.items[].quantity.measure.unit | string | Y | Quantity unit | Buyer NP |
| message.order.@ondc/org/linked_order.items[].quantity.measure.value | number | Y | Quantity value | Buyer NP |
| message.order.@ondc/org/linked_order.items[].price.currency | string | Y | Item currency | Buyer NP |
| message.order.@ondc/org/linked_order.items[].price.value | string | Y | Item price | Buyer NP |
| message.order.@ondc/org/linked_order.provider.descriptor.name | string | Y | Retail provider name | Buyer NP |
| message.order.@ondc/org/linked_order.provider.address.name | string | Y | Retail provider address | Buyer NP |
| message.order.@ondc/org/linked_order.provider.address.building | string | Y | Retail provider building | Buyer NP |
| message.order.@ondc/org/linked_order.provider.address.locality | string | Y | Retail provider locality | Buyer NP |
| message.order.@ondc/org/linked_order.provider.address.city | string | Y | Retail provider city | Buyer NP |
| message.order.@ondc/org/linked_order.provider.address.state | string | Y | Retail provider state | Buyer NP |
| message.order.@ondc/org/linked_order.provider.address.area_code | string | Y | Retail provider pincode | Buyer NP |
| message.order.@ondc/org/linked_order.order.id | string | Y | Linked retail order ID | Buyer NP |
| message.order.@ondc/org/linked_order.order.weight.unit | string | Y | Package weight unit | Buyer NP |
| message.order.@ondc/org/linked_order.order.weight.value | number | Y | Package weight value | Buyer NP |
| message.order.@ondc/org/linked_order.order.dimensions.length.unit | string | N | Package length unit | Buyer NP |
| message.order.@ondc/org/linked_order.order.dimensions.length.value | number | N | Package length | Buyer NP |
| message.order.@ondc/org/linked_order.order.dimensions.breadth.unit | string | N | Package breadth unit | Buyer NP |
| message.order.@ondc/org/linked_order.order.dimensions.breadth.value | number | N | Package breadth | Buyer NP |
| message.order.@ondc/org/linked_order.order.dimensions.height.unit | string | N | Package height unit | Buyer NP |
| message.order.@ondc/org/linked_order.order.dimensions.height.value | number | N | Package height | Buyer NP |
| message.order.tags | array | Y | Terms acceptance tags | Buyer NP |
| message.order.created_at | string | Y | Order creation timestamp | Buyer NP |
| message.order.updated_at | string | Y | Order update timestamp | Buyer NP |

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
    "action": "confirm",
    "core_version": "1.2.0",
    "bap_id": "logistics_buyer.com",
    "bap_uri": "https://logistics_buyer.com/ondc",
    "bpp_id": "lsp.com",
    "bpp_uri": "https://lsp.com/ondc",
    "transaction_id": "T1",
    "message_id": "M3",
    "timestamp": "2023-06-06T22:00:30.000Z"
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
      "code": "60008",
      "message": "TAT mismatch with quoted value"
    }
  },
  "context": {
    "domain": "nic2004:60232",
    "country": "IND",
    "city": "std:080",
    "action": "confirm",
    "core_version": "1.2.0",
    "bap_id": "logistics_buyer.com",
    "bap_uri": "https://logistics_buyer.com/ondc",
    "bpp_id": "lsp.com",
    "bpp_uri": "https://lsp.com/ondc",
    "transaction_id": "T1",
    "message_id": "M3",
    "timestamp": "2023-06-06T22:00:30.000Z"
  }
}
```

## 6. Asynchronous Callback (if applicable)

### 6.1 Callback Endpoint
- /on_confirm

### 6.2 Full Callback Payload Example
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

### 6.3 Callback Field Schema
| Field Path | Data Type | Required (Y/N) | Description | Source |
|------------|-----------|----------------|-------------|--------|
| context.domain | string | Y | Domain identifier for logistics | Seller NP |
| context.country | string | Y | Country code | Seller NP |
| context.city | string | Y | City code | Seller NP |
| context.action | string | Y | Action name (on_confirm) | Seller NP |
| context.core_version | string | Y | ONDC core version | Seller NP |
| context.bap_id | string | Y | Buyer NP subscriber ID | Seller NP |
| context.bap_uri | string | Y | Buyer NP callback URI | Seller NP |
| context.bpp_id | string | Y | Seller NP subscriber ID | Seller NP |
| context.bpp_uri | string | Y | Seller NP callback URI | Seller NP |
| context.transaction_id | string | Y | Transaction ID from request | Seller NP |
| context.message_id | string | Y | Message ID from request | Seller NP |
| context.timestamp | string | Y | Response timestamp | Seller NP |
| message.order.id | string | Y | Order ID from request | Seller NP |
| message.order.state | string | Y | Order state (Accepted/Created) | Seller NP |
| message.order.provider.id | string | Y | Provider ID | Seller NP |
| message.order.provider.locations[].id | string | Y | Location ID | Seller NP |
| message.order.items[].id | string | Y | Item ID | Seller NP |
| message.order.items[].fulfillment_id | string | Y | Fulfillment ID | Seller NP |
| message.order.items[].category_id | string | Y | Item category | Seller NP |
| message.order.items[].descriptor.code | string | Y | Item code | Seller NP |
| message.order.items[].time.label | string | Y | Time label | Seller NP |
| message.order.items[].time.duration | string | Y | TAT duration | Seller NP |
| message.order.items[].time.timestamp | string | Y | TAT date | Seller NP |
| message.order.quote.price.currency | string | Y | Currency code | Seller NP |
| message.order.quote.price.value | string | Y | Total quote value | Seller NP |
| message.order.quote.breakup | array | Y | Price breakup | Seller NP |
| message.order.fulfillments[].id | string | Y | Fulfillment ID | Seller NP |
| message.order.fulfillments[].type | string | Y | Fulfillment type | Seller NP |
| message.order.fulfillments[].state.descriptor.code | string | Y | Fulfillment state | Seller NP |
| message.order.fulfillments[].@ondc/org/awb_no | string | N | AWB number | Seller NP |
| message.order.fulfillments[].tracking | boolean | N | Tracking enabled flag | Seller NP |
| message.order.fulfillments[].start.person.name | string | N | Pickup person name | Seller NP |
| message.order.fulfillments[].start.location.gps | string | Y | Pickup GPS | Seller NP |
| message.order.fulfillments[].start.location.address | object | Y | Pickup address details | Seller NP |
| message.order.fulfillments[].start.contact.phone | string | Y | Pickup phone | Seller NP |
| message.order.fulfillments[].start.contact.email | string | N | Pickup email | Seller NP |
| message.order.fulfillments[].start.instructions | object | N | Pickup instructions | Seller NP |
| message.order.fulfillments[].start.time.duration | string | Y | Pickup time duration | Seller NP |
| message.order.fulfillments[].start.time.range.start | string | Y | Pickup time start | Seller NP |
| message.order.fulfillments[].start.time.range.end | string | Y | Pickup time end | Seller NP |
| message.order.fulfillments[].end.person.name | string | N | Delivery person name | Seller NP |
| message.order.fulfillments[].end.location.gps | string | Y | Dropoff GPS | Seller NP |
| message.order.fulfillments[].end.location.address | object | Y | Dropoff address details | Seller NP |
| message.order.fulfillments[].end.contact.phone | string | Y | Dropoff phone | Seller NP |
| message.order.fulfillments[].end.contact.email | string | N | Dropoff email | Seller NP |
| message.order.fulfillments[].end.instructions | object | N | Delivery instructions | Seller NP |
| message.order.fulfillments[].end.time.range.start | string | Y | Delivery time start | Seller NP |
| message.order.fulfillments[].end.time.range.end | string | Y | Delivery time end | Seller NP |
| message.order.fulfillments[].agent.name | string | N | Agent name | Seller NP |
| message.order.fulfillments[].agent.phone | string | N | Agent phone | Seller NP |
| message.order.fulfillments[].vehicle.registration | string | N | Vehicle registration | Seller NP |
| message.order.fulfillments[].tags | array | N | Fulfillment tags | Seller NP |
| message.order.billing | object | Y | Billing details | Seller NP |
| message.order.payment | object | Y | Payment details | Seller NP |
| message.order.@ondc/org/linked_order | object | Y | Linked retail order details | Seller NP |
| message.order.cancellation_terms | array | Y | Cancellation terms | Seller NP |
| message.order.tags | array | Y | Terms tags | Seller NP |
| message.order.created_at | string | Y | Order creation timestamp | Seller NP |
| message.order.updated_at | string | Y | Order update timestamp | Seller NP |

## 7. State & Correlation
- transaction_id: Unique identifier for the confirm transaction, set by Buyer NP. Seller NP must use the same transaction_id in ACK, NACK, and callback responses
- message_id: Unique identifier for this confirm request, set by Buyer NP. Seller NP must echo this in ACK/NACK responses
- order_id: New order identifier assigned by Buyer NP. Seller NP must use this order_id in all subsequent API calls for this order
- Correlation: Seller NP correlates the callback to the original request using transaction_id + message_id. The callback should be sent to bap_uri endpoint

## 8. Validation Rules
- Mandatory field checks: All fields marked as Required=Y in the schema must be present
- Enum validations: order.state must be Created, fulfillment.type must be Delivery, payment.type must be valid
- Timestamp & TTL validation: context.timestamp must be current time, response must be sent within TTL window (PT30S)
- Stale request handling: If timestamp is older than acceptable window, respond with NACK error code 65003
- Signing verification: Verify request signature using Buyer's public key from registry
- Order validation: Quote must match /on_init response, fulfillment details must be complete
- TAT validation: TAT must match what was quoted in catalog and init responses

## 9. Error Scenarios

| Scenario | Error Code | When it occurs | Seller NP Action |
|----------|------------|----------------|------------------|
| Invalid order details | 66002 | Order validation failure | Send NACK with LSP-ERROR |
| TAT mismatch | 60008 | TAT different from quoted | Send NACK with LSP-ERROR |
| Weight/dimensions changed | 60011 | Package details changed | Send NACK with LSP-ERROR |
| Service temporarily unavailable | 50001 | Internal LSP system issues | Send NACK with LSP-ERROR |
| Stale request | 65003 | Request timestamp too old | Send NACK with PROTOCOL-ERROR |
| Invalid signature | 401 | Signature verification fails | Return HTTP 401 |
| Terms not accepted | 65002 | BPP terms not accepted | Send NACK with LSP-ERROR |

## 10. Important Notes (Seller NP)
- Seller NP MAY internally create shipment records upon successful /confirm processing (manifest is internal, not mandated by ONDC for P2P)
- Seller NP MUST validate that all order details match the /init and /on_init responses
- Seller NP MUST ensure TAT values are consistent with quoted values
- Seller NP MAY assign AWB number if internally tracking shipments (optional for P2P, not mandatory)
- Seller NP MUST set appropriate pickup and delivery time slots
- Seller NP SHOULD assign agent and vehicle details if available at confirmation time (rider assignment typically happens post-confirm for P2P)
- Seller NP MAY include weather conditions and other fulfillment tags as applicable
- Common mistake to avoid: Accepting orders without final serviceability validation or sending incorrect time slots
