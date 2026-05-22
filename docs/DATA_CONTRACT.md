# Canonical Telemetry Contract

All collectors must output canonical telemetry.
Adapters must only accept canonical telemetry.

## Required Fields

- schema_version
- event_id
- device.name
- device.ip
- device.site
- metric.name
- metric.value
- metric.unit
- timestamp
- source.protocol