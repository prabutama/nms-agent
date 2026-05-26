# Device Profiles (Phase 6 MVP)

Device profiles define which SNMP OIDs should be collected per vendor/model.
They live in the `profiles/` directory and are loaded at runtime.

## Profile Selection

Profiles are selected per device using this precedence:

1. Exact `vendor` + `model`
2. Vendor default (`vendor` + empty `model`)
3. Standard profile (empty `vendor` and `model`)

## YAML Schema

Each profile file contains a single profile object:

```yaml
name: standard
match:
  vendor: ""
  model: ""

metrics:
  - metric: snmp.uptime_seconds
    oid: 1.3.6.1.2.1.1.3.0
    type: get
    unit: s

  - metric: snmp.if.oper_status
    oid: 1.3.6.1.2.1.2.2.1.8
    type: walk
    index: true
```

### Fields

- `name` (string, required): profile name.
- `match.vendor` (string): vendor match, empty means wildcard.
- `match.model` (string): model match, empty means wildcard.
- `metrics[]` (required): list of polled SNMP metrics.

Each metric:

- `metric` (string, required): canonical metric name.
- `oid` (string, required): numeric OID string.
- `type` (string, required): `get` or `walk`.
- `unit` (string, optional): unit tag.
- `index` (bool, optional): when `true`, adds `ifIndex` tag from walked OID suffix.

## Notes

- Profiles are loaded from `profiles/*.yml` at runtime.
- No cache is used in Phase 6; profiles are loaded on startup.
- SNMPv3 is not part of this MVP.
