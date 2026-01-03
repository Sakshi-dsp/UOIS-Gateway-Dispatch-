# ONDC-DISPATCH Contracts

## Dispatch Contracts

1. **⚠️ Important:** **UOIS (Dispatch) does NOT accept COD (ON-FULFILLMENT) payment type.** Orders with `payment.type: "ON-FULFILLMENT"` will be rejected.

2. **⚠️ Important:** **UOIS (Dispatch) only supports specific delivery categories:**
   - ✅ `"Immediate Delivery"` - Supported (requires `time.duration <= PT60M` if duration is provided)
   - ✅ `"Standard Delivery"` - Supported only with immediate subcategory (requires `time.duration <= PT60M`)
   - ❌ `"Same Day Delivery"` - **NOT SUPPORTED** (will be rejected with error code 66002)
   - ❌ `"Next Day Delivery"` - **NOT SUPPORTED** (will be rejected with error code 66002)
   - ❌ `"Express Delivery"` - **NOT SUPPORTED** (will be rejected with error code 66002)
   
   **Validation Rules:**
   - If Buyer NP sends unsupported delivery categories (e.g., "Same Day Delivery", "Next Day Delivery", "Express Delivery"), Seller NP responds with NACK: `"Order Validation Failed: unsupported delivery category '<category>'. Only 'Immediate Delivery' or 'Standard Delivery' with immediate subcategory (time.duration <= PT60M) is supported"`
   - For "Standard Delivery" without immediate subcategory (duration > PT60M or missing), the response is: `"Order Validation Failed: unsupported delivery category 'Standard Delivery'. Standard Delivery is only accepted with immediate subcategory (time.duration <= PT60M)"`
   - **Error Code**: `66002` (order validation failure)
   - **Reference**: ONDC API Contract for Logistics (v1.2.0), lines 549-552

3. **⚠️ Important:** **Billing Information Storage:**
   - Billing information is **NOT** returned by the orchestrator in quote responses
   - Billing is ONDC-specific and comes from the Buyer NP in the `/init` request
   - When `/init` request is received, billing is extracted from `message.order.billing` and stored in Redis:
     - **Redis Key:** `ondc_billing:{transaction_id}`
     - **TTL:** 24 hours
     - **Storage Location:** `internal/services/ondc/storage/billing_storage_service.go`
   - Billing information is used in `/on_confirm` and other post-order APIs (per ONDC spec footnote [^118]: "billing should be same as in /init")
   - **Reference**: ONDC API Contract for Logistics (v1.2.0), lines 697-715 (billing in /init), footnote [^55] (billing details for invoicing), footnote [^118] (billing consistency)

4. **⚠️ Important:** **Fulfillment Contacts Storage:**
   - Fulfillment contacts (`start.contact` and `end.contact`) are provided by the Buyer NP in `/init` and `/confirm` requests
   - These contacts represent the pickup location contact (merchant/warehouse staff) and delivery location contact (customer)
   - When `/init` or `/confirm` request is received, fulfillment contacts are extracted from `message.order.fulfillments[0].start.contact` and `message.order.fulfillments[0].end.contact` and stored in Redis:
     - **Redis Key:** `ondc_fulfillment_contacts:{transaction_id}`
     - **TTL:** 30 days (same as order mapping)
     - **Storage Location:** `internal/services/ondc/storage/fulfillment_contacts_storage_service.go`
     - **Extraction Function:** `internal/utils/ondc/request_parser.go::ExtractFulfillmentContactsFromRequest`
   - When building `/on_init`, `/on_confirm`, `/on_status`, and `/on_cancel` responses, fulfillment contacts are retrieved from Redis using `transaction_id` and used to populate `fulfillment.start.contact` and `fulfillment.end.contact` fields
   - This ensures that the Buyer NP receives the same contact information they originally provided, rather than trying to extract contacts from orchestrator location data (which may have different structure or missing fields)
   - **Contact Fields:** Each contact object includes:
     - `name`: Person name (optional)
     - `phone`: Contact phone number
     - `email`: Contact email address
   - **Reference**: ONDC API Contract for Logistics (v1.2.0), lines 667-671, 689-693 (`/init` request), lines 835-839, 857-861 (`/on_init` response), lines 1554-1558, 1604-1608 (`/on_confirm` response), lines 3689-3693, 3750-3754 (`/on_status` response)

## ONDC Contracts

1. *⚠️ **ONDC Contract Requirement - `/on_search` Pricing:**
   - Each item in `/on_search` must include a `price` object with:
     - `currency` (e.g., "INR")
     - `value` (e.g., "59.00")
   - **Footnote [^41]**: "price is tax-inclusive here, itemized in /on_init"
   **the rate card basePrice should includes tax, or needed to  add explicit tax calculation to ensure tax-inclusive pricing per ONDC spec.**
   - **Reference**: ONDC API Contract for Logistics (v1.2.0), lines 499-503

See [ONDC - API Contract for Logistics (v1.2.0).md](./ONDC%20-%20API%20Contract%20for%20Logistics%20(v1.2.0).md) for complete ONDC API contract specifications.

