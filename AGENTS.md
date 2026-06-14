# NMS Agent

A Go agent deployed per site. It polls network devices via SNMP/ICMP, stores
data in local SQLite, then sends to any monitoring platform (ThingsBoard, MQTT,
TUI) through pluggable adapters.

## Core Flow

```
Collect (SNMP/ICMP) → Preprocess & Normalize → Queue (SQLite) → Adapter Send
```

Each stage is a Go interface: `collectors.Collector`, `processors.Processor`,
`queue.Queue`, `adapters.Adapter`. Telemetry must hit the queue before send.
Failed deliveries stay pending for retry.


Windows: use `make.bat` instead. No CGO (uses `modernc.org/sqlite`).

## Deeper Docs

| File | Covers |
|---|---|
| `docs/AI_CONTEXT.md` | What to read per task type |
| `docs/ARCHITECTURE.md` | Hexagonal design rules |
| `docs/DATA_CONTRACT.md` | Canonical telemetry format |
| `docs/ADAPTER_CONTRACT.md` | Adapter interface rules |
| `docs/QUEUE_DESIGN.md` | SQLite queue data model |
| `docs/CONFIG_SCHEMA.md` | YAML config reference |
| `docs/DEVICE_PROFILE.md` | SNMP profile YAML schema |
| `docs/CLI_COMMANDS.md` | All nms-agentctl commands |
| `docs/DEVELOPMENT_WORKFLOW.md` | Dev flow and conventions |
