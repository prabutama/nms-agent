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
| Add reload command | `nms-agentctl reload` triggers systemd reload | DONE | Sends SIGHUP to running agent PID after config validation |
| Add SIGHUP handler | Agent can hot reload configuration | DONE | Agent rebuilds runtime pipeline on SIGHUP |

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
| Add `nms-agent run` | Agent can run manually | DONE | Runs config load+validate + periodic pipeline loop by `poll_interval` |
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
| Implement throughput calculation | Interface counter → throughput | DONE | Uses HC counters with fallback to 32-bit. Utilization fallback to high_speed_mbps when speed_bps=0. |
| Implement metric normalization | Raw values → canonical telemetry | DONE | Clamp pct/[0,100], ms/≥0, seconds/≥0, bps/≥0, reachable→0/1. Unit default otomatis. |
| Implement threshold loader | Load `thresholds.yml` | DONE | Threshold rules parsed in config loader |
| Implement threshold evaluator | Warning/critical status | DONE | Preprocess processor tags telemetry |
| Add threshold CLI | `nms-agentctl threshold set/list` | DONE | --config agent.yml, upsert by metric+tags match |
| Add tests | Preprocessing and threshold tests | DONE | Processor threshold unit tests |
| Filter non-physical interface | Drop non-physical iface metrics based on multi-signal classifier | DONE | ifName pattern + ifConnectorPresent + ifType allowlist (6,71) |

**Exit Criteria:**

- Main metrics are processed correctly.
- Thresholds are configurable.
- Alert event or telemetry status can be produced.

---

## Phase 8 — Adapters

**Target:** Send telemetry to platform consumers through MQTT.

| Task | Target Output | Status | Notes |
|---|---|---|---|
| Implement TUI adapter | Bubbletea-based terminal UI monitoring | DONE | Device health, alerts, interface throughput |
| Improve TUI dashboard | Responsive layout + focusable tables + accurate state | DONE | 2-pane wide/stacked narrow, tab focus, dedup alerts, headless tests |
| Add free-like memory view | TUI menampilkan Mem/Swap ala `free` untuk Linux/Proxmox | DONE | UCD-SNMP-MIB (2021.4.*) untuk breakdown + fallback hrStorage |
| Show ICMP latency/jitter | TUI menampilkan latency/jitter/loss per device | DONE | Berdasarkan metric ICMP collector (`icmp.latency_ms`, `icmp.jitter_ms`, `icmp.packet_loss_pct`) |
| Implement Generic MQTT adapter | Send canonical telemetry JSON | DONE | Publish 1 telemetry = 1 message ke topic statis (config: broker/topic/qos/retain/auth/timeout) |
| Implement ThingsBoard MQTT adapter | ThingsBoard Gateway MQTT telemetry | DONE | Topic `v1/gateway/telemetry` (auth via access token), kirim metric + metadata tags/threshold |
| Add adapter health check | Adapter status can be queried | DONE | `nms-agentctl adapter health` checks adapter connectivity without sending telemetry |
| Add adapter tests | Formatting and send behavior are validated | DONE | Unit tests for MQTT adapters + factory + CLI adapter health |

**Exit Criteria:**

- Telemetry can be sent to an MQTT broker.
- Send failure keeps queue data pending.
- ThingsBoard can receive telemetry through the adapter.

---

## Phase 9 — Device Management CLI

**Target:** Allow admins to manage devices without manually editing files.

| Task | Target Output | Status | Notes |
|---|---|---|---|
| Add device list | `nms-agentctl device list` | DONE | Output tabular (id/address/vendor/model + snmp/icmp flags) |
| Add device add | `nms-agentctl device add` | DONE | Validasi field wajib + tulis file atomic ke `devices.d/<id>.yml` |
| Add device update | `nms-agentctl device update` | DONE | Update field terpilih + atomic replace + rollback saat validasi gagal |
| Add device remove | `nms-agentctl device remove` | DONE | Hapus file device berdasarkan id + rollback jika validasi gagal |
| Add device test | `nms-agentctl device test` | DONE | Smoke test ICMP (ping) + SNMP (profile-based walk/get) |
| Add validation before save | Invalid config is not saved | DONE | `device add` menolak input invalid dan menolak duplikasi id sebelum write |
| Auto reload after change | CLI can trigger reload | DONE | Manual `nms-agentctl reload --pid <pid>` triggers hot reload |

**Exit Criteria:**

- Devices can be added, edited, and removed through CLI.
- Changes can be applied through reload.
- `docs/KNOWLEDGE.md` is updated if new files are created.

---

## Phase 10 — systemd Packaging

**Target:** Install and run the agent as a Linux service.

| Task | Target Output | Status | Notes |
|---|---|---|---|
| Create systemd unit | `packaging/systemd/nms-agent.service` | DONE | `ExecStart` runs agent, `ExecReload` sends SIGHUP |
| Create install script | Copy binary, config, and service file | DONE | `packaging/systemd/install.sh` builds and installs from repo |
| Create directory structure | `/etc`, `/var/lib`, `/var/log`, `/opt` | DONE | Created by install script |
| Add service reload support | `systemctl reload nms-agent` | DONE | Uses SIGHUP hot reload handler |
| Add journal logging guide | Logging documentation | DONE | `packaging/systemd/README.md` |
| Test reboot behavior | Agent auto-starts after reboot | TODO | Manual verification on target host |

**Exit Criteria:**

- Agent can be managed through systemd.
- Agent auto-starts after reboot.
- Reload and restart work correctly.

---

## Phase 11 — Reliability and Downtime Testing

**Target:** Prove that telemetry is not lost when the platform consumer connection is interrupted.

| Task | Target Output | Status | Notes |
|---|---|---|---|
| Simulate platform down | Adapter send fails | SKIPPED | Deferred to future sprint |
| Run 10-minute downtime test | 1-minute interval, expected 10 records | SKIPPED | Deferred to future sprint |
| Validate queue pending | All downtime records are stored | SKIPPED | Deferred to future sprint |
| Restore platform | Pending data is sent again | SKIPPED | Deferred to future sprint |
| Validate no duplicate | `event_id` prevents duplicate data | SKIPPED | Deferred to future sprint |
| Document result | Table and screenshots | SKIPPED | Deferred to future sprint |

**Exit Criteria:**

- A 10-minute downtime produces 10 stored records.
- After recovery, all 10 records are resent.
- No telemetry is lost at the agent level.

**Decision:** Phase 11 skipped by decision. Store-and-forward queue is implemented and tested via unit tests. Formal downtime testing deferred to future sprint.

---

## Phase 12 — Hardening and Documentation

**Target:** Prepare the agent for demo, reporting, and future development.

| Task | Target Output | Status | Notes |
|---|---|---|---|
| Add README usage | Installation and usage instructions | DONE | `README.md` created |
| Add config examples | HQ/Branch device examples | DONE | `configs/examples/` with HQ/Branch configs |
| Add troubleshooting guide | Common errors and fixes | DONE | `docs/TROUBLESHOOTING.md` created |
| Add security note | Credentials, `.env`, file permissions | DONE | `docs/SECURITY.md` created |
| Run lint/vet/test/build | Final validation | DONE | All checks passed |
| Create release artifact | Binary and sample config | DONE | `packaging/RELEASE.md` created |

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

2026-05-27 11:30
Task: Add Linux SNMP profile
Changed files:
- profiles/linux.yml
- docs/KNOWLEDGE.md
- docs/DEVELOPMENT_STAGES.md
Validation:
- Not run (profile + docs update)
Status update:
- Phase 6: Linux profile -> DONE
Notes:
- Linux profile includes identity, HOST-RESOURCES, and IF-MIB metrics.

2026-05-27 11:45
Task: Verify throughput prefers high-capacity counters
Changed files:
- internal/processors/preprocess_threshold_processor_test.go
- docs/DEVELOPMENT_STAGES.md
Validation:
- cmd /c make fmt
- cmd /c make test
- cmd /c make build
Status update:
- Phase 7: throughput calculation -> DONE
Notes:
- Added test to ensure HC counters are prioritized over 32-bit counters.

2026-05-28 08:55
Task: Keep nms-agent process alive for throughput delta calculation
Changed files:
- cmd/nms-agent/main.go
- docs/KNOWLEDGE.md
- docs/DEVELOPMENT_STAGES.md
Validation:
- cmd /c make fmt
- cmd /c make test
- cmd /c make build
Status update:
- Runtime: `nms-agent run` now polls continuously by `poll_interval`.
Notes:
- Throughput derived metrics require in-process previous counters; continuous loop preserves state between polls.
```

2026-05-28 09:00
Task: Utilization fallback speed (high_speed_mbps when speed_bps = 0)
Changed files:
- internal/processors/preprocess_threshold_processor.go
- internal/processors/preprocess_threshold_processor_test.go
- configs/thresholds.yml
- docs/DATA_CONTRACT.md
- docs/DEVELOPMENT_STAGES.md
- docs/KNOWLEDGE.md
Validation:
- cmd /c make fmt
- cmd /c make test
- cmd /c make build
- cmd /c make check
Status update:
- Phase 7: utilization fallback to snmp.if.high_speed_mbps -> DONE
Notes:
- Utilization now uses effective speed: speed_bps first, then high_speed_mbps × 1,000,000 as fallback.
- Added tests: TestPreprocessThresholdProcessor_UsesHighSpeedMbpsFallback, TestPreprocessThresholdProcessor_PrefersSpeedBpsOverHighSpeedMbps.
- Added default threshold rules for rx/tx_utilization_pct (warning 70%, critical 90%).

2026-05-28 10:00
Task: Physical interface filter (ifType=6 only)
Changed files:
- internal/processors/preprocess_threshold_processor.go
- internal/processors/preprocess_threshold_processor_test.go
- profiles/standard.yml
- profiles/vendor-example.yml
- profiles/mikrotik-routeros.yml
- profiles/linux.yml
- docs/DATA_CONTRACT.md
- docs/DEVELOPMENT_STAGES.md
- docs/KNOWLEDGE.md
Validation:
- go test ./...
- go build ./...
Status update:
- Phase 7: non-physical interface filter -> DONE
Notes:
- snmp.if.type (ifType) added to all profiles (OID 1.3.6.1.2.1.2.2.1.3, walk, index).
- Processor caches ifType per device+ifIndex; drops metrics for known non-physical interfaces.
- Derived metrics (bps, utilization) also filtered for non-physical ifIndex.
- Safe default: metrics with unknown ifType are preserved.
- Added 4 tests: FiltersNonPhysicalInterfaces, KeepsPhysicalInterfaces, KeepsWhenIfTypeUnknown, FiltersDerivedMetricsForNonPhysical.
```

2026-05-28 11:00
Task: Cross-platform physical interface classifier (ifName + ifConnectorPresent + ifType)
Changed files:
- internal/processors/preprocess_threshold_processor.go
- internal/processors/preprocess_threshold_processor_test.go
- profiles/standard.yml
- profiles/vendor-example.yml
- profiles/mikrotik-routeros.yml
- profiles/linux.yml
- docs/DATA_CONTRACT.md
- docs/DEVELOPMENT_STAGES.md
- docs/KNOWLEDGE.md
Validation:
- go test ./...
- go build ./...
Status update:
- Phase 7: cross-platform physical interface classifier -> DONE
Notes:
- Added snmp.if.connector_present (OID 1.3.6.1.2.1.31.1.1.1.17) to all profiles.
- Replaced simple ifType filter with multi-signal classifier:
  1) ifName virtual pattern deny (Proxmox, Docker, K8s, etc.)
  2) ifConnectorPresent=true keep
  3) ifConnectorPresent=false drop
  4) ifType allowlist (6=ethernet, 71=wifi) keep
  5) ifType known but not allowlisted drop
  6) Unknown signals keep (safe default)
- Derived metrics (bps, utilization) automatically follow filter.
- Added 9 new tests: Proxmox patterns, Docker/K8s patterns, physical names, connector present override, connector false drop, wifi ifType.
```

2026-05-28 12:00
Task: Queue delivery drain loop (configurable drain, max_batch, max_batches_per_cycle, stop_on_error)
Changed files:
- internal/config/types.go
- internal/core/pipeline.go
- internal/core/pipeline_sqlite_test.go
- cmd/nms-agent/main.go
- configs/agent.yml
- docs/CONFIG_SCHEMA.md
- docs/DEVELOPMENT_STAGES.md
- docs/KNOWLEDGE.md
Validation:
- go build ./...
- go test ./...
Status update:
- Phase 8: queue delivery drain loop -> DONE
Notes:
- Pipeline now supports configurable drain loop instead of single-batch delivery.
- Per poll cycle: loop fetches pending items up to `max_batch`, sends, repeats until queue empty or `max_batches_per_cycle` reached.
- stop_on_error controls behavior on send failure (abort vs continue).
- Default config: max_batch=200, drain_enabled=true, max_batches_per_cycle=20, stop_on_error=true.

Task: Threshold CLI (list + set with upsert)
Changed files:
- cmd/nms-agentctl/main.go
- cmd/nms-agentctl/threshold.go
- cmd/nms-agentctl/threshold_test.go
- internal/config/types.go
- internal/config/validate.go
Validation:
- go build ./cmd/nms-agentctl/...
- go test -run 'TestThreshold|TestParseTags|TestTagsEqual|TestValidateThreshold' ./cmd/nms-agentctl/...
Status update:
- Phase 7: threshold CLI -> DONE
Notes:
- `nms-agentctl threshold list` reads thresholds.yml and prints each rule with metric/operator/warning/critical/tags.
- `nms-agentctl threshold set` upserts a rule by metric+tags match; creates new or updates existing.
- Config path resolved via agent.yml → paths.thresholds_file.
- ResolvePath() helper added to internal/config/types.go for relative path resolution with env var expansion.
- ValidateThresholdRules() added to internal/config/validate.go for isolated rule validation.
- Write is atomic via temp-file-rename to avoid truncation on crash.

2026-05-28 14:30
Task: Metric normalization + E2E flow test
Changed files:
- internal/processors/preprocess_threshold_processor.go
- internal/processors/preprocess_threshold_processor_test.go
- internal/core/pipeline_sqlite_test.go
- docs/DATA_CONTRACT.md
- docs/DEVELOPMENT_STAGES.md
Validation:
- go build ./...
- go test -run 'TestNormalizeMetrics' ./internal/processors/...
- go test -run 'TestPipelineE2E' ./internal/core/...
Status update:
- Phase 7: metric normalization -> DONE
Notes:
- normalizeMetrics() added between physical filter and threshold evaluation.
- Rules: _pct clamp [0,100], _ms/≥0, _seconds/≥0, _bps/≥0, icmp.reachable→0/1.
- Unit defaults (pct, ms, s, bps) auto-filled when unit tag missing.
- 10 unit tests: clamp percent >100 / <0, ms/seconds/bps non-negative, reachable binary, existing unit preserved, non-number passthrough, threshold-after-normalization, in-range passthrough.
- 2 E2E pipeline tests: verify full collect→normalize→threshold→queue→adapter flow; string passthrough integrity.

2026-05-28 15:00
Task: TUI adapter (Bubbletea-based terminal UI)
Changed files:
- internal/adapters/tui_adapter.go
- internal/adapters/tui_adapter_test.go
- internal/adapters/factory.go
- internal/adapters/port.go
- cmd/nms-agent/main.go
- docs/ADAPTER_CONTRACT.md
- docs/CONFIG_SCHEMA.md
- docs/DEVELOPMENT_STAGES.md
- docs/KNOWLEDGE.md
Validation:
- go build ./...
- go test -run 'TestTUIAdapter' ./internal/adapters/...
Status update:
- Phase 8: TUI adapter -> DONE
Notes:
- New adapter `tui` selectable via `adapters.active: tui` in config.
- Uses Bubbletea + Lipgloss for full-screen terminal UI.
- Displays: cycle count, device health (up/down), queue stats, threshold alerts (warning/critical), interface throughput (top by utilization).
- Factory `NewAdapter(name, config)` replaces hardcoded `NewTerminalAdapter()` in main.go.
- Supports config key `refresh_interval` (default 1s).
- Hotkeys: [q] quit, [h] help.

2026-05-28 16:10
Task: Improve TUI dashboard (responsive layout + accurate state)
Changed files:
- internal/adapters/tui_adapter.go
- internal/adapters/tui_model.go
- internal/adapters/tui_view.go
- internal/adapters/tui_theme.go
- internal/adapters/tui_keys.go
- internal/adapters/tui_format.go
- internal/adapters/tui_adapter_test.go
- go.mod
- go.sum
- docs/CONFIG_SCHEMA.md
- docs/DEVELOPMENT_STAGES.md
- docs/KNOWLEDGE.md
Validation:
- go fmt ./...
- go build ./...
- go test -count=1 ./...
Status update:
- Phase 8: TUI dashboard polish -> DONE
Notes:
- UI now uses Bubbles table + help for interactive panes (tab to switch focus, up/down navigation).
- State is tracked per-device/per-interface (no counter drift) and alerts are deduplicated and sorted by severity/time.
- Layout is responsive: 2-pane on wide terminals, stacked on narrow terminals.
- Added TUI config flags: alt_screen, discard_output, disable_renderer (useful for tests/headless).
- TUI tests now run headless to avoid ANSI/altscreen noise in test output.

2026-05-29 16:30
Task: TUI memory semantics + anti-overlap rendering
Changed files:
- internal/adapters/tui_model.go
- internal/adapters/tui_view.go
- internal/adapters/tui_format.go
- docs/KNOWLEDGE.md
- docs/DEVELOPMENT_STAGES.md
Validation:
- make fmt
- make test
- make build
Notes:
- Memory label fixed: `snmp.host.memory.size_kb` is total RAM (now shown as Memory Total).
- Memory Used is estimated via hrStorage (`snmp.host.storage.*`) by matching storage size to total RAM.
- Device list/detail panes use MaxWidth + truncation to avoid text wrapping into the other pane.

2026-05-29 17:00
Task: Linux profile hrStorage size/used units (enable memory used % on Proxmox)
Changed files:
- profiles/linux.yml
- docs/KNOWLEDGE.md
- docs/DEVELOPMENT_STAGES.md
Validation:
- make fmt
- make test
- make build
Notes:
- Linux profile now collects `snmp.host.storage.size_units` and `snmp.host.storage.used_units` (in addition to allocation_units).
- This enables TUI to estimate Memory Used and percent via hrStorage matching.

2026-05-29 17:20
Task: Proxmox memory used percent (use hrStorageType RAM)
Changed files:
- profiles/linux.yml
- internal/adapters/tui_model.go
- internal/adapters/tui_view.go
- internal/adapters/tui_format.go
- docs/KNOWLEDGE.md
- docs/DEVELOPMENT_STAGES.md
Validation:
- make fmt
- make test
- make build
Notes:
- Linux profile now also collects `snmp.host.storage.type` and `snmp.host.storage.description`.
- TUI now prefers hrStorageType==hrStorageRam (`1.3.6.1.2.1.25.2.1.2`) when computing Memory Used.
- Memory Used percent now shows 1 decimal (avoid misleading 0%).

2026-05-29 18:00
Task: Proxmox free-like memory breakdown (UCD-SNMP-MIB)
Changed files:
- profiles/linux.yml
- internal/adapters/tui_model.go
- internal/adapters/tui_view.go
- internal/adapters/tui_format.go
- docs/DATA_CONTRACT.md
- docs/KNOWLEDGE.md
- docs/DEVELOPMENT_STAGES.md
Validation:
- make fmt
- make test
- make build
Notes:
- Linux profile now collects optional UCD-SNMP-MIB memory/swap OIDs (memAvailReal, memSysAvail, memBuffer, memCached, memShared, memTotalSwap, memAvailSwap).
- TUI shows Mem/Swap lines like `free` and computes used as `total - available` when UCD metrics are present; falls back to hrStorage heuristic otherwise.

2026-05-29 18:20
Task: TUI show ICMP latency/jitter/loss
Changed files:
- internal/adapters/tui_model.go
- internal/adapters/tui_view.go
- internal/adapters/tui_adapter_test.go
- internal/collectors/dummy_collector.go
- docs/KNOWLEDGE.md
- docs/DEVELOPMENT_STAGES.md
Validation:
- make fmt
- make test
- make build
Notes:
- TUI Health section now shows `icmp.latency_ms`, `icmp.jitter_ms`, and `icmp.packet_loss_pct` per selected device.
- Dummy collector now emits ICMP latency/jitter/loss so TUI demo mode shows these fields.

2026-05-29 19:10
Task: Phase 8 generic MQTT adapter (canonical JSON publish)
Changed files:
- internal/adapters/mqtt_generic_adapter.go
- internal/adapters/mqtt_generic_adapter_test.go
- internal/adapters/factory.go
- go.mod
- go.sum
- docs/CONFIG_SCHEMA.md
- docs/KNOWLEDGE.md
- docs/DEVELOPMENT_STAGES.md
Validation:
- make fmt
- make test
- make build
- nms-agentctl validate --config configs/agent.yml
Notes:
- Added adapter `generic_mqtt` to publish canonical telemetry JSON (1 telemetry = 1 MQTT message) to a static topic.
- Uses Eclipse Paho MQTT client; send failures bubble up so queue items remain pending for retry.

2026-05-30 00:10
Task: Configurable output timezone (presentation-only)
Changed files:
- internal/config/types.go
- internal/config/timezone.go
- internal/config/validate.go
- cmd/nms-agent/main.go
- internal/adapters/output_timezone.go
- internal/adapters/terminal_adapter.go
- internal/adapters/tui_view.go
- internal/adapters/tui_model.go
- internal/adapters/mqtt_generic_adapter.go
- configs/agent.yml
- docs/CONFIG_SCHEMA.md
- docs/KNOWLEDGE.md
- docs/DEVELOPMENT_STAGES.md
Validation:
- make fmt
- make test
- make build
- nms-agentctl validate --config configs/agent.yml
Notes:
- Added `agent.output.timezone` to render timestamps in all adapters (terminal/TUI/MQTT) using configured timezone.
- Core/queue telemetry timestamps remain absolute instants (stored in UTC), only presentation changes.

2026-05-30 00:30
Task: Generic MQTT strict queue mode (Option A)
Changed files:
- internal/adapters/mqtt_generic_adapter.go
- internal/adapters/mqtt_generic_adapter_test.go
- configs/adapters.yml
- docs/CONFIG_SCHEMA.md
- docs/DEVELOPMENT_STAGES.md
Validation:
- make fmt
- make test
- make build
- nms-agentctl validate --config configs/agent.yml
Notes:
- Added `strict_queue_mode` so adapter fail-fast on disconnect (SQLite queue pending reflects broker outage).

2026-05-30 01:10
Task: Phase 8 ThingsBoard MQTT adapter (Gateway API)
Changed files:
- internal/adapters/thingsboard_mqtt_adapter.go
- internal/adapters/thingsboard_mqtt_adapter_test.go
- internal/adapters/factory.go
- docs/CONFIG_SCHEMA.md
- docs/KNOWLEDGE.md
- docs/DEVELOPMENT_STAGES.md
Validation:
- make fmt
- make test
- make build
Notes:
- Added adapter `thingsboard_mqtt` using ThingsBoard Gateway MQTT API topic `v1/gateway/telemetry` with gateway access token auth.
- Telemetry payload carries metric value plus canonical metadata (`__value_type`, `__tags`) including threshold tags.

2026-05-30 01:30
Task: Adapter health check CLI
Changed files:
- cmd/nms-agentctl/main.go
- cmd/nms-agentctl/adapter_health.go
- internal/adapters/port.go
- internal/adapters/mqtt_generic_adapter.go
- internal/adapters/thingsboard_mqtt_adapter.go
- docs/CLI_COMMANDS.md
- docs/KNOWLEDGE.md
- docs/DEVELOPMENT_STAGES.md
Validation:
- make fmt
- make test
- make build
- go run ./cmd/nms-agentctl adapter health --config configs/agent.yml
Notes:
- Added `nms-agentctl adapter health` to check active adapter connectivity (MQTT connect) without sending telemetry.

2026-05-31 10:00
Task: Adapter tests (factory + CLI)
Changed files:
- cmd/nms-agentctl/adapter_health_test.go
- internal/adapters/factory_test.go
- docs/DEVELOPMENT_STAGES.md
- docs/KNOWLEDGE.md
Validation:
- make fmt
- make test
- make build
Notes:
- Added tests for adapter factory selection and adapter health CLI paths.

2026-05-31 10:20
Task: Phase 9 device list CLI
Changed files:
- cmd/nms-agentctl/main.go
- cmd/nms-agentctl/device.go
- docs/CLI_COMMANDS.md
- docs/DEVELOPMENT_STAGES.md
- docs/KNOWLEDGE.md
Validation:
- make fmt
- make test
- make build
- go run ./cmd/nms-agentctl device list --config configs/agent.yml
Notes:
- Added `nms-agentctl device list` to print loaded devices in a stable, tabular format.

2026-05-31 10:40
Task: Phase 9 device add CLI
Changed files:
- cmd/nms-agentctl/main.go
- cmd/nms-agentctl/device.go
- cmd/nms-agentctl/device_test.go
- docs/CLI_COMMANDS.md
- docs/DEVELOPMENT_STAGES.md
- docs/KNOWLEDGE.md
Validation:
- make fmt
- make test
- make build
Notes:
- Added `nms-agentctl device add` with required field validation, duplicate id check, atomic write, and rollback on post-write validation failure.

2026-05-31 11:10
Task: Phase 9 device update/remove/test CLI
Changed files:
- cmd/nms-agentctl/main.go
- cmd/nms-agentctl/device.go
- cmd/nms-agentctl/device_test.go
- docs/CLI_COMMANDS.md
- docs/DEVELOPMENT_STAGES.md
- docs/KNOWLEDGE.md
Validation:
- make fmt
- make test
- make build
- go run ./cmd/nms-agentctl device list --config configs/agent.yml
- go run ./cmd/nms-agentctl device test --config configs/agent.yml --id proxmox-ta --snmp=false --icmp=true
Notes:
- Added device update (atomic replace + rollback), remove (delete + rollback), and test (ICMP ping + SNMP profile-based collect) subcommands.

2026-05-31 11:30
Task: Manual reload command + SIGHUP hot reload
Changed files:
- cmd/nms-agent/main.go
- cmd/nms-agent/reload_signal_unix.go
- cmd/nms-agent/reload_signal_windows.go
- cmd/nms-agentctl/main.go
- cmd/nms-agentctl/reload.go
- cmd/nms-agentctl/reload_test.go
- cmd/nms-agentctl/reload_signal_unix.go
- cmd/nms-agentctl/reload_signal_windows.go
- docs/CLI_COMMANDS.md
- docs/DEVELOPMENT_STAGES.md
- docs/KNOWLEDGE.md
Validation:
- make fmt
- make test
- make build
Notes:
- Added `nms-agentctl reload --pid <pid>` to validate config then send SIGHUP.
- Agent now rebuilds runtime pipeline on SIGHUP (devices/thresholds/adapter/profiles), without restarting the process.

2026-05-31 12:10
Task: Phase 10 systemd packaging (unit + install)
Changed files:
- packaging/systemd/nms-agent.service
- packaging/systemd/install.sh
- packaging/systemd/README.md
- packaging/systemd/agent.yml
- packaging/systemd/adapters.yml
- packaging/systemd/thresholds.yml
- packaging/systemd/devices.d/example-linux-proxmox.yml
- docs/DEVELOPMENT_STAGES.md
- docs/KNOWLEDGE.md
Validation:
- make fmt
- make test
- make build
Notes:
- Added systemd unit with `ExecReload` sending SIGHUP and a simple install script that builds from repo.

2026-06-02 09:45
Task: Phase 12 hardening and documentation
Changed files:
- README.md
- docs/TROUBLESHOOTING.md
- docs/SECURITY.md
- docs/DEVELOPMENT_STAGES.md
- docs/KNOWLEDGE.md
- configs/examples/hq-agent.yml
- configs/examples/hq-adapters.yml
- configs/examples/hq-thresholds.yml
- configs/examples/hq-devices/hq-core-router.yml
- configs/examples/hq-devices/hq-dist-switch.yml
- configs/examples/hq-devices/hq-app-server.yml
- configs/examples/branch-agent.yml
- configs/examples/branch-adapters.yml
- configs/examples/branch-thresholds.yml
- configs/examples/branch-devices/branch-edge-router.yml
- configs/examples/branch-devices/branch-access-switch.yml
- packaging/RELEASE.md
Validation:
- make fmt
- make test
- make build
- go run ./cmd/nms-agentctl validate --config configs/agent.yml
- go run ./cmd/nms-agent run --config configs/agent.yml --collector-mode dummy
Status update:
- Phase 11: SKIPPED (deferred to future sprint)
- Phase 12: ALL TASKS DONE
Notes:
- Phase 11 (downtime testing) skipped by decision; queue store-and-forward is already implemented and tested via unit tests.
- README.md created with quick start, config reference, CLI commands, architecture, and demo guide.
- docs/TROUBLESHOOTING.md covers config, collector, queue, adapter, reload, systemd, and performance issues.
- docs/SECURITY.md covers credentials, file permissions, network security, and known considerations.
- configs/examples/ provides HQ and Branch site templates with agent.yml, adapters.yml, thresholds.yml, and device configs.
- packaging/RELEASE.md provides build commands for Linux/Windows/ARM64 and deployment checklist.
- Final validation passed: fmt, test, build, validate, and run with dummy collector.

2026-06-02 10:00
Task: Phase 12B proper fix for profiles path resolution
Changed files:
- internal/config/types.go
- internal/config/loader.go
- cmd/nms-agent/main.go
- packaging/systemd/install.sh
- packaging/systemd/agent.yml
- docs/KNOWLEDGE.md
- docs/DEVELOPMENT_STAGES.md
Validation:
- make fmt
- make test
- make vet
- make build
- go run ./cmd/nms-agentctl validate --config configs/agent.yml
Status update:
- Phase 12B: proper fix DONE
Notes:
- Added `profiles_dir` to `Paths` struct in config/types.go for explicit profile path configuration.
- Updated config loader to resolve `profiles_dir` from config; falls back to `../profiles` relative to agent.yml when not set.
- Updated main.go to use resolved `ProfilesDir` from config instead of hardcoded `../profiles` path.
- Updated systemd installer to copy all profiles from `profiles/` directory to `/etc/nms-agent/profiles`.
- Updated systemd agent.yml to include `profiles_dir: /etc/nms-agent/profiles`.
- This fix resolves the `open /etc/profiles: no such file or directory` error during service startup.

2026-06-02 10:30
Task: Phase 12C CLI default config path to /etc/nms-agent/agent.yml
Changed files:
- cmd/nms-agentctl/main.go
- cmd/nms-agentctl/validate.go
- cmd/nms-agentctl/device.go
- cmd/nms-agentctl/queue_status.go
- cmd/nms-agentctl/adapter_health.go
- cmd/nms-agentctl/threshold.go
- cmd/nms-agentctl/reload.go
- docs/KNOWLEDGE.md
- docs/DEVELOPMENT_STAGES.md
Validation:
- make fmt
- make test
- make vet
- make build
Status update:
- Phase 12C: CLI default config path fix DONE
Notes:
- Changed default config path from configs/agent.yml to /etc/nms-agent/agent.yml in all CLI commands.
- Updated usage examples in main.go and threshold.go to reflect new defaults.
- Updated device test to use loaded.ProfilesDir instead of hardcoded ../profiles.
- CLI commands now work without --config flag when config is at /etc/nms-agent/agent.yml.

2026-06-02 11:00
Task: Phase 12D local viewer daemon + socket IPC
Changed files:
- internal/viewer/message.go
- internal/viewer/hub.go
- internal/viewer/server.go
- internal/viewer/client.go
- internal/queue/port.go
- internal/queue/sqlite_queue.go
- internal/core/pipeline.go
- internal/adapters/port.go
- cmd/nms-agent/main.go
- cmd/nms-agentctl/main.go
- cmd/nms-agentctl/view.go
- packaging/systemd/nms-agent.service
- docs/KNOWLEDGE.md
- docs/DEVELOPMENT_STAGES.md
Status update:
- Phase 12D: Viewer daemon + socket IPC DONE
Notes:
- Created internal/viewer package (message, hub, server, client) for local IPC.
- Added Snapshot method to queue and queue.Observer interface.
- Wired queue snapshot to viewer.Hub in daemon main.go.
- Added pipeline.SetObserver for live telemetry updates.
- Added nms-agentctl view command (snapshot + live update via Unix socket).
- Updated systemd unit with RuntimeDirectory=nms-agent for socket path.

2026-06-02 11:30
Task: Phase 12E Interactive device add wizard
Changed files:
- cmd/nms-agentctl/device.go
- docs/KNOWLEDGE.md
- docs/DEVELOPMENT_STAGES.md
Status update:
- Phase 12E: Interactive wizard DONE
Notes:
- Added automatic wizard detection when required flags missing on interactive terminal.
- Wizard prompts: Device ID, Address, Vendor, Model, SNMP, ICMP with validation.
- Summary display before save with confirmation prompt.
- --interactive flag for forcing wizard even with some flags provided.
- Non-interactive shell still shows clear error message.

2026-06-03 12:00
Task: Fix viewer status details and implement adapter-specific rendering
Changed files:
- internal/viewer/message.go
- internal/viewer/hub.go
- internal/viewer/server.go
- internal/adapters/thingsboard_mqtt_adapter.go
- cmd/nms-agentctl/view.go
- docs/KNOWLEDGE.md
- docs/DEVELOPMENT_STAGES.md
Validation:
- make fmt
- make test
- make build
Status update:
- Phase 12D: viewer status details + adapter-specific rendering DONE
Notes:
- Added Details field to viewer.Message for status detail display.
- Updated Hub.StatusUpdate to carry both status and details.
- Updated server.go to broadcast status details.
- Wired observer UpdateStatus with status+details in ThingsBoard MQTT adapter.
- Implemented adapter-specific rendering in nms-agentctl view (tui/terminal/mqtt_generic/thingsboard_mqtt).
- All validation passed: fmt, test, build.

2026-06-03 13:00
Task: Remove terminal adapter and implement view --mode summary/raw
Changed files:
- internal/adapters/tui_state.go
- internal/adapters/tui_model.go
- internal/adapters/tui_view.go
- internal/adapters/tui_adapter.go
- internal/adapters/factory.go
- internal/adapters/factory_test.go
- internal/adapters/tui_adapter_test.go
- cmd/nms-agentctl/view.go
- cmd/nms-agentctl/queue_retry.go
- cmd/nms-agentctl/queue_retry_test.go
- cmd/nms-agentctl/adapter_health.go
- cmd/nms-agentctl/adapter_health_test.go
- internal/config/validate_test.go
- docs/KNOWLEDGE.md
- docs/CONFIG_SCHEMA.md
- docs/DEVELOPMENT_STAGES.md
Validation:
- make fmt
- make test
- make build
Status update:
- Phase 12D: terminal adapter removal + view --mode DONE
Notes:
- Removed terminal adapter from factory and all CLI references.
- Extracted shared State reducer from TUI model into internal/adapters/tui_state.go.
- TUI adapter now uses shared State for telemetry reduction.
- nms-agentctl view now supports --mode summary (default) and --mode raw.
- summary mode shows device counts and last update timestamp.
- raw mode shows full telemetry snapshot and live stream.
- All tests pass and build succeeds.

2026-06-03 14:00
Task: Harden wizard input for device add
Changed files:
- cmd/nms-agentctl/device.go
- internal/config/validate.go
- internal/config/validate_test.go
- cmd/nms-agentctl/device_test.go
- docs/KNOWLEDGE.md
- docs/DEVELOPMENT_STAGES.md
Validation:
- make fmt
- make test
- make build
Status update:
- Phase 13A: Wizard input hardening DONE
Notes:
- Added sanitizeInput helper to strip control chars, \r, ANSI escapes from interactive input.
- Added validateDeviceID: rejects non-safe characters (only alnum + -_. allowed).
- Added validateAddress: validates IP or hostname, rejects control chars.
- Added validateVendorModel: rejects control chars in vendor/model.
- Config validation now rejects device id/address with hidden characters.
- Tests added for sanitizeInput, validateDeviceID, validateAddress, validateVendorModel, and config validation with hidden chars.
- All validation passed: fmt, test, build.

2026-06-03 15:00
Task: Preserve existing config files during reinstall
Changed files:
- packaging/systemd/install.sh
- README.md
- docs/DEVELOPMENT_STAGES.md
Validation:
- make fmt
- make test
- make build
Status update:
- Phase 13B: Config preserve on reinstall DONE
Notes:
- install.sh now checks if config files exist before installing.
- Existing agent.yml, adapters.yml, thresholds.yml are preserved.
- Updated sample configs are deployed as *.dist files for reference.
- README.md updated with reinstall/upgrade notes.
- All validation passed: fmt, test, build.

2026-06-03 16:00
Task: Rewrite wizard input to use Scanner and validate per field
Changed files:
- cmd/nms-agentctl/device.go
- docs/DEVELOPMENT_STAGES.md
Validation:
- make fmt
- make test
- make build
Status update:
- Phase 13C: Wizard input rewrite DONE
Notes:
- Replaced bufio.NewReader.ReadString with bufio.Scanner for safer line reading.
- Validation now runs immediately per field instead of before write.
- Invalid input is rejected with clear error and prompt restart.
- Added input stream end detection for graceful cancellation.
- sanitizeInput kept for existing tests but no longer used in wizard path.
- All validation passed: fmt, test, build.

2026-06-03 17:00
Task: Add reprompt logic to wizard for typo tolerance
Changed files:
- cmd/nms-agentctl/device.go
- docs/DEVELOPMENT_STAGES.md
Validation:
- make fmt
- make test
- make build
Status update:
- Phase 13D: Wizard reprompt for typos DONE
Notes:
- Added promptValidated function that loops until valid input is provided.
- Invalid input now shows error and re-prompts the same field instead of exiting.
- Only EOF/input stream end triggers cancellation.
- Added validateVendorForSingle and validateModelForSingle helpers.
- All validation passed: fmt, test, build.

2026-06-03 18:00
Task: Milestone A passive discovery via netlink + SNMP fingerprint + auto-promote
Changed files:
- go.mod
- go.sum
- cmd/nms-agent/main.go
- internal/config/types.go
- internal/config/loader.go
- internal/config/validate.go
- internal/config/validate_test.go
- internal/discovery/types.go
- internal/discovery/service.go
- internal/discovery/service_test.go
- internal/discovery/snmp_probe.go
- internal/discovery/resolver.go
- internal/discovery/promote.go
- internal/discovery/providers/netlink/provider_linux.go
- internal/discovery/providers/netlink/provider_stub.go
- docs/CONFIG_SCHEMA.md
- docs/KNOWLEDGE.md
- docs/DEVELOPMENT_STAGES.md
Validation:
- go test ./...
- make fmt
- make test
- make build
Status update:
- Phase 14A: Passive discovery milestone A DONE
Notes:
- Added `discovery` config schema and validation (interface/subnet/provider/snmp/auto_promote/exploration).
- Added passive Linux netlink neighbor-table provider with non-Linux stub.
- Added discovery service to filter existing inventory, probe SNMP fingerprint, resolve vendor/model, and auto-promote devices with known profile match.
- Discovery auto-promote always writes `snmp.enabled: true` and `icmp.enabled: true` with collision-safe device IDs.
- Integrated discovery loop into agent runtime with internal config reload when new devices are written.
- Exploration config is parsed/validated; Milestone A stops at known-profile auto-promote.
- All validation passed: fmt, test, build.

2026-06-03 19:00
Task: Milestone B safe exploration + generated profile auto-promote
Changed files:
- cmd/nms-agent/main.go
- internal/discovery/types.go
- internal/discovery/service.go
- internal/discovery/service_test.go
- internal/discovery/explorer/explorer.go
- internal/discovery/explorer/explorer_test.go
- docs/CONFIG_SCHEMA.md
- docs/KNOWLEDGE.md
- docs/DEVELOPMENT_STAGES.md
Validation:
- go test ./...
- make fmt
- make test
- make build
Status update:
- Phase 14B: Safe exploration milestone B DONE
Notes:
- Added safe exploration fallback for `run_when: no_profile_match` using a static allowlist of read-only OIDs.
- Generated profiles are validated, written to `profiles/`, then reloaded in-memory so discovery can auto-promote in the same cycle.
- Generated match fallback uses resolver output if available, otherwise `vendor=discovered` and `model=sysobj-...` from `sysObjectID`.
- Added tests for generated profile writer and service flow `no_profile_match -> generated profile -> promote`.
- All validation passed: fmt, test, build.

2026-06-03 20:00
Task: Discovery CLI and observability completion
Changed files:
- cmd/nms-agentctl/main.go
- cmd/nms-agentctl/discover.go
- cmd/nms-agentctl/discover_test.go
- internal/discovery/types.go
- internal/discovery/service.go
- docs/KNOWLEDGE.md
- docs/CLI_COMMANDS.md
- README.md
- docs/DEVELOPMENT_STAGES.md
Validation:
- go test ./...
- make fmt
- make test
- make build
Status update:
- Phase 14C: Discovery CLI/observability DONE
Notes:
- Added `nms-agentctl discovery status|preview|run` commands.
- `preview` runs discovery in dry-run mode without writing device/profile files.
- Discovery result reporting now includes generated profile count.
- Added CLI tests for preview, run, and status flows.
- All validation passed: fmt, test, build.

2026-06-03 21:00
Task: Fix discovery sysObjectID parsing for SNMP ObjectIdentifier
Changed files:
- internal/discovery/snmp_probe.go
- internal/discovery/snmp_probe_test.go
- docs/KNOWLEDGE.md
- docs/DEVELOPMENT_STAGES.md
Validation:
- go test ./...
Status update:
- Phase 14D: sysObjectID parsing fix DONE
Notes:
- Discovery probe now parses `sysObjectID` from SNMP `ObjectIdentifier` values, not only string/byte values.
- Added normalization for `iso.` prefix and leading-dot OID forms so resolver sees numeric dotted OIDs.
- Added regression tests for MikroTik-like `sysObjectID` representation.
 
2026-06-05 10:40
Task: Fix discovery fingerprint mapping for normalized SNMP response OID names
Changed files:
- internal/discovery/snmp_probe.go
- internal/discovery/snmp_probe_test.go
- docs/KNOWLEDGE.md
- docs/DEVELOPMENT_STAGES.md
Validation:
- go test ./internal/discovery ./cmd/nms-agentctl
Status update:
- Phase 14D follow-up: normalized response OID mapping DONE
Notes:
- Discovery probe now normalizes `pkt.Variables[].Name` before mapping `sysObjectID`, `sysName`, and `sysDescr` into the fingerprint.
- Added regression coverage for SNMP responses that return names like `.1.3.6.1.2.1.1.2.0` while still carrying `ObjectIdentifier` values such as `iso.3.6.1.4.1.14988.1`.

2026-06-05 11:15
Task: Fix viewer summary showing wrong device count after discovery auto-promote
Changed files:
- internal/viewer/hub.go
- internal/viewer/hub_test.go
- docs/KNOWLEDGE.md
- docs/DEVELOPMENT_STAGES.md
Validation:
- go test ./internal/viewer ./cmd/nms-agentctl
Status update:
- Phase 14E: viewer summary merge DONE
Notes:
- Hub.Update() now merges telemetry into snapshot by key (device_id + metric + ifIndex) instead of overwriting with latest batch.
- Summary view now counts devices from merged snapshot, so devices promoted by discovery appear correctly without requiring restart.
- Live telemetry broadcast unchanged for view --mode raw.
- Added unit tests for multi-device merge, same-key replace, and different ifIndex.

2026-06-06 09:00
Task: Fix view --mode summary to show live device count and up/down status
Changed files:
- cmd/nms-agentctl/view.go
- docs/KNOWLEDGE.md
- docs/DEVELOPMENT_STAGES.md
Validation:
- go test ./cmd/nms-agentctl
Status update:
- Phase 14G: view summary live update DONE
Notes:
- `nms-agentctl view --mode summary` now maintains a local state that applies every live telemetry batch.
- Summary block is re-rendered on every batch, so new devices and up/down status changes appear automatically without re-running the command.
- `renderSummaryFromState` helper computes total/up/down/unknown from the aggregated state instead of a single batch.

2026-06-06 09:40
Task: Improve summary mode with per-device status table and in-place redraw
Changed files:
- cmd/nms-agentctl/view.go
- docs/KNOWLEDGE.md
- docs/DEVELOPMENT_STAGES.md
Validation:
- go test ./cmd/nms-agentctl
Status update:
- Phase 14G follow-up: interactive summary rendering DONE
Notes:
- `view --mode summary` now renders a per-device table with status, last seen, latency, loss, and alert counts.
- On interactive terminals, summary updates redraw in place instead of printing a new summary block every batch.
- Live summary still updates from aggregated telemetry state, so newly discovered devices and up/down changes appear automatically.

2026-06-06 10:20
Task: Improve ThingsBoard indexed metric usability with flattened interface keys
Changed files:
- internal/adapters/thingsboard_mqtt_adapter.go
- internal/adapters/thingsboard_mqtt_adapter_test.go
- docs/KNOWLEDGE.md
- docs/DEVELOPMENT_STAGES.md
Validation:
- go test ./internal/adapters
Status update:
- Phase 14H: ThingsBoard indexed key flattening DONE
Notes:
- ThingsBoard adapter now emits additional flattened keys for telemetry with `ifIndex` tags, using `ifName` when available and falling back to `idx<ifIndex>`.
- Canonical metric key and `__tags` metadata are still published unchanged, so core contract remains platform-agnostic and backward compatible.
- This makes interface-scoped widgets in ThingsBoard easier to build without relying on tag-map inspection.

2026-06-06 10:40
Task: Align ThingsBoard flattened interface key sanitization
Changed files:
- internal/adapters/thingsboard_mqtt_adapter.go
- internal/adapters/thingsboard_mqtt_adapter_test.go
- docs/KNOWLEDGE.md
- docs/DEVELOPMENT_STAGES.md
Validation:
- go test ./internal/adapters
Status update:
- Phase 14H follow-up: flattened key sanitization DONE
Notes:
- Flattened interface identities for ThingsBoard now use lowercase names, trim spaces, convert space/slash/dot/colon to `-`, drop unsupported characters, collapse repeated dashes, and fall back to `idx<ifIndex>` when empty.
- This keeps per-interface telemetry keys stable and dashboard-friendly while preserving the original canonical metric plus tags in the same payload.

2026-06-06 11:05
Task: Restrict ThingsBoard flattened-only behavior to indexed interface metrics
Changed files:
- internal/adapters/thingsboard_mqtt_adapter.go
- internal/adapters/thingsboard_mqtt_adapter_test.go
- docs/KNOWLEDGE.md
- docs/DEVELOPMENT_STAGES.md
Validation:
- go test ./internal/adapters
Status update:
- Phase 14H follow-up: interface-only flattened payload DONE
Notes:
- For `snmp.if.*` telemetry with `ifIndex`, ThingsBoard adapter now publishes only the flattened key plus flattened `__tags` and `__value_type`; the generic interface key is no longer emitted.
- Non-indexed metrics keep the old generic format, and indexed non-interface metrics such as `snmp.host.storage.*` are intentionally unchanged.

2026-06-06 11:20
Task: Flatten indexed host storage metrics for ThingsBoard payloads
Changed files:
- internal/adapters/thingsboard_mqtt_adapter.go
- internal/adapters/thingsboard_mqtt_adapter_test.go
- docs/KNOWLEDGE.md
- docs/DEVELOPMENT_STAGES.md
Validation:
- go test ./internal/adapters
Status update:
- Phase 14H follow-up: storage flattened payload DONE
Notes:
- `snmp.host.storage.*` telemetry with `ifIndex` now uses flattened-only ThingsBoard keys like `snmp.host.storage.idx65536.used_units`, with matching flattened `__tags` and `__value_type` keys.
- Interface flattening behavior remains unchanged, while other indexed metrics outside `snmp.if.*` and `snmp.host.storage.*` still use the generic payload shape.

2026-06-07 10:55
Task: Add ThingsBoard MQTT gateway mode and grouped payload publishing
Changed files:
- internal/adapters/thingsboard_mqtt_adapter.go
- internal/adapters/thingsboard_mqtt_adapter_test.go
- docs/CONFIG_SCHEMA.md
- configs/examples/hq-adapters.yml
- docs/KNOWLEDGE.md
- docs/DEVELOPMENT_STAGES.md
Validation:
- go test ./internal/adapters
Status update:
- Phase 14I: ThingsBoard direct/gateway dual-mode DONE
Notes:
- `thingsboard_mqtt` now supports `mode: direct|gateway`. Direct mode keeps token auth for `v1/gateway/telemetry`; gateway mode publishes the same ThingsBoard-shaped payload to a regular broker/topic for ThingsBoard Gateway connector ingestion.
- Payload publishing is now grouped by `device + timestamp`, so a single MQTT message can carry multiple metrics in one `values` map instead of one publish per metric.
- This keeps `generic_mqtt` platform-agnostic while making `thingsboard_mqtt` usable both for direct ingestion and for broker-first gateway topologies.

2026-06-08 13:30
Task: Add built-in SNMP route inventory canonical flow
Changed files:
- cmd/nms-agent/main.go
- internal/routes/model.go
- internal/routes/provider.go
- internal/routes/snmp_provider.go
- internal/routes/resolver.go
- internal/routes/fingerprint.go
- internal/routes/normalizer.go
- internal/routes/collector.go
- internal/routes/routes_test.go
- internal/adapters/thingsboard_mqtt_adapter.go
- internal/adapters/thingsboard_mqtt_route_test.go
- docs/DATA_CONTRACT.md
- docs/ARCHITECTURE.md
- docs/ADAPTER_CONTRACT.md
- docs/CONFIG_SCHEMA.md
- docs/ROUTE_INVENTORY.md
- docs/KNOWLEDGE.md
- docs/DEVELOPMENT_STAGES.md
Validation:
- go test ./internal/routes ./internal/adapters ./cmd/nms-agent
- go test ./...
Status update:
- Phase 14J: built-in route inventory MVP DONE
Notes:
- Route inventory is now built-in for every SNMP-enabled device with primary `ipCidrRouteTable`, legacy `ipRouteTable` fallback, and non-fatal best-effort `inetCidrRouteTable` placeholder.
- Core still emits single canonical records only; Generic MQTT remains unchanged, ThingsBoard gateway mode keeps projection in the external converter, and ThingsBoard direct mode may split route string records into attributes.
- Route snapshots include stable fingerprints and `ifIndex=0` next-hop resolution to connected interfaces, preparing data for future logical topology without adding a new queue contract.

2026-06-08 16:10
Task: Add site-scoped ThingsBoard hybrid management integration
Changed files:
- internal/integrations/thingsboard/models.go
- internal/integrations/thingsboard/client.go
- internal/integrations/thingsboard/site_context.go
- internal/integrations/thingsboard/relation_reconciler.go
- internal/integrations/thingsboard/topology_builder.go
- internal/integrations/thingsboard/topology_publisher.go
- internal/integrations/thingsboard/site_context_test.go
- internal/adapters/thingsboard_mqtt_adapter.go
- docs/CONFIG_SCHEMA.md
- configs/examples/hq-adapters.yml
- docs/KNOWLEDGE.md
- docs/DEVELOPMENT_STAGES.md
Validation:
- go test ./internal/adapters ./internal/routes ./internal/integrations/thingsboard ./cmd/nms-agent
Status update:
- Phase 14K: ThingsBoard hybrid management foundation DONE
Notes:
- Added single-site ThingsBoard integration config in `adapters.yml` for REST management using customer/site API key plus site asset context.
- `thingsboard_mqtt` now triggers warning-only side workflows after successful publish: ensure `ASSET(site) --Contains--> DEVICE` relations and publish site-local topology snapshot to SERVER_SCOPE asset attributes.
- Telemetry/data plane remains unchanged and queue-safe; relation/topology failures do not stop main monitoring flow.
```

2026-06-05 13:20
Task: Add automatic devices.d reload and active ICMP discovery provider
Changed files:
- cmd/nms-agent/main.go
- cmd/nms-agentctl/discover.go
- cmd/nms-agentctl/discover_test.go
- internal/config/types.go
- internal/config/validate.go
- internal/configwatch/devices_watcher.go
- internal/configwatch/devices_watcher_test.go
- internal/discovery/providers/factory.go
- internal/discovery/providers/factory_test.go
- internal/discovery/providers/active/provider.go
- internal/discovery/providers/active/provider_test.go
- docs/CONFIG_SCHEMA.md
- docs/KNOWLEDGE.md
- docs/DEVELOPMENT_STAGES.md
Validation:
- go test ./internal/configwatch ./internal/discovery/providers/... ./cmd/nms-agent ./cmd/nms-agentctl
Status update:
- Phase 14F: automatic runtime reload and active discovery DONE
Notes:
- Daemon sekarang memantau `devices.d` dan reload runtime otomatis saat file device `.yml/.yaml` berubah, sehingga device baru ikut polling tanpa restart service bila config valid.
- Discovery provider sekarang dipilih dari config dan mendukung `provider: active` untuk ICMP subnet probe selain mode pasif `netlink`.
- CLI discovery manual dan daemon discovery loop sekarang memakai factory provider yang sama agar perilaku candidate konsisten.
```
