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

## Current Documentation Notes

- Use `README.md`, `docs/CLI_COMMANDS.md`, and code as current operational source of truth.
- `nms-agentctl queue retry` is currently limited. For empty or `tui` active adapter it uses no-op delivery; real adapter redelivery is not implemented for other adapters yet.
- `nms-agentctl view` depends on local socket `/run/nms-agent/view.sock` and is mainly intended for Linux/systemd-style runtime environments.
- Do not assume every `nms-agentctl` subcommand uses same default config path. Prefer `--config <path>` explicitly.

## Production Features (added in production readiness phase)

| Feature | Package / File | Config |
|---------|---------------|--------|
| Structured logging | `internal/logger/` | `agent.logging.level` / `agent.logging.format` |
| Runtime status | `internal/status/` | `nms-agentctl status [--watch]` |
| Retry backoff | `internal/queue/sqlite_queue.go` | `agent.delivery.retry.*` |
| Dead-letter | `internal/queue/sqlite_queue.go` | `max_retries` / `retention_days` |
| systemd hardening | `packaging/systemd/nms-agent.service` | `MemoryMax`, `LimitNOFILE`, `StartLimitBurst` |
| Failure isolation | `cmd/nms-agent/main.go` (`combinedCollector`) | Best-effort per-collector |
| CI quality gate | `Makefile` / `make.bat` | `make check-all` (race, lint, vuln) |

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
| `docs/PRODUCTION_ROADMAP.md` | Full phased production plan |
| `docs/BASELINE_0.md` | Test/behavior baseline |
| `docs/SOAK_TEST_PLAN.md` | Load & soak test procedures |
| `docs/SECURITY.md` | Security hardening guide |
