## FILE KNOWLEDGE TABLE

| File                           | Peran                                                                                                                                              |
| ------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------- |
| `go.mod`                       | Identitas modul Go. File ini menyimpan nama module, versi Go, dan dependency yang digunakan.                                                       |
| `cmd/nms-agent/main.go`        | Entrypoint agent service. Memilih collector runtime via `--collector-mode`, adapter via factory (`terminal`/`tui`/`generic_mqtt`/`thingsboard_mqtt`), menjalankan pipeline periodik, dan hot reload config via SIGHUP. |
| `cmd/nms-agent/reload_signal_unix.go` | Platform helper (non-Windows): definisikan signal reload (SIGHUP).                                                                      |
| `cmd/nms-agent/reload_signal_windows.go` | Platform helper (Windows): disable reload signal handling.                                                                            |
| `cmd/nms-agentctl/main.go`     | Entrypoint CLI admin. Menyediakan validate, reload, device management, queue, threshold, dan adapter health check. Default config: `/etc/nms-agent/agent.yml`. |
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
| `internal/config/timezone.go`  | Parser timezone config (`agent.output.timezone`) untuk presentation-only output (IANA atau fixed offset `UTC+7`).                                   |
| `internal/config/loader_test.go` | Unit test loader: path relatif + load devices directory.                                                                                        |
| `internal/config/validate_test.go` | Unit test validator: field wajib + duplikasi device id.                                                                                        |
| `cmd/nms-agentctl/validate.go` | Implementasi command `nms-agentctl validate` untuk load+validate config dan exit code yang sesuai.                                                |
| `cmd/nms-agentctl/reload.go`   | Implementasi `nms-agentctl reload`: validasi config lalu trigger hot reload agent (kirim SIGHUP ke PID).                                          |
| `cmd/nms-agentctl/reload_test.go` | Unit test reload CLI: memastikan arg wajib (`--pid`) divalidasi sebelum eksekusi signal.                                                      |
| `cmd/nms-agentctl/reload_signal_unix.go` | Platform helper (non-Windows): kirim SIGHUP ke proses agent.                                                                          |
| `cmd/nms-agentctl/reload_signal_windows.go` | Platform helper (Windows): reload via signal tidak didukung (arahkan pakai WSL/Linux).                                                 |
| `cmd/nms-agentctl/queue_status.go` | Implementasi `nms-agentctl queue status` untuk menampilkan ringkasan queue SQLite (pending + max retry).                                    |
| `cmd/nms-agentctl/queue_retry.go` | Implementasi `nms-agentctl queue retry` untuk mencoba mengirim batch pending dari SQLite queue lalu ack (delivered) atau increment retry (failed). |
| `cmd/nms-agentctl/queue_retry_test.go` | Test CLI queue retry: seed SQLite queue lalu pastikan item pending terkirim dan terhapus.                                                |
| `cmd/nms-agentctl/threshold.go` | Implementasi `nms-agentctl threshold list` dan `threshold set`. List baca thresholds.yml dan print rules. Set upsert by metric+tags match, tulis atomic ke YAML. |
| `cmd/nms-agentctl/device.go`   | Implementasi `nms-agentctl device` subcommands: list/add/update/remove/test (validasi + atomic write/rollback untuk perubahan file di `devices.d`). |
| `cmd/nms-agentctl/device_test.go` | Unit test device CLI: add/update/remove (write file baru, update field, hapus file) dan cek duplikasi id.                                       |
| `internal/adapters/factory.go` | Factory adapter berdasarkan nama (`terminal`/`tui`/`generic_mqtt`/`thingsboard_mqtt`), dipanggil dari `cmd/nms-agent/main.go` gantikan hardcoded `NewTerminalAdapter()`. |
| `internal/adapters/factory_test.go` | Unit test factory adapter: pastikan adapter yang didukung bisa dibuat (TUI headless) dan unknown name mengembalikan error.                     |
| `internal/adapters/output_timezone.go` | Konfigurasi global timezone untuk output adapter (terminal/TUI/MQTT) berdasarkan `agent.output.timezone`.                                      |
| `internal/adapters/mqtt_generic_adapter.go` | Generic MQTT adapter Phase 8: publish canonical telemetry JSON ke broker MQTT (config: broker/topic/qos/retain/auth/timeout + `strict_queue_mode`). |
| `internal/adapters/thingsboard_mqtt_adapter.go` | ThingsBoard MQTT adapter Phase 8: publish canonical telemetry ke ThingsBoard Gateway API (`v1/gateway/telemetry`) dengan metadata tags/threshold. |
| `internal/adapters/tui_adapter.go` | Adapter TUI: parsing config, start Bubble Tea program, `SendBatch()` inject telemetry via `Program.Send()`, `Close()` quit.                      |
| `internal/adapters/tui_model.go` | Model TUI: state per-device/per-interface, simpan health ICMP (reachable/latency/jitter/loss), dedup alerts, filter `snmp.if.*`, memory ala `free` (UCD) dengan fallback hrStorage. |
| `internal/adapters/tui_view.go` | View/layout TUI: 2-pane (device list + detail), truncation/MaxWidth anti-overlap, render Health (reachable/latency/jitter/loss), resources+Mem/Swap ala `free`. |
| `internal/adapters/tui_theme.go` | Tema/styling TUI (lipgloss styles) untuk header/cards/status warna.                                                                          |
| `internal/adapters/tui_keys.go` | Keymap global TUI + integrasi help bubble (short/full).                                                                                      |
| `internal/adapters/tui_format.go` | Helper format untuk throughput (bps -> K/M/Gbps) dan memory (KB -> Ki/Mi/Gi seperti `free -h`).                                                 |
| `internal/adapters/tui_adapter_test.go` | Smoke test TUI adapter (headless): SendBatch berbagai metric, multiple batch, close tanpa send.                                         |
| `internal/adapters/mqtt_generic_adapter_test.go` | Unit test generic MQTT adapter: validasi config + publish sukses/gagal/timeout tanpa broker real (fake client/token).                     |
| `internal/adapters/thingsboard_mqtt_adapter_test.go` | Unit test ThingsBoard MQTT adapter: validasi config + payload shape + publish error/strict mode tanpa broker real.                      |
| `go.sum`                       | Lockfile dependency Go modules hasil `go mod tidy`.                                                                                                |
| `cmd/nms-agentctl/threshold_test.go` | Unit test threshold CLI: add new rule, update existing by metric+tags match, append for different tags, missing metric fail, list output, parseTags, tagsEqual, ValidateThresholdRules. |
| `Makefile`                     | Target build sederhana untuk fmt/test/vet/build/check (utama untuk environment non-Windows).                                                       |
| `make.bat`                     | Shim `make` untuk Windows. Mendukung target fmt/test/vet/build/check dengan memanggil perintah Go.                                                 |
| `docs/FLOW.md`                 | Diagram arsitektur dan alur runtime agent (CLI, config load/validate, pipeline, SQLite queue, adapter send, retry).                                |
| `docs/CLI_COMMANDS.md`         | Contoh command CLI `nms-agentctl` untuk validate, device list, queue status/retry, adapter health, threshold list/set.                            |
| `docs/DATA_CONTRACT.md`        | Kontrak canonical telemetry (field wajib, tags threshold, derived metrics, normalization, termasuk hrStorage dan optional UCD memory/swap breakdown). |
| `docs/CONFIG_SCHEMA.md`        | Dokumentasi schema config YAML (agent.yml/adapters.yml/devices/thresholds) dan opsi adapter (terminal/tui/generic_mqtt/thingsboard_mqtt).           |
| `docs/ADAPTER_CONTRACT.md`     | Kontrak adapter: aturan boundary adapter terhadap queue dan canonical telemetry, daftar adapter MVP.                                                |
| `docs/DEVELOPMENT_STAGES.md`   | Checklist phase/stage pengembangan + development log + catatan validasi per task.                                                                  |
| `packaging/systemd/nms-agent.service` | Unit file systemd untuk menjalankan `nms-agent` sebagai service, termasuk `ExecReload` (SIGHUP).                                               |
| `packaging/systemd/install.sh` | Script install systemd (build dari repo): setup user/dir, install config, install unit, enable+start service.                                       |
| `packaging/systemd/README.md`  | Panduan operasi systemd: install, start/stop/reload, dan cara melihat log journald.                                                                 |
| `packaging/systemd/agent.yml`  | Sample config untuk deployment systemd (path absolut `/etc`, `/var/lib`, dll).                                                                      |
| `packaging/systemd/adapters.yml` | Sample adapter config untuk systemd deployment.                                                                                                   |
| `packaging/systemd/thresholds.yml` | Sample thresholds file untuk systemd deployment.                                                                                                 |
| `packaging/systemd/devices.d/example-linux-proxmox.yml` | Sample device entry untuk deployment (Linux/Proxmox).                                                                                  |
| `cmd/nms-agentctl/adapter_health.go` | Implementasi `nms-agentctl adapter health` untuk cek konektivitas adapter aktif (MQTT connect) tanpa mengirim telemetry.                      |
| `cmd/nms-agentctl/adapter_health_test.go` | Unit test `adapter health`: terminal ok, unknown adapter fail (menggunakan config temp).                                                     |
| `cmd/nms-agentctl/view.go`                 | Implementasi `nms-agentctl view` untuk connect ke daemon via Unix socket, menampilkan snapshot + live update telemetry dengan adapter-specific rendering dan timezone dari config.                      |
| `cmd/nms-agentctl/device.go`                 | CLI device management dengan wizard interaktif otomatis saat flag tidak lengkap (deteksi TTY).                                             |
| `internal/viewer/message.go`                 | Tipe pesan JSON untuk viewer client (`snapshot` / `telemetry` / `status` dengan status + details).                                                                             |
| `internal/viewer/hub.go`                     | Hub lokal untuk menyimpan snapshot telemetry dan broadcast live update ke subscriber + status updates (StatusUpdate).                                                         |
| `internal/viewer/server.go`                  | Unix socket server untuk daemon: menerima koneksi viewer, kirim snapshot + stream live.                                                        |
| `internal/viewer/client.go`                  | Unix socket client untuk `nms-agentctl view`: connect, baca snapshot, subscribe live.                                                          |
| `README.md`                      | Dokumentasi utama proyek: quick start, install, config reference, CLI commands, arsitektur, dan demo guide.                                   |
| `docs/TROUBLESHOOTING.md`        | Panduan troubleshooting: config errors, collector errors, queue errors, adapter errors, reload errors, systemd issues, dan performance.    |
| `docs/SECURITY.md`               | Panduan keamanan: credential handling, file permissions, network security, TLS, queue data, dan known security considerations.               |
| `configs/examples/hq-agent.yml`  | Contoh config agent untuk site HQ (poll 60s, delivery batch 200, timezone UTC+7).                                                           |
| `configs/examples/hq-adapters.yml` | Contoh adapter ThingsBoard MQTT untuk site HQ.                                                                                             |
| `configs/examples/hq-thresholds.yml` | Contoh threshold rules untuk site HQ (ICMP, interface, CPU, memory).                                                                   |
| `configs/examples/branch-agent.yml` | Contoh config agent untuk site Branch (poll 120s, delivery batch 100, timezone UTC+7).                                                    |
| `configs/examples/branch-adapters.yml` | Contoh adapter Generic MQTT untuk site Branch (forward ke central broker).                                                               |
| `configs/examples/branch-thresholds.yml` | Contoh threshold rules untuk site Branch (threshold lebih longgar dari HQ).                                                              |
| `configs/examples/hq-devices/hq-core-router.yml` | Contoh device config: core router MikroTik di HQ.                                                                                          |
| `configs/examples/hq-devices/hq-dist-switch.yml` | Contoh device config: distribution switch Cisco Catalyst di HQ.                                                                            |
| `configs/examples/hq-devices/hq-app-server.yml` | Contoh device config: server Ubuntu di HQ.                                                                                                 |
| `configs/examples/branch-devices/branch-edge-router.yml` | Contoh device config: edge router MikroTik di branch.                                                                                      |
| `configs/examples/branch-devices/branch-access-switch.yml` | Contoh device config: access switch Cisco di branch.                                                                                       |
| `packaging/RELEASE.md`           | Panduan release: build commands (Linux amd64/arm64, Windows), release package contents, deployment checklist, dan verification commands.    |
| `internal/config/types.go`       | Definisi struct config (root agent.yml, device entry, placeholders thresholds/adapters) + `ResolvePath()` untuk relative path + env expansion, `Delivery.WithDefaults()`, `ProfilesDir` pada `Paths`. |
| `internal/config/loader.go`      | Loader konfigurasi YAML. Membaca `agent.yml`, memuat `devices.d/*.yml`, thresholds, adapters, profiles, dan resolve path dengan `filepath`.   |
