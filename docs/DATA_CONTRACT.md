# Canonical Telemetry Contract

All collectors must output canonical telemetry.
Adapters must only accept canonical telemetry.

## Required Fields

- metric.name
- metric.value_type
- metric.value_number
- metric.value_string
- metric.unit
- timestamp
- source.protocol

## Threshold Tags (Phase 7)

When a threshold rule matches, the following tags are added to telemetry:

- threshold.status: ok|warning|critical
- threshold.matched: true
- threshold.rule: <metric>#<index>

## Derived Interface Metrics (Phase 7)

- snmp.if.rx_bps
- snmp.if.tx_bps
- snmp.if.rx_utilization_pct
- snmp.if.tx_utilization_pct

## Device Resource Metrics (Phase 6)

- snmp.host.cpu.load_pct
- snmp.host.memory.size_kb

## Device Identity Metrics (Phase 7)

- snmp.system.description
- snmp.system.name
- snmp.if.name

## Metric Value Types

- value_type: number|string (required)
- value_number: float (required when value_type=number)
- value_string: string (required when value_type=string)
