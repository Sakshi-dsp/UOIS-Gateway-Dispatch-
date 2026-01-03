# ONDC-DISPATCH Contracts

## Dispatch Contracts

1. **⚠️ Important:** **UOIS (Dispatch) does NOT accept COD (ON-FULFILLMENT) payment type.** Orders with `payment.type: "ON-FULFILLMENT"` will be rejected.

## ONDC Contracts

1. *⚠️ **ONDC Contract Requirement - `/on_search` Pricing:**
   - Each item in `/on_search` must include a `price` object with:
     - `currency` (e.g., "INR")
     - `value` (e.g., "59.00")
   - **Footnote [^41]**: "price is tax-inclusive here, itemized in /on_init"
   **the rate card basePrice should includes tax, or needed to  add explicit tax calculation to ensure tax-inclusive pricing per ONDC spec.**
   - **Reference**: ONDC API Contract for Logistics (v1.2.0), lines 499-503

See [ONDC - API Contract for Logistics (v1.2.0).md](./ONDC%20-%20API%20Contract%20for%20Logistics%20(v1.2.0).md) for complete ONDC API contract specifications.

