# Development Stages — Platform-Agnostic NMS Agent

This document defines the development stages for the NMS Agent project. The AI coding agent must update this document whenever a task is started, completed, blocked, revised, or when the development scope changes.

## Status Legend

| Status | Meaning |
|---|---|
| `TODO` | Not started |
| `IN_PROGRESS` | Currently being worked on |
| `DONE` | Completed and validated |
| `BLOCKED` | Blocked by dependency, error, or design decision |
| `REVISE` | Implemented but needs correction |

## AI Update Rules

For every implementation task, the AI coding agent must:

1. Update the relevant task status in this document.
2. Add newly created files to `docs/KNOWLEDGE.md`.
3. Add a short summary to the **Development Log** section.
4. List validation commands that were executed.
5. Never mark a task as `DONE` unless build/test validation succeeds.
6. If a data, adapter, queue, or config contract changes, update the related document:
   - `docs/DATA_CONTRACT.md`
   - `docs/ADAPTER_CONTRACT.md`
   - `docs/QUEUE_DESIGN.md`
   - `docs/CONFIG_SCHEMA.md`

---

## Phase 0 — Project Foundation

**Target:** Prepare repository documentation, AI agent rules, and project structure so development remains controlled.

| Task | Target Output | Status | Notes |
|---|---|---|---|
| Setup `AGENTS.md` | AI agent rules are available | DONE | Already created |
| Setup architecture document | `docs/ARCHITECTURE.md` is available | DONE | Already created |
| Setup short PRD | `docs/PRD.md` is available | DONE | Already created |
| Setup AI context guide | `docs/AI_CONTEXT.md` is available | DONE | Already created |
| Setup repo map | `docs/REPO_MAP.md` is available | DONE | Already created |
| Setup knowledge table | `docs/KNOWLEDGE.md` exists and is updated when new files are added | IN_PROGRESS | Must be updated for every new file |
| Setup development stages | `docs/DEVELOPMENT_STAGES.md` is available | TODO | This document |

**Exit Criteria:**

- All core documents are available.
- The AI agent understands the architecture boundaries.
- Every new file can be tracked through `docs/KNOWLEDGE.md`.

---

## Phase 1 — Go Skeleton and Core Contracts

**Target:** Create the initial Go project skeleton and layer contracts without real implementation.

| Task | Target Output | Status | Notes |
|---|---|---|---|
| Initialize Go module | `go.mod` exists | DONE | Initial skeleton already created |
| Create agent entrypoint | `cmd/nms-agent/main.go` | DONE | Service entrypoint |
| Create CLI entrypoint | `cmd/nms-agentctl/main.go` | DONE | CLI entrypoint |
| Define canonical telemetry model | `internal/models/telemetry.go` | DONE | Must follow `DATA_CONTRACT.md` |
| Define collector port | `internal/collectors/port.go` | DONE | SNMP/ICMP collector contract |
| Define processor port | `internal/processors/port.go` | DONE | Preprocessing/normalization contract |
| Define queue port | `internal/queue/port.go` | DONE | Store-and-forward contract |
| Define adapter port | `internal/adapters/port.go` | DONE | Output adapter contract |
| Create pipeline orchestrator stub | `internal/core/pipeline.go` | DONE | Must follow the hexagonal flow |
| Validate skeleton | `go fmt`, `go test`, `go vet`, `go build` pass | DONE | Required before moving to Phase 2 |

**Exit Criteria:**

- The project can be built successfully.
- All layer contracts are available.
- Core code has no platform-specific dependency.

---

## Phase 2 — Configuration and CLI Foundation

**Target:** Provide configuration loading and basic CLI commands for validation, status, and reload.

| Task | Target Output | Status | Notes |
|---|---|---|---|
| Define config schema | `agent.yml`, `devices.d/*.yml`, `thresholds.yml`, `adapters.yml` | DONE | Updated `CONFIG_SCHEMA.md` with MVP |
| Implement config loader | Package `internal/config` | DONE | YAML loader + path resolution with `filepath` |
| Add config validation | Validate config before the agent starts | DONE | Basic required fields + duplicate device IDs |
| Add `nms-agentctl validate` | CLI config validation command | DONE | Loads config and exits non-zero on invalid |
| Add `nms-agentctl status` | CLI agent status command | TODO | |
| Add reload command | `nms-agentctl reload` triggers systemd reload | TODO | |
| Add SIGHUP handler | Agent can hot reload configuration | TODO | |

**Exit Criteria:**

- Config can be read and validated.
- Basic CLI commands are available.
- Config reload works without full restart.

---

## Phase 3 — Terminal Adapter and Dummy Pipeline

**Target:** Prove that the pipeline works without any external monitoring platform.

| Task | Target Output | Status | Notes |
|---|---|---|---|
| Implement terminal adapter | Print telemetry to terminal | DONE | Prints canonical telemetry lines |
| Implement dummy collector | Generate dummy telemetry | DONE | Deterministic RawSample generator |
| Implement minimal pipeline run | Collector → processor → queue stub → terminal adapter | DONE | One `RunOnce` demo pass |
| Add `nms-agent run` | Agent can run manually | DONE | Runs config load+validate + one dummy pipeline pass |
| Add basic logging | Startup, telemetry, and error logs | TODO | |

**Exit Criteria:**

- Agent can run without ThingsBoard/Zabbix.
- Terminal output shows telemetry.
- The architecture flow can be demonstrated.

---

## Phase 4 — SQLite Queue and Store-and-Forward

**Target:** Solve reliability issues by persisting telemetry at the agent level before sending.

| Task | Target Output | Status | Notes |
|---|---|---|---|
| Design SQLite schema | Queue table exists | TODO | Update `QUEUE_DESIGN.md` |
| Implement enqueue | Telemetry is stored before send | TODO | |
| Implement pending list | Worker can read pending data | TODO | |
| Implement mark sent/delete | Sent data is marked or deleted | TODO | |
| Implement retry count | Retry attempts are tracked | TODO | |
| Implement TTL cleanup | Old data can expire | TODO | |
| Add queue status CLI | `nms-agentctl queue status` | TODO | |
| Add retry CLI | `nms-agentctl queue retry` | TODO | |
| Add queue persistence test | Data survives service restart | TODO | |

**Exit Criteria:**

- Telemetry always enters the queue before sending.
- Pending data survives service restart.
- Send failure does not delete telemetry.
- Queue status is visible from CLI.

---

## Phase 5 — SNMP and ICMP Collectors

**Target:** Collect real monitoring data from network devices.

| Task | Target Output | Status | Notes |
|---|---|---|---|
| Implement ICMP collector | Reachability, latency, packet loss, jitter | TODO | |
| Implement basic SNMP collector | Uptime, interface status, traffic | TODO | |
| Add timeout handling | Slow devices do not block other polling tasks | TODO | |
| Add partial snapshot behavior | Valid data is still processed when some metrics fail | TODO | |
| Add collector tests | Basic unit/integration tests | TODO | |

**Exit Criteria:**

- Agent can collect real data using SNMP/ICMP.
- Timeout on one device does not stop the full pipeline.
- Output remains canonical telemetry.

---

## Phase 6 — Device Profile and Multi-Vendor Support

**Target:** Handle standard and vendor-specific OIDs efficiently.

| Task | Target Output | Status | Notes |
|---|---|---|---|
| Define standard profile | Standard OIDs for interface and uptime | TODO | Update `DEVICE_PROFILE.md` |
| Define vendor profile format | YAML profile for vendor-specific metrics | TODO | |
| Implement profile loader | Load profiles from `profiles/` | TODO | |
| Implement profile-based polling | Standard OID for common metrics, vendor OID for specific metrics | TODO | |
| Add profile validation | Validate OID and metric names | TODO | |
| Add profile cache | Avoid repeated vendor/profile detection on every polling cycle | TODO | Optional for MVP |

**Exit Criteria:**

- Devices can use profiles.
- OIDs are not hardcoded inside collectors.
- Agent supports the partial hybrid strategy.

---

## Phase 7 — Preprocessing and Threshold

**Target:** Convert raw metrics into monitoring-ready telemetry and generate alert events when thresholds are exceeded.

| Task | Target Output | Status | Notes |
|---|---|---|---|
| Implement throughput calculation | Interface counter → throughput | TODO | |
| Implement metric normalization | Raw values → canonical telemetry | TODO | |
| Implement threshold loader | Load `thresholds.yml` | TODO | |
| Implement threshold evaluator | Warning/critical status | TODO | |
| Add threshold CLI | `nms-agentctl threshold set/list` | TODO | |
| Add tests | Preprocessing and threshold tests | TODO | |

**Exit Criteria:**

- Main metrics are processed correctly.
- Thresholds are configurable.
- Alert event or telemetry status can be produced.

---

## Phase 8 — MQTT Adapters

**Target:** Send telemetry to platform consumers through MQTT.

| Task | Target Output | Status | Notes |
|---|---|---|---|
| Implement Generic MQTT adapter | Send canonical telemetry JSON | TODO | |
| Implement ThingsBoard MQTT adapter | Format `{deviceName,key,value,ts}` | TODO | |
| Add QoS configuration | QoS can be configured | TODO | |
| Add reconnect behavior | Adapter reconnects when connection drops | TODO | |
| Add adapter health check | Adapter status can be queried | TODO | |
| Add adapter tests | Formatting and send behavior are validated | TODO | |

**Exit Criteria:**

- Telemetry can be sent to an MQTT broker.
- Send failure keeps queue data pending.
- ThingsBoard can receive telemetry through the adapter.

---

## Phase 9 — Device Management CLI

**Target:** Allow admins to manage devices without manually editing files.

| Task | Target Output | Status | Notes |
|---|---|---|---|
| Add device list | `nms-agentctl device list` | TODO | |
| Add device add | `nms-agentctl device add` | TODO | |
| Add device update | `nms-agentctl device update` | TODO | |
| Add device remove | `nms-agentctl device remove` | TODO | |
| Add device test | `nms-agentctl device test` | TODO | Test SNMP/ICMP |
| Add validation before save | Invalid config is not saved | TODO | |
| Auto reload after change | CLI can trigger reload | TODO | |

**Exit Criteria:**

- Devices can be added, edited, and removed through CLI.
- Changes can be applied through reload.
- `docs/KNOWLEDGE.md` is updated if new files are created.

---

## Phase 10 — systemd Packaging

**Target:** Install and run the agent as a Linux service.

| Task | Target Output | Status | Notes |
|---|---|---|---|
| Create systemd unit | `packaging/systemd/nms-agent.service` | TODO | |
| Create install script | Copy binary, config, and service file | TODO | |
| Create directory structure | `/etc`, `/var/lib`, `/var/log`, `/opt` | TODO | |
| Add service reload support | `systemctl reload nms-agent` | TODO | |
| Add journal logging guide | Logging documentation | TODO | |
| Test reboot behavior | Agent auto-starts after reboot | TODO | |

**Exit Criteria:**

- Agent can be managed through systemd.
- Agent auto-starts after reboot.
- Reload and restart work correctly.

---

## Phase 11 — Reliability and Downtime Testing

**Target:** Prove that telemetry is not lost when the platform consumer connection is interrupted.

| Task | Target Output | Status | Notes |
|---|---|---|---|
| Simulate platform down | Adapter send fails | TODO | |
| Run 10-minute downtime test | 1-minute interval, expected 10 records | TODO | |
| Validate queue pending | All downtime records are stored | TODO | |
| Restore platform | Pending data is sent again | TODO | |
| Validate no duplicate | `event_id` prevents duplicate data | TODO | |
| Document result | Table and screenshots | TODO | |

**Exit Criteria:**

- A 10-minute downtime produces 10 stored records.
- After recovery, all 10 records are resent.
- No telemetry is lost at the agent level.

---

## Phase 12 — Hardening and Documentation

**Target:** Prepare the agent for demo, reporting, and future development.

| Task | Target Output | Status | Notes |
|---|---|---|---|
| Add README usage | Installation and usage instructions | TODO | |
| Add config examples | HQ/Branch device examples | TODO | |
| Add troubleshooting guide | Common errors and fixes | TODO | |
| Add security note | Credentials, `.env`, file permissions | TODO | |
| Run lint/vet/test/build | Final validation | TODO | |
| Create release artifact | Binary and sample config | TODO | |

**Exit Criteria:**

- Agent is ready for demo.
- Documentation is enough for user/admin operation.
- Binary can be used on a gateway.

---

## Development Log

The AI coding agent must add a new entry here after every change.

Format:

```text
YYYY-MM-DD HH:mm
Task:
Changed files:
Validation:
Status update:
Notes:
```

### Log Entries

```text
2026-05-21 22:30
Task: Initial skeleton generated
Changed files:
- go.mod
- cmd/nms-agent/main.go
- cmd/nms-agentctl/main.go
- internal/core/pipeline.go
- internal/models/telemetry.go
- internal/collectors/port.go
- internal/processors/port.go
- internal/queue/port.go
- internal/adapters/port.go
Validation:
- Pending confirmation from developer
Status update:
- Phase 1 mostly DONE
Notes:
- Next step is validating the skeleton with go fmt, go test, go vet, and go build.

2026-05-22 00:15
Task: Phase 2 config foundation (MVP)
Changed files:
- go.mod
- configs/agent.yml
- configs/devices.d/example-router.yml
- configs/thresholds.yml
- configs/adapters.yml
- internal/config/types.go
- internal/config/loader.go
- internal/config/validate.go
- internal/config/loader_test.go
- internal/config/validate_test.go
- cmd/nms-agentctl/main.go
- cmd/nms-agentctl/validate.go
- docs/CONFIG_SCHEMA.md
- docs/KNOWLEDGE.md
- docs/DEVELOPMENT_STAGES.md
Validation:
- go fmt ./...
- go test ./...
- go vet ./...
- go build ./...
- go run ./cmd/nms-agentctl validate --config configs/agent.yml
Status update:
- Phase 2: schema/loader/validation/validate command -> DONE
Notes:
- .env files are not loaded yet; only ${ENV_VAR} expansion is supported for path strings.
- Fixed sample `configs/agent.yml` paths to be relative to the agent.yml directory.

2026-05-22 00:40
Task: nms-agent startup-time config loading
Changed files:
- cmd/nms-agent/main.go
- docs/DEVELOPMENT_STAGES.md
Validation:
- go fmt ./...
- go test ./...
- go vet ./...
- go build ./...
- go run ./cmd/nms-agent run --config configs/agent.yml
- go run ./cmd/nms-agentctl validate --config configs/agent.yml
Status update:
- Phase 3: `nms-agent run` -> DONE (config load+validate only)
Notes:
- No collectors/queue/adapters are executed yet; run command stops after successful config validation.

2026-05-22 01:05
Task: Phase 3 terminal adapter and dummy pipeline
Changed files:
- internal/collectors/dummy_collector.go
- internal/processors/passthrough_processor.go
- internal/queue/memory_queue.go
- internal/adapters/terminal_adapter.go
- cmd/nms-agent/main.go
- docs/KNOWLEDGE.md
- docs/DEVELOPMENT_STAGES.md
Validation:
- go fmt ./...
- go test ./...
- go vet ./...
- go build ./...
- go run ./cmd/nms-agent run --config configs/agent.yml
Status update:
- Phase 3: terminal adapter/dummy collector/minimal pipeline run -> DONE
Notes:
- In-memory queue is a Phase 3 stub; durability is implemented in Phase 4 (SQLite).

2026-05-22 01:15
Task: Add make targets for validation
Changed files:
- Makefile
- make.bat
- docs/KNOWLEDGE.md
- docs/DEVELOPMENT_STAGES.md
Validation:
- cmd /c make check
Status update:
- Validation can be run via `make` in Windows and non-Windows environments.
Notes:
- On Windows, PowerShell does not have GNU make by default; `make.bat` provides equivalent targets.
```
