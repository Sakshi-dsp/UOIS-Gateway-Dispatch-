# init API (Seller NP)

## 1. Overview
- Purpose of this API: This API allows Buyer NP to specify selected logistics services and confirm terms before order placement
- Who calls it (Buyer NP / Seller NP): Called by Buyer NP (BAP)
- Sync vs Async behavior: Asynchronous - Buyer NP sends request, Seller NP responds with ACK/NACK synchronously, then sends callback via /on_init
- Callback expectations: Seller NP MUST send /on_init callback with quote and terms within TTL window
- TTL behavior (if applicable): TTL is specified in context.ttl (default PT30S)

## 2. Role Perspective
We are Seller NP (BPP / Logistics Service Provider). In this API:
- We receive the buyer's selected provider, items, and fulfillment details
- We perform final serviceability validation for the selected options
- We return ACK/NACK synchronously
- We send detailed quote, terms, and fulfillment details asynchronously via /on_init callback
- We must validate that the selected items match our catalog offerings

## 3. Endpoint Details
- HTTP Method: POST
- Endpoint path: /init
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
    "action": "init",
    "core_version": "1.2.0",
    "bap_id": "logistics_buyer.com",
    "bap_uri": "https://logistics_buyer.com/ondc",
    "bpp_id": "lsp.com",
    "bpp_uri": "https://lsp.com/ondc",
    "transaction_id": "T1",
    "message_id": "M2",
    "timestamp": "2023-06-06T21:30:00.000Z",
    "ttl": "PT30S"
  },
  "message": {
    "order": {
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
          }
        }
      ],
      "fulfillments": [
        {
          "id": "1",
          "type": "Delivery",
          "start": {
            "location": {
              "gps": "12.453544,77.928379",
              "address": {
                "name": "My store name",
                "building": "My building name",
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
            }
          },
          "end": {
            "location": {
              "gps": "12.453544,77.928379",
              "address": {
                "name": "My house #",
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
            }
          }
        }
      ],
      "billing": {
        "name": "ONDC Logistics Buyer NP",
        "address": {
          "name": "My house or building no",
          "building": "My house or building name",
          "locality": "Jayanagar",
          "city": "Bengaluru",
          "state": "Karnataka",
          "country": "India",
          "area_code": "560076"
        },
        "tax_number": "XXXXXXXXXXXXXXX",
        "phone": "9886098860",
        "email": "abcd.efgh@gmail.com",
        "created_at": "2023-02-06T21:30:00.000Z",
        "updated_at": "2023-02-06T21:30:00.000Z"
      },
      "payment": {
        "@ondc/org/collection_amount": "300.00",
        "collected_by": "BPP",
        "type": "ON-FULFILLMENT",
        "@ondc/org/settlement_details": [
          {
            "settlement_counterparty": "buyer-app",
            "settlement_type": "upi",
            "beneficiary_name": "xxxxx",
            "upi_address": "gft@oksbi",
            "settlement_bank_account_no": "XXXXXXXXXX",
            "settlement_ifsc_code": "XXXXXXXXX"
          }
        ]
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
| context.action | string | Y | Action name (init) | Buyer NP |
| context.core_version | string | Y | ONDC core version (1.2.0) | Buyer NP |
| context.bap_id | string | Y | Buyer NP subscriber ID | Buyer NP |
| context.bap_uri | string | Y | Buyer NP callback URI | Buyer NP |
| context.bpp_id | string | Y | Seller NP subscriber ID | Buyer NP |
| context.bpp_uri | string | Y | Seller NP callback URI | Buyer NP |
| context.transaction_id | string | Y | Unique transaction identifier | Buyer NP |
| context.message_id | string | Y | Unique message identifier | Buyer NP |
| context.timestamp | string | Y | Request timestamp in RFC3339 format | Buyer NP |
| context.ttl | string | Y | Time to live for response (PT30S) | Buyer NP |
| message.order.provider.id | string | Y | Selected provider ID from catalog | Buyer NP |
| message.order.provider.locations[].id | string | N | Selected location ID (mandatory only if provider.locations was returned in /on_search, not required for P2P) | Buyer NP |
| message.order.items[].id | string | Y | Selected item ID from catalog | Buyer NP |
| message.order.items[].fulfillment_id | string | Y | Associated fulfillment ID | Buyer NP |
| message.order.items[].category_id | string | Y | Item category | Buyer NP |
| message.order.items[].descriptor.code | string | Y | Item code (P2P) | Buyer NP |
| message.order.fulfillments[].id | string | Y | Fulfillment unique identifier | Buyer NP |
| message.order.fulfillments[].type | string | Y | Fulfillment type (Delivery) | Buyer NP |
| message.order.fulfillments[].start.location.gps | string | Y | Pickup GPS coordinates | Buyer NP |
| message.order.fulfillments[].start.location.address.name | string | Y | Pickup location name | Buyer NP |
| message.order.fulfillments[].start.location.address.building | string | Y | Pickup building name | Buyer NP |
| message.order.fulfillments[].start.location.address.locality | string | Y | Pickup locality | Buyer NP |
| message.order.fulfillments[].start.location.address.city | string | Y | Pickup city | Buyer NP |
| message.order.fulfillments[].start.location.address.state | string | Y | Pickup state | Buyer NP |
| message.order.fulfillments[].start.location.address.country | string | Y | Pickup country | Buyer NP |
| message.order.fulfillments[].start.location.address.area_code | string | Y | Pickup pincode | Buyer NP |
| message.order.fulfillments[].start.contact.phone | string | Y | Pickup contact phone | Buyer NP |
| message.order.fulfillments[].start.contact.email | string | N | Pickup contact email | Buyer NP |
| message.order.fulfillments[].end.location.gps | string | Y | Dropoff GPS coordinates | Buyer NP |
| message.order.fulfillments[].end.location.address.name | string | Y | Dropoff location name | Buyer NP |
| message.order.fulfillments[].end.location.address.building | string | Y | Dropoff building name | Buyer NP |
| message.order.fulfillments[].end.location.address.locality | string | Y | Dropoff locality | Buyer NP |
| message.order.fulfillments[].end.location.address.city | string | Y | Dropoff city | Buyer NP |
| message.order.fulfillments[].end.location.address.state | string | Y | Dropoff state | Buyer NP |
| message.order.fulfillments[].end.location.address.country | string | Y | Dropoff country | Buyer NP |
| message.order.fulfillments[].end.location.address.area_code | string | Y | Dropoff pincode | Buyer NP |
| message.order.fulfillments[].end.contact.phone | string | Y | Dropoff contact phone | Buyer NP |
| message.order.fulfillments[].end.contact.email | string | N | Dropoff contact email | Buyer NP |
| message.order.billing.name | string | Y | Billing entity name | Buyer NP |
| message.order.billing.address.name | string | Y | Billing address name | Buyer NP |
| message.order.billing.address.building | string | Y | Billing building | Buyer NP |
| message.order.billing.address.locality | string | Y | Billing locality | Buyer NP |
| message.order.billing.address.city | string | Y | Billing city | Buyer NP |
| message.order.billing.address.state | string | Y | Billing state | Buyer NP |
| message.order.billing.address.country | string | Y | Billing country | Buyer NP |
| message.order.billing.address.area_code | string | Y | Billing pincode | Buyer NP |
| message.order.billing.tax_number | string | Y | GST number of buyer NP | Buyer NP |
| message.order.billing.phone | string | Y | Billing phone | Buyer NP |
| message.order.billing.email | string | Y | Billing email | Buyer NP |
| message.order.billing.created_at | string | Y | Billing record creation timestamp | Buyer NP |
| message.order.billing.updated_at | string | Y | Billing record update timestamp | Buyer NP |
| message.order.payment.@ondc/org/collection_amount | string | N | Collection amount for CoD | Buyer NP |
| message.order.payment.collected_by | string | Y | Who collects payment (BPP/BAP) | Buyer NP |
| message.order.payment.type | string | Y | Payment type (ON-ORDER/ON-FULFILLMENT/POST-FULFILLMENT) | Buyer NP |
| message.order.payment.@ondc/org/settlement_details[].settlement_counterparty | string | N | Settlement counterparty | Buyer NP |
| message.order.payment.@ondc/org/settlement_details[].settlement_type | string | N | Settlement type (upi/neft/rtgs) | Buyer NP |
| message.order.payment.@ondc/org/settlement_details[].beneficiary_name | string | N | Beneficiary name | Buyer NP |
| message.order.payment.@ondc/org/settlement_details[].upi_address | string | N | UPI address if applicable | Buyer NP |
| message.order.payment.@ondc/org/settlement_details[].settlement_bank_account_no | string | N | Bank account number | Buyer NP |
| message.order.payment.@ondc/org/settlement_details[].settlement_ifsc_code | string | N | IFSC code | Buyer NP |

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
    "action": "init",
    "core_version": "1.2.0",
    "bap_id": "logistics_buyer.com",
    "bap_uri": "https://logistics_buyer.com/ondc",
    "bpp_id": "lsp.com",
    "bpp_uri": "https://lsp.com/ondc",
    "transaction_id": "T1",
    "message_id": "M2",
    "timestamp": "2023-06-06T21:30:30.000Z"
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
      "code": "60001",
      "message": "Service not available for selected locations"
    }
  },
  "context": {
    "domain": "nic2004:60232",
    "country": "IND",
    "city": "std:080",
    "action": "init",
    "core_version": "1.2.0",
    "bap_id": "logistics_buyer.com",
    "bap_uri": "https://logistics_buyer.com/ondc",
    "bpp_id": "lsp.com",
    "bpp_uri": "https://lsp.com/ondc",
    "transaction_id": "T1",
    "message_id": "M2",
    "timestamp": "2023-06-06T21:30:30.000Z"
  }
}
```

## 6. Asynchronous Callback (if applicable)

### 6.1 Callback Endpoint
- /on_init

### 6.2 Full Callback Payload Example
```json
{
  "context": {
    "domain": "nic2004:60232",
    "country": "IND",
    "city": "std:080",
    "action": "on_init",
    "core_version": "1.2.0",
    "bap_id": "logistics_buyer.com",
    "bap_uri": "https://logistics_buyer.com/ondc",
    "bpp_id": "lsp.com",
    "bpp_uri": "https://lsp.com/ondc",
    "transaction_id": "T1",
    "message_id": "M2",
    "timestamp": "2023-02-06T21:30:30.000Z"
  },
  "message": {
    "order": {
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
          "fulfillment_id": "1"
        }
      ],
      "fulfillments": [
        {
          "id": "1",
          "type": "Delivery",
          "start": {
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
            }
          },
          "end": {
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
            }
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
        ],
        "ttl": "PT15M"
      },
      "payment": {
        "@ondc/org/collection_amount": "300.00",
        "type": "ON-FULFILLMENT",
        "collected_by": "BPP",
        "@ondc/org/settlement_details": [
          {
            "settlement_counterparty": "buyer-app",
            "settlement_type": "upi",
            "beneficiary_name": "xxxxx",
            "upi_address": "gft@oksbi",
            "settlement_bank_account_no": "XXXXXXXXXX",
            "settlement_ifsc_code": "XXXXXXXXX"
          }
        ]
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
| context.action | string | Y | Action name (on_init) | Seller NP |
| context.core_version | string | Y | ONDC core version | Seller NP |
| context.bap_id | string | Y | Buyer NP subscriber ID | Seller NP |
| context.bap_uri | string | Y | Buyer NP callback URI | Seller NP |
| context.bpp_id | string | Y | Seller NP subscriber ID | Seller NP |
| context.bpp_uri | string | Y | Seller NP callback URI | Seller NP |
| context.transaction_id | string | Y | Transaction ID from request | Seller NP |
| context.message_id | string | Y | Message ID from request | Seller NP |
| context.timestamp | string | Y | Response timestamp | Seller NP |
| message.order.provider.id | string | Y | Provider ID | Seller NP |
| message.order.provider.locations[].id | string | Y | Location ID | Seller NP |
| message.order.items[].id | string | Y | Item ID | Seller NP |
| message.order.items[].fulfillment_id | string | Y | Fulfillment ID | Seller NP |
| message.order.fulfillments[].id | string | Y | Fulfillment ID | Seller NP |
| message.order.fulfillments[].type | string | Y | Fulfillment type | Seller NP |
| message.order.fulfillments[].start.location.gps | string | Y | Pickup GPS | Seller NP |
| message.order.fulfillments[].start.location.address.name | string | Y | Pickup address details | Seller NP |
| message.order.fulfillments[].start.location.address.building | string | Y | Pickup building | Seller NP |
| message.order.fulfillments[].start.location.address.locality | string | Y | Pickup locality | Seller NP |
| message.order.fulfillments[].start.location.address.city | string | Y | Pickup city | Seller NP |
| message.order.fulfillments[].start.location.address.state | string | Y | Pickup state | Seller NP |
| message.order.fulfillments[].start.location.address.country | string | Y | Pickup country | Seller NP |
| message.order.fulfillments[].start.location.address.area_code | string | Y | Pickup pincode | Seller NP |
| message.order.fulfillments[].start.contact.phone | string | Y | Pickup phone | Seller NP |
| message.order.fulfillments[].start.contact.email | string | Y | Pickup email | Seller NP |
| message.order.fulfillments[].end.location.gps | string | Y | Dropoff GPS | Seller NP |
| message.order.fulfillments[].end.location.address.name | string | Y | Dropoff address details | Seller NP |
| message.order.fulfillments[].end.location.address.building | string | Y | Dropoff building | Seller NP |
| message.order.fulfillments[].end.location.address.locality | string | Y | Dropoff locality | Seller NP |
| message.order.fulfillments[].end.location.address.city | string | Y | Dropoff city | Seller NP |
| message.order.fulfillments[].end.location.address.state | string | Y | Dropoff state | Seller NP |
| message.order.fulfillments[].end.location.address.country | string | Y | Dropoff country | Seller NP |
| message.order.fulfillments[].end.location.address.area_code | string | Y | Dropoff pincode | Seller NP |
| message.order.fulfillments[].end.contact.phone | string | Y | Dropoff phone | Seller NP |
| message.order.fulfillments[].end.contact.email | string | Y | Dropoff email | Seller NP |
| message.order.fulfillments[].tags | array | N | Optional fulfillment metadata tags | Seller NP |
| message.order.quote.price.currency | string | Y | Currency code | Seller NP |
| message.order.quote.price.value | string | Y | Total quote value (tax inclusive) | Seller NP |
| message.order.quote.breakup[].@ondc/org/item_id | string | Y | Item ID for breakup | Seller NP |
| message.order.quote.breakup[].@ondc/org/title_type | string | Y | Breakup type (delivery/tax) | Seller NP |
| message.order.quote.breakup[].price.currency | string | Y | Currency for breakup item | Seller NP |
| message.order.quote.breakup[].price.value | string | Y | Value for breakup item | Seller NP |
| message.order.quote.ttl | string | Y | Quote validity TTL | Seller NP |
| message.order.payment.@ondc/org/collection_amount | string | N | Collection amount | Seller NP |
| message.order.payment.type | string | Y | Payment type | Seller NP |
| message.order.payment.collected_by | string | Y | Payment collector | Seller NP |
| message.order.payment.@ondc/org/settlement_details | array | N | Settlement details | Seller NP |
| message.order.cancellation_terms[].fulfillment_state.descriptor.code | string | Y | Fulfillment state for cancellation | Seller NP |
| message.order.cancellation_terms[].fulfillment_state.descriptor.short_desc | string | Y | Reason codes | Seller NP |
| message.order.cancellation_terms[].cancellation_fee.percentage | string | Y | Cancellation fee percentage | Seller NP |
| message.order.cancellation_terms[].cancellation_fee.amount.currency | string | Y | Fee currency | Seller NP |
| message.order.cancellation_terms[].cancellation_fee.amount.value | string | Y | Fee amount | Seller NP |
| message.order.tags | array | Y | BPP terms and conditions | Seller NP |

## 7. State & Correlation
- transaction_id: Unique identifier for the init transaction, set by Buyer NP. Seller NP must use the same transaction_id in ACK, NACK, and callback responses
- message_id: Unique identifier for this init request, set by Buyer NP. Seller NP must echo this in ACK/NACK responses
- Correlation: Seller NP correlates the callback to the original request using transaction_id + message_id. The callback should be sent to bap_uri endpoint
- No order_id exists yet as order is not confirmed

## 8. Validation Rules
- Mandatory field checks: All fields marked as Required=Y in the schema must be present
- Enum validations: fulfillment.type must be Delivery, payment.type must be valid, cancellation_fee percentage must be valid
- Timestamp & TTL validation: context.timestamp must be current time, response must be sent within TTL window (PT30S)
- Stale request handling: If timestamp is older than acceptable window, respond with NACK error code 65003
- Signing verification: Verify request signature using Buyer's public key from registry
- Serviceability validation: Re-validate pickup and dropoff locations for final confirmation
- Quote validation: Ensure quote matches catalog pricing and itemized tax breakup is correct

## 9. Error Scenarios

| Scenario | Error Code | When it occurs | Seller NP Action |
|----------|------------|----------------|------------------|
| Invalid request payload | 40001 | JSON schema validation fails | Send NACK with DOMAIN-ERROR |
| Service not available | 60001 | Locations not serviceable | Send NACK with LSP-ERROR |
| Internal server error | 50001 | Unexpected system failure | Send NACK with LSP-ERROR |
| Stale request | 65003 | Request timestamp too old | Send NACK with PROTOCOL-ERROR |
| Invalid signature | 401 | Signature verification fails | Return HTTP 401 |
| Quote mismatch | 60002 | Quote doesn't match catalog | Send NACK with LSP-ERROR |

## 10. Important Notes (Seller NP)
- Seller NP MUST perform final serviceability check before sending ACK to /init
- Seller NP MUST provide detailed cancellation terms for all fulfillment states
- Seller NP MUST include tax breakup in quote (separate delivery and tax line items)
- Seller NP MUST provide quote.ttl for quote validity duration
- Seller NP MUST validate that selected items match the catalog sent in /on_search
- Seller NP MUST perform final serviceability validation (pickup + dropoff locations)
- Seller NP MUST recompute pricing if distance or payload differs from /on_search
- Seller NP MUST include bpp_terms tags with liability, arbitration, and jurisdiction details
- Common mistake to avoid: Not re-validating serviceability or sending quote without proper tax breakup

## 11. P2P Constraints (Seller NP)

- Seller NP supports ONLY P2P (Point-to-Point) delivery
- No hub routing (P2H2P) is supported
- No rider availability guarantees are provided
- Pickup time is best-effort, not SLA-bound
- Distance-based pricing uses internal distance calculation (motorable distance optional, not required to be surfaced)
