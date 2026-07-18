# Repo Map

## cmd/nms-agent
Main agent service entrypoint.

## cmd/nms-agentctl
CLI management tool.

## internal/collectors
SNMP and ICMP collectors.

## internal/discovery
Discovery runtime, explorer, and active/netlink providers.

## internal/configwatch
Runtime watchers for hot reload of `devices.d`.

## internal/profiles
Device/vendor profile loader.

## internal/processors
Derived metrics, normalization, and threshold logic.

## internal/queue
SQLite local queue and retry state.

## internal/adapters
TUI, ThingsBoard MQTT, Generic MQTT, and future adapters.

## internal/integrations
ThingsBoard REST-side integration helpers for relations, topology, and alarms.

## internal/models
Canonical telemetry structs.
