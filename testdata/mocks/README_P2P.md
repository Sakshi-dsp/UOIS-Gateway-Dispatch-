# P2P Delivery Mock Payloads

## Context

**Dispatch is a logistics seller NP (BPP) for P2P (Point-to-Point) delivery only.**

All mock payloads in this directory reflect P2P delivery characteristics as per ONDC Logistics API Contract v1.2.0.

## P2P vs P2H2P

### P2P (Point-to-Point) - Used by Dispatch
- **Delivery Type**: Direct delivery from pickup to drop location
- **Routing**: No hub routing - rider delivers directly
- **AWB Number**: **NOT required** (only for P2H2P)
- **Packaging**: Standard packaging (no hub-specific requirements)
- **Item Descriptor Code**: `"P2P"`

### P2H2P (Point-to-Hub-to-Point) - NOT Used by Dispatch
- **Delivery Type**: Package routed through a hub
- **Routing**: Pickup → Hub → Drop
- **AWB Number**: **Required** for hub routing
- **Packaging**: Hub-specific packaging requirements
- **Item Descriptor Code**: `"P2H2P"`

## Mock Payload Characteristics

All mock payloads in this directory:

1. **Use `"P2P"` code** in `item.descriptor.code` and `fulfillment` items
2. **Do NOT include AWB numbers** (AWB only required for P2H2P)
3. **Use direct GPS coordinates** for pickup and drop locations
4. **Reflect immediate/same-day delivery** (typical for P2P hyperlocal)

## Payload Locations

- **ONDC Requests**: `testdata/mocks/ondc/requests/`
  - `search.json` - P2P serviceability search
  - `init.json` - P2P quote initialization
  - `confirm.json` - P2P order confirmation

- **ONDC Callbacks**: `testdata/mocks/ondc/callbacks/`
  - `on_search.json` - P2P catalog response
  - `on_init.json` - P2P quote response
  - `on_confirm.json` - P2P order acceptance

## Validation

When using these mock payloads:

1. ✅ Verify `item.descriptor.code` = `"P2P"` (not `"P2H2P"`)
2. ✅ Verify `fulfillment.type` = `"Delivery"` (not hub-based)
3. ✅ Verify AWB number is **NOT** present (P2P doesn't require AWB)
4. ✅ Verify GPS coordinates are provided for both pickup and drop
5. ✅ Verify delivery time windows are appropriate for P2P (immediate/same-day)

## References

- [ONDC API Contract for Logistics v1.2.0](../../docs/production-docs/ONDC%20-%20API%20Contract%20for%20Logistics%20(v1.2.0).md)
- [UOIS Gateway Functional Requirements](../../docs/production-docs/UOISGateway_FR.md)
- [Integration Boundary](../../docs/production-docs/INTEGRATION_BOUNDARY.md)

