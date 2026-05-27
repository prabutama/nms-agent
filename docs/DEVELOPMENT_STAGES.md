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
| Design SQLite schema | Queue table exists | DONE | `queue_items` schema in `QUEUE_DESIGN.md` |
| Implement enqueue | Telemetry is stored before send | DONE | SQLite insert, JSON payload |
| Implement pending list | Worker can read pending data | DONE | Oldest-first pending reads |
| Implement mark sent/delete | Sent data is marked or deleted | DONE | Delete by queue item IDs |
| Implement retry count | Retry attempts are tracked | DONE | `MarkFailed` increments `retry_count` |
| Implement TTL cleanup | Old data can expire | TODO | |
| Add queue status CLI | `nms-agentctl queue status` | DONE | Prints pending count and max retry_count |
| Add retry CLI | `nms-agentctl queue retry` | DONE | Retries pending batch and ACKs by IDs |
| Add queue persistence test | Data survives service restart | DONE | Unit tests for restart persistence |

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
| Implement ICMP collector | Reachability, latency, packet loss, jitter | DONE | Uses system `ping` with parse + jitter (peak-to-peak) |
| Implement basic SNMP collector | Uptime, interface status, traffic | DONE | GoSNMP: sysUpTime + ifOperStatus + ifHC octets |
| Add timeout handling | Slow devices do not block other polling tasks | DONE | Per-target timeouts and ctx-aware bounds |
| Add partial snapshot behavior | Valid data is still processed when some metrics fail | DONE | Best-effort parse/walk; does not fail whole pass |
| Add collector tests | Basic unit/integration tests | DONE | Unit tests with injected exec/fake SNMP client |

**Exit Criteria:**

- Agent can collect real data using SNMP/ICMP.
- Timeout on one device does not stop the full pipeline.
- Output remains canonical telemetry.

---

## Phase 6 — Device Profile and Multi-Vendor Support

**Target:** Handle standard and vendor-specific OIDs efficiently.

| Task | Target Output | Status | Notes |
|---|---|---|---|
| Define standard profile | Standard OIDs for interface and uptime | DONE | `profiles/standard.yml` |
| Define vendor profile format | YAML profile for vendor-specific metrics | DONE | `profiles/vendor-example.yml` |
| Implement profile loader | Load profiles from `profiles/` | DONE | `internal/profiles/loader.go` |
| Implement profile-based polling | Standard OID for common metrics, vendor OID for specific metrics | DONE | SNMP collector uses profile metrics |
| Add profile validation | Validate OID and metric names | DONE | `internal/profiles/validate.go` |
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
| Implement threshold loader | Load `thresholds.yml` | DONE | Threshold rules parsed in config loader |
| Implement threshold evaluator | Warning/critical status | DONE | Preprocess processor tags telemetry |
| Add threshold CLI | `nms-agentctl threshold set/list` | TODO | |
| Add tests | Preprocessing and threshold tests | DONE | Processor threshold unit tests |

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
- configs/devices.d/mikrotik-routeros.yml
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

2026-05-22 02:05
Task: Phase 4A SQLite queue core
Changed files:
- internal/queue/port.go
- internal/queue/memory_queue.go
- internal/queue/sqlite_queue.go
- internal/queue/sqlite_queue_test.go
- internal/core/pipeline.go
- go.mod
- go.sum
- docs/QUEUE_DESIGN.md
- docs/KNOWLEDGE.md
- docs/DEVELOPMENT_STAGES.md
Validation:
- cmd /c make fmt
- cmd /c make test
- cmd /c make build
- cmd /c make check
Status update:
- Phase 4: schema/enqueue/pending/delivered/retry/persistence test -> DONE
Notes:
- Queue port updated to use QueueItem IDs and MarkFailed for retry tracking.

2026-05-22 02:20
Task: Phase 4 wire SQLite queue into nms-agent runtime
Changed files:
- internal/config/types.go
- internal/config/loader.go
- internal/config/validate.go
- configs/agent.yml
- docs/CONFIG_SCHEMA.md
- docs/QUEUE_DESIGN.md
- cmd/nms-agent/main.go
- internal/core/pipeline_sqlite_test.go
- docs/KNOWLEDGE.md
- docs/DEVELOPMENT_STAGES.md
Validation:
- cmd /c make fmt
- cmd /c make test
- cmd /c make build
Status update:
- Phase 4 runtime wiring -> DONE
Notes:
- Runtime auto-creates the queue DB parent directory only.

2026-05-22 02:35
Task: Phase 4 queue status CLI
Changed files:
- cmd/nms-agentctl/main.go
- cmd/nms-agentctl/queue_status.go
- internal/queue/sqlite_queue_stats.go
- docs/KNOWLEDGE.md
- docs/DEVELOPMENT_STAGES.md
Validation:
- cmd /c make fmt
- cmd /c make test
- cmd /c make build
- cmd /c make check
Status update:
- Phase 4: queue status CLI -> DONE
Notes:
- Command reads `paths.queue_db` from config and prints stable one-line output.

2026-05-22 02:55
Task: Phase 4 queue retry CLI
Changed files:
- cmd/nms-agentctl/main.go
- cmd/nms-agentctl/queue_retry.go
- cmd/nms-agentctl/queue_retry_test.go
- docs/KNOWLEDGE.md
- docs/DEVELOPMENT_STAGES.md
Validation:
- cmd /c make fmt
- cmd /c make test
- cmd /c make build
- cmd /c make check
Status update:
- Phase 4: retry CLI -> DONE
Notes:
- Current adapter support: terminal only (others will error as not implemented).

2026-05-22 03:05
Task: Update FLOW diagram (queue status + retry paths)
Changed files:
- docs/FLOW.md
- docs/KNOWLEDGE.md
- docs/DEVELOPMENT_STAGES.md
Validation:
- cmd /c make fmt
- cmd /c make test
- cmd /c make build
- cmd /c make check
Status update:
- Docs: FLOW diagram reflects current CLI and retry behavior.
Notes:
- Diagram now includes `nms-agentctl queue retry` and routes CLI commands through config load/validate.

2026-05-23 09:10
Task: Phase 5 ICMP + SNMP collectors (MVP)
Changed files:
- internal/collectors/targets.go
- internal/collectors/icmp_collector.go
- internal/collectors/icmp_collector_test.go
- internal/collectors/snmp_collector.go
- internal/collectors/snmp_collector_test.go
- internal/processors/passthrough_processor.go
- go.mod
- go.sum
- docs/KNOWLEDGE.md
- docs/DEVELOPMENT_STAGES.md
Validation:
- cmd /c make fmt
- cmd /c make test
- cmd /c make build
Status update:
- Phase 5: ICMP/SNMP/timeout/partial snapshot/tests -> DONE
Notes:
- ICMP collector uses system `ping` for unprivileged portability.
- SNMP collector uses GoSNMP for sysUpTime + basic interface metrics.

2026-05-23 09:40
Task: Phase 5 wire ICMP/SNMP collectors into runtime
Changed files:
- cmd/nms-agent/main.go
- internal/config/types.go
- internal/config/validate.go
- configs/devices.d/mikrotik-routeros.yml
- docs/CONFIG_SCHEMA.md
- docs/FLOW.md
- docs/KNOWLEDGE.md
- docs/DEVELOPMENT_STAGES.md
Validation:
- cmd /c make fmt
- cmd /c make test
- cmd /c make build
- cmd /c make check
- go run ./cmd/nms-agent run --config configs/agent.yml --collector-mode dummy
- go run ./cmd/nms-agent run --config configs/agent.yml --collector-mode auto
Status update:
- Phase 5: runtime wiring -> DONE
Notes:
- `--collector-mode auto|dummy|real` selects Dummy vs combined ICMP/SNMP collectors.

2026-05-23 11:20
Task: Phase 6 device profile MVP (no cache)
Changed files:
- internal/profiles/types.go
- internal/profiles/loader.go
- internal/profiles/validate.go
- internal/profiles/loader_test.go
- internal/profiles/validate_test.go
- profiles/standard.yml
- profiles/vendor-example.yml
- internal/collectors/targets.go
- internal/collectors/snmp_collector.go
- internal/collectors/snmp_collector_test.go
- cmd/nms-agent/main.go
- docs/DEVICE_PROFILE.md
- docs/KNOWLEDGE.md
- docs/DEVELOPMENT_STAGES.md
Validation:
- cmd /c make fmt
- cmd /c make test
- cmd /c make build
- cmd /c make check
- go run ./cmd/nms-agent run --config configs/agent.yml --collector-mode real
Status update:
- Phase 6: profile schema/loader/validation/polling -> DONE
Notes:
- Profiles are loaded from `profiles/` and selected by vendor/model precedence.

2026-05-23 11:45
Task: Phase 6 profile-driven SNMP test
Changed files:
- internal/collectors/snmp_collector_test.go
- docs/DEVELOPMENT_STAGES.md
Validation:
- cmd /c make fmt
- cmd /c make test
- cmd /c make build
- cmd /c make check
Status update:
- Phase 6: profile-driven SNMP test -> DONE
Notes:
- Test loads profile YAML from temp dir and verifies OID usage + ifIndex tagging.

2026-05-26 12:05
Task: Phase 7 threshold MVP (preprocess + tags)
Changed files:
- internal/models/thresholds.go
- internal/config/types.go
- internal/config/validate.go
- internal/processors/preprocess_threshold_processor.go
- internal/processors/preprocess_threshold_processor_test.go
- cmd/nms-agent/main.go
- configs/thresholds.yml
- docs/CONFIG_SCHEMA.md
- docs/DATA_CONTRACT.md
- docs/FLOW.md
- docs/KNOWLEDGE.md
- docs/DEVELOPMENT_STAGES.md
Validation:
- cmd /c make fmt
- cmd /c make test
- cmd /c make build
- cmd /c make check
Status update:
- Phase 7: threshold loader/evaluator/tests -> DONE
Notes:
- Threshold results are tagged as `threshold.status`, `threshold.matched`, and `threshold.rule`.

2026-05-26 12:40
Task: Phase 7 interface throughput calculation
Changed files:
- internal/processors/preprocess_threshold_processor.go
- internal/processors/preprocess_threshold_processor_test.go
- docs/DATA_CONTRACT.md
- docs/KNOWLEDGE.md
- docs/DEVELOPMENT_STAGES.md
Validation:
- cmd /c make fmt
- cmd /c make test
- cmd /c make build
- cmd /c make check
Status update:
- Phase 7: throughput/utilization derived metrics -> DONE
Notes:
- Derived metrics emitted only after two samples and valid delta.

2026-05-27 09:20
Task: Rename example device to MikroTik RouterOS
Changed files:
- configs/devices.d/mikrotik-routeros.yml
- docs/CONFIG_SCHEMA.md
- docs/KNOWLEDGE.md
- docs/DEVELOPMENT_STAGES.md
Validation:
- Not run (rename/documentation update only)
Status update:
- Phase 2/Phase 5 device sample rename -> DONE
Notes:
- Updated sample vendor/model to mikrotik/routeros for clarity.

2026-05-27 09:35
Task: Add MikroTik RouterOS SNMP profile (same as standard)
Changed files:
- profiles/mikrotik-routeros.yml
- docs/KNOWLEDGE.md
- docs/DEVELOPMENT_STAGES.md
Validation:
- Not run (profile file + docs only)
Status update:
- Phase 6: vendor profile extension -> DONE
Notes:
- MikroTik RouterOS profile currently mirrors standard metrics (uptime + ifTable).

2026-05-27 10:05
Task: Extend MikroTik RouterOS profile with CPU/memory
Changed files:
- profiles/mikrotik-routeros.yml
- docs/DATA_CONTRACT.md
- docs/KNOWLEDGE.md
- docs/DEVELOPMENT_STAGES.md
Validation:
- Not run (profile + docs update)
Status update:
- Phase 6: MikroTik profile resource metrics -> DONE
Notes:
- Added HOST-RESOURCES CPU load and memory size metrics.

2026-05-27 11:10
Task: Add value_type + string metric support in telemetry
Changed files:
- internal/models/telemetry.go
- internal/collectors/snmp_collector.go
- internal/collectors/icmp_collector.go
- internal/collectors/dummy_collector.go
- internal/processors/passthrough_processor.go
- internal/processors/preprocess_threshold_processor.go
- internal/adapters/terminal_adapter.go
- internal/queue/sqlite_queue.go
- internal/collectors/snmp_collector_test.go
- internal/collectors/icmp_collector_test.go
- internal/processors/preprocess_threshold_processor_test.go
- internal/core/pipeline_sqlite_test.go
- internal/queue/sqlite_queue_test.go
- cmd/nms-agentctl/queue_retry_test.go
- docs/DATA_CONTRACT.md
- docs/KNOWLEDGE.md
- docs/DEVELOPMENT_STAGES.md
Validation:
- cmd /c make fmt
- cmd /c make test
- cmd /c make build
- cmd /c make check
Status update:
- Phase 7: telemetry value_type support -> DONE
Notes:
- value_type is required; number-only thresholds skip string metrics.

2026-05-27 11:20
Task: Add MikroTik string metrics (sysDescr/sysName/ifName)
Changed files:
- profiles/mikrotik-routeros.yml
- docs/DATA_CONTRACT.md
- docs/KNOWLEDGE.md
- docs/DEVELOPMENT_STAGES.md
Validation:
- Not run (profile + docs update)
Status update:
- Phase 6: MikroTik profile string metrics -> DONE
Notes:
- Added system description/name and interface name metrics.
```
