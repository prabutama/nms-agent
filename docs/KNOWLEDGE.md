## FILE KNOWLEDGE TABLE

| File                           | Peran                                                                                                                                              |
| ------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------- |
| `go.mod`                       | Identitas modul Go. File ini menyimpan nama module, versi Go, dan dependency yang digunakan.                                                       |
| `cmd/nms-agent/main.go`        | Entrypoint agent service. Memilih collector runtime via `--collector-mode`, adapter via factory (`terminal`/`tui`), lalu menjalankan pipeline berulang sesuai `poll_interval`. |
| `cmd/nms-agentctl/main.go`     | Entrypoint CLI admin. Nantinya dipakai untuk command seperti tambah device, cek status, queue status, reload, dan validasi config.                 |
| `internal/models/telemetry.go` | Definisi **canonical telemetry format** termasuk value_type dan value_number/value_string untuk data numerik maupun string.                        |
| `internal/collectors/port.go`  | Kontrak/interface untuk collector. Nantinya SNMP collector dan ICMP collector harus mengikuti interface ini.                                       |
| `internal/collectors/dummy_collector.go` | Dummy collector Phase 3. Menghasilkan raw sample deterministik (health ICMP + interface + resource) untuk demo pipeline tanpa SNMP/ICMP.    |
| `internal/collectors/targets.go` | Tipe `Target` minimal (DeviceID+Address) yang dipakai collector tanpa mengimpor package config.                                                        |
| `internal/collectors/icmp_collector.go` | ICMP collector Phase 5 (MVP) berbasis perintah `ping`: reachability, latency, jitter, packet loss.                                          |
| `internal/collectors/icmp_collector_test.go` | Unit test ICMP collector: parse output ping + partial snapshot saat error.                                                                |
| `internal/collectors/snmp_collector.go` | SNMP collector Phase 5 (MVP) menggunakan GoSNMP: uptime, ifOperStatus, dan counter octets per interface.                                     |
| `internal/collectors/snmp_collector_test.go` | Unit test SNMP collector dengan fake client: uptime + walk interface metrics.                                                             |
| `internal/profiles/types.go` | Definisi domain profile SNMP (match vendor/model dan daftar metrics OID).                                                                           |
| `internal/profiles/loader.go` | Loader YAML untuk membaca profile dari direktori `profiles/`.                                                                                   |
| `internal/profiles/validate.go` | Validasi profile: OID/metric duplikat dan standar profile wajib.                                                                              |
| `internal/profiles/loader_test.go` | Unit test loader profile YAML.                                                                                                             |
| `internal/profiles/validate_test.go` | Unit test validasi profile + precedence pemilihan profile.                                                                                 |
| `profiles/linux.yml`             | Profil SNMP untuk device vendor `linux` (termasuk Proxmox): system/uptime, host cpu+memory, hrStorage (type/desc + allocation/size/used units), plus UCD-SNMP-MIB memory/swap breakdown agar bisa tampil ala `free`. |
| `internal/processors/port.go`  | Kontrak/interface untuk preprocessing dan normalization. Misalnya hitung throughput, latency, packet loss, jitter, lalu ubah ke telemetry standar. |
| `internal/processors/passthrough_processor.go` | Processor Phase 3 (passthrough). Memetakan RawSample dummy menjadi canonical telemetry sederhana.                               |
| `internal/processors/preprocess_threshold_processor.go` | Processor Phase 7: preprocessing + threshold + derived throughput + multi-signal physical interface classifier (ifName, ifConnectorPresent, ifType) + normalizeMetrics (pct clamp, ms/seconds/bps≥0, reachable 0/1, unit default). |
| `internal/processors/preprocess_threshold_processor_test.go` | Unit test processor: threshold, derived metrics, multi-signal physical classifier (Proxmox/Docker/K8s patterns, connector present, wifi). |
| `internal/config/types.go`     | Definisi struct config (root agent.yml, device entry, placeholders thresholds/adapters) + `ResolvePath()` untuk relative path + env expansion, `Delivery.WithDefaults()`. |
| `internal/config/loader.go`    | Loader konfigurasi YAML. Membaca `agent.yml`, memuat `devices.d/*.yml`, thresholds, adapters, dan resolve path dengan `filepath`.                   |
| `internal/config/validate.go`  | Validasi dasar konfigurasi (field wajib, duplikasi device id, sanity-check YAML) + `ValidateThresholdRules()` untuk isolated rule validation.    |
| `internal/config/loader_test.go` | Unit test loader: path relatif + load devices directory.                                                                                        |
| `internal/config/validate_test.go` | Unit test validator: field wajib + duplikasi device id.                                                                                        |
| `cmd/nms-agentctl/validate.go` | Implementasi command `nms-agentctl validate` untuk load+validate config dan exit code yang sesuai.                                                |
| `cmd/nms-agentctl/queue_status.go` | Implementasi `nms-agentctl queue status` untuk menampilkan ringkasan queue SQLite (pending + max retry).                                    |
| `cmd/nms-agentctl/queue_retry.go` | Implementasi `nms-agentctl queue retry` untuk mencoba mengirim batch pending dari SQLite queue lalu ack (delivered) atau increment retry (failed). |
| `cmd/nms-agentctl/queue_retry_test.go` | Test CLI queue retry: seed SQLite queue lalu pastikan item pending terkirim dan terhapus.                                                |
| `cmd/nms-agentctl/threshold.go` | Implementasi `nms-agentctl threshold list` dan `threshold set`. List baca thresholds.yml dan print rules. Set upsert by metric+tags match, tulis atomic ke YAML. |
| `internal/adapters/factory.go` | Factory adapter berdasarkan nama (`terminal`/`tui`), dipanggil dari `cmd/nms-agent/main.go` gantikan hardcoded `NewTerminalAdapter()`.             |
| `internal/adapters/tui_adapter.go` | Adapter TUI: parsing config, start Bubble Tea program, `SendBatch()` inject telemetry via `Program.Send()`, `Close()` quit.                      |
| `internal/adapters/tui_model.go` | Model TUI: state per-device/per-interface, simpan health ICMP (reachable/latency/jitter/loss), dedup alerts, filter `snmp.if.*`, memory ala `free` (UCD) dengan fallback hrStorage. |
| `internal/adapters/tui_view.go` | View/layout TUI: 2-pane (device list + detail), truncation/MaxWidth anti-overlap, render Health (reachable/latency/jitter/loss), resources+Mem/Swap ala `free`. |
| `internal/adapters/tui_theme.go` | Tema/styling TUI (lipgloss styles) untuk header/cards/status warna.                                                                          |
| `internal/adapters/tui_keys.go` | Keymap global TUI + integrasi help bubble (short/full).                                                                                      |
| `internal/adapters/tui_format.go` | Helper format untuk throughput (bps -> K/M/Gbps) dan memory (KB -> Ki/Mi/Gi seperti `free -h`).                                                 |
| `internal/adapters/tui_adapter_test.go` | Smoke test TUI adapter (headless): SendBatch berbagai metric, multiple batch, close tanpa send.                                         |
| `go.sum`                       | Lockfile dependency Go modules hasil `go mod tidy`.                                                                                                |
| `cmd/nms-agentctl/threshold_test.go` | Unit test threshold CLI: add new rule, update existing by metric+tags match, append for different tags, missing metric fail, list output, parseTags, tagsEqual, ValidateThresholdRules. |
| `Makefile`                     | Target build sederhana untuk fmt/test/vet/build/check (utama untuk environment non-Windows).                                                       |
| `make.bat`                     | Shim `make` untuk Windows. Mendukung target fmt/test/vet/build/check dengan memanggil perintah Go.                                                 |
| `docs/FLOW.md`                 | Diagram arsitektur dan alur runtime agent (CLI, config load/validate, pipeline, SQLite queue, adapter send, retry).                                |
| `docs/CLI_COMMANDS.md`         | Contoh command CLI `nms-agentctl` untuk validate, queue status/retry, threshold list/set.                                                          |
| `docs/DATA_CONTRACT.md`        | Kontrak canonical telemetry (field wajib, tags threshold, derived metrics, normalization, termasuk hrStorage dan optional UCD memory/swap breakdown). |
| `docs/CONFIG_SCHEMA.md`        | Dokumentasi schema config YAML (agent.yml/adapters.yml/devices/thresholds) dan opsi adapter (terminal/tui).                                         |
| `docs/ADAPTER_CONTRACT.md`     | Kontrak adapter: aturan boundary adapter terhadap queue dan canonical telemetry, daftar adapter MVP.                                                |
| `docs/DEVELOPMENT_STAGES.md`   | Checklist phase/stage pengembangan + development log + catatan validasi per task.                                                                  |
