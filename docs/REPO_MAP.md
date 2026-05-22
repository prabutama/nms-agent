# Repo Map

## cmd/nms-agent
Main agent service entrypoint.

## cmd/nms-agentctl
CLI management tool.

## internal/collectors
SNMP and ICMP collectors.

## internal/profiles
Device/vendor profile loader.

## internal/processors
Throughput, normalization, and threshold logic.

## internal/queue
SQLite local queue and retry state.

## internal/adapters
Terminal, ThingsBoard MQTT, Generic MQTT, and future adapters.

## internal/models
Canonical telemetry structs.