# Device Profiles

Device profiles define which raw SNMP OIDs should be collected per vendor/model.
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
- `metrics[]` (required): list of polled raw SNMP metrics.

Each metric:

- `metric` (string, required): canonical metric name.
- `oid` (string, required): numeric OID string.
- `type` (string, required): `get` or `walk`.
- `unit` (string, optional): unit tag.
- `index` (bool, optional): when `true`, adds `ifIndex` tag from walked OID suffix. This is used for interface tables and also hrStorage tables.

## Notes

- Profiles are loaded from `profiles/*.yml` at runtime.
- Device profiles only describe raw source OIDs. Derived metrics such as interface utilization, memory `used_pct`, or storage `used_pct`/`*_bytes` are created later by preprocessors and must not be added to profiles.
- Vendor-specific OIDs are allowed when standard MIBs are missing or not useful on a platform.
- No cache is used; profiles are loaded on startup.
- SNMPv3 is not part of this MVP.
