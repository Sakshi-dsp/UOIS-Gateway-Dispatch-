# on_init API (Seller NP)

## 1. Overview
- Purpose of this API: This API allows Seller NP to provide final quote, terms, and conditions for the selected logistics services
- Who calls it (Buyer NP / Seller NP): Called by Seller NP (BPP) as callback response to /init
- Sync vs Async behavior: Asynchronous - This is the callback response sent after ACK to /init request
- Callback expectations: Buyer NP acknowledges receipt with ACK/NACK
- TTL behavior (if applicable): Response must be sent within TTL specified in original /init request

## 2. Role Perspective
We are Seller NP (BPP / Logistics Service Provider). In this API:
- We provide the final quote with itemized pricing and taxes
- We specify cancellation terms and conditions
- We confirm fulfillment details and serviceability
- We provide transaction-level terms for the order
- We send this as asynchronous callback to the buyer's /on_init endpoint
- We must ensure all terms are acceptable to proceed to order confirmation

## 3. Endpoint Details
- HTTP Method: POST
- Endpoint path: /on_init (at Buyer NP's bap_uri)
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

### 4.2 Field-Level Schema (Table)

| Field Path | Data Type | Required (Y/N) | Description | Source of Truth |
|------------|-----------|----------------|-------------|-----------------|
| context.domain | string | Y | Domain identifier for logistics (nic2004:60232) | Seller NP |
| context.country | string | Y | Country code (IND) | Seller NP |
| context.city | string | Y | City code (std:080) | Seller NP |
| context.action | string | Y | Action name (on_init) | Seller NP |
| context.core_version | string | Y | ONDC core version (1.2.0) | Seller NP |
| context.bap_id | string | Y | Buyer NP subscriber ID | Seller NP |
| context.bap_uri | string | Y | Buyer NP callback URI | Seller NP |
| context.bpp_id | string | Y | Seller NP subscriber ID | Seller NP |
| context.bpp_uri | string | Y | Seller NP callback URI | Seller NP |
| context.transaction_id | string | Y | Transaction ID from original init request | Seller NP |
| context.message_id | string | Y | Message ID from original init request | Seller NP |
| context.timestamp | string | Y | Response timestamp in RFC3339 format | Seller NP |
| message.order.provider.id | string | Y | Selected provider ID | Seller NP |
| message.order.provider.locations[].id | string | N | Selected location ID (mandatory only if provider.locations was returned in /on_search, not required for P2P) | Seller NP |
| message.order.items[].id | string | Y | Selected item ID | Seller NP |
| message.order.items[].fulfillment_id | string | Y | Associated fulfillment ID | Seller NP |
| message.order.fulfillments[].id | string | Y | Fulfillment unique identifier | Seller NP |
| message.order.fulfillments[].type | string | Y | Fulfillment type (Delivery) | Seller NP |
| message.order.fulfillments[].start.location.gps | string | Y | Pickup GPS coordinates | Seller NP |
| message.order.fulfillments[].start.location.address.name | string | Y | Pickup location name | Seller NP |
| message.order.fulfillments[].start.location.address.building | string | Y | Pickup building name | Seller NP |
| message.order.fulfillments[].start.location.address.locality | string | Y | Pickup locality | Seller NP |
| message.order.fulfillments[].start.location.address.city | string | Y | Pickup city | Seller NP |
| message.order.fulfillments[].start.location.address.state | string | Y | Pickup state | Seller NP |
| message.order.fulfillments[].start.location.address.country | string | Y | Pickup country | Seller NP |
| message.order.fulfillments[].start.location.address.area_code | string | Y | Pickup pincode | Seller NP |
| message.order.fulfillments[].start.contact.phone | string | Y | Pickup contact phone | Seller NP |
| message.order.fulfillments[].start.contact.email | string | N | Pickup contact email (optional) | Seller NP |
| message.order.fulfillments[].end.location.gps | string | Y | Dropoff GPS coordinates | Seller NP |
| message.order.fulfillments[].end.location.address.name | string | Y | Dropoff location name | Seller NP |
| message.order.fulfillments[].end.location.address.building | string | Y | Dropoff building name | Seller NP |
| message.order.fulfillments[].end.location.address.locality | string | Y | Dropoff locality | Seller NP |
| message.order.fulfillments[].end.location.address.city | string | Y | Dropoff city | Seller NP |
| message.order.fulfillments[].end.location.address.state | string | Y | Dropoff state | Seller NP |
| message.order.fulfillments[].end.location.address.country | string | Y | Dropoff country | Seller NP |
| message.order.fulfillments[].end.location.address.area_code | string | Y | Dropoff pincode | Seller NP |
| message.order.fulfillments[].end.contact.phone | string | Y | Dropoff contact phone | Seller NP |
| message.order.fulfillments[].end.contact.email | string | N | Dropoff contact email (optional) | Seller NP |
| message.order.fulfillments[].tags | array | N | Optional fulfillment metadata tags | Seller NP |
| message.order.quote.price.currency | string | Y | Currency code (INR) | Seller NP |
| message.order.quote.price.value | string | Y | Total quote value (tax inclusive) | Seller NP |
| message.order.quote.breakup[].@ondc/org/item_id | string | Y | Item ID for price breakup | Seller NP |
| message.order.quote.breakup[].@ondc/org/title_type | string | Y | Breakup type (delivery/tax) | Seller NP |
| message.order.quote.breakup[].price.currency | string | Y | Currency for breakup item | Seller NP |
| message.order.quote.breakup[].price.value | string | Y | Value for breakup item | Seller NP |
| message.order.quote.ttl | string | Y | Quote validity duration | Seller NP |
| message.order.payment.@ondc/org/collection_amount | string | N | Collection amount for CoD | Seller NP |
| message.order.payment.type | string | Y | Payment type | Seller NP |
| message.order.payment.collected_by | string | Y | Payment collector (BPP/BAP) | Seller NP |
| message.order.payment.@ondc/org/settlement_details | array | N | Settlement details | Seller NP |
| message.order.cancellation_terms[].fulfillment_state.descriptor.code | string | Y | Fulfillment state for cancellation | Seller NP |
| message.order.cancellation_terms[].fulfillment_state.descriptor.short_desc | string | Y | Reason codes for cancellation | Seller NP |
| message.order.cancellation_terms[].cancellation_fee.percentage | string | Y | Cancellation fee percentage | Seller NP |
| message.order.cancellation_terms[].cancellation_fee.amount.currency | string | Y | Fee currency | Seller NP |
| message.order.cancellation_terms[].cancellation_fee.amount.value | string | Y | Fee amount | Seller NP |
| message.order.tags[].code | string | Y | Tag code (bpp_terms) | Seller NP |
| message.order.tags[].list[].code | string | Y | Terms code | Seller NP |
| message.order.tags[].list[].value | string | Y | Terms value | Seller NP |

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
      "code": "62501",
      "message": "Cancellation terms not acceptable"
    }
  }
}
```

## 6. Asynchronous Callback (if applicable)
This API does not have an asynchronous callback - it is itself the callback response to /init.

## 7. State & Correlation
- transaction_id: Must match the transaction_id from the original /init request
- message_id: Must match the message_id from the original /init request
- Correlation: Buyer NP correlates this callback to the original init using transaction_id + message_id
- No order_id exists yet as order is not confirmed

## 8. Validation Rules
- Mandatory field checks: All fields marked as Required=Y in the schema must be present
- Enum validations: fulfillment.type must be Delivery, payment.type must be valid, cancellation_fee percentage must be valid
- Timestamp & TTL validation: context.timestamp must be current time and within TTL window from original request
- Stale request handling: If timestamp is older than acceptable window, Buyer NP may NACK with error code 65003
- Signing verification: Buyer NP must verify Seller NP's signature using public key from registry
- Terms validation: Cancellation terms must be reasonable and bpp_terms must be complete
- Quote validation: Quote breakup must be accurate and TTL must be reasonable

## 9. Error Scenarios

| Scenario | Error Code | When it occurs | Seller NP Action |
|----------|------------|----------------|------------------|
| Cancellation terms not acceptable | 62501 | Buyer NP doesn't accept terms | Buyer NP sends NACK |
| Invalid quote breakup | 62502 | Quote format or values invalid | Buyer NP sends NACK |
| Service temporarily unavailable | 50001 | Internal LSP system issues | Buyer NP sends NACK |
| Invalid terms format | 40001 | bpp_terms structure invalid | Buyer NP sends NACK |
| Stale callback | 65003 | Callback sent after TTL expired | Buyer NP sends NACK |

## 10. Important Notes (Seller NP)
- Seller NP MUST provide comprehensive cancellation terms for all fulfillment states
- Seller NP MUST itemize taxes separately from delivery charges in quote.breakup
- Seller NP MUST include bpp_terms with liability limits, arbitration, jurisdiction, and static terms
- Seller NP MUST set appropriate quote.ttl (typically PT15M)
- Seller NP MUST validate that quote matches catalog pricing and includes proper tax calculation
- Seller NP MUST ensure all address fields are properly formatted and complete
- Common mistake to avoid: Incomplete cancellation terms or missing bpp_terms tags
