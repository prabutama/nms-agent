## FILE KNOWLEDGE TABLE

| File                           | Peran                                                                                                                                              |
| ------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------- |
| `go.mod`                       | Identitas modul Go. File ini menyimpan nama module, versi Go, dan dependency yang digunakan.                                                       |
| `cmd/nms-agent/main.go`        | Entrypoint agent service. Memilih collector runtime via `--collector-mode` lalu menjalankan pipeline (queue SQLite -> terminal adapter).          |
| `cmd/nms-agentctl/main.go`     | Entrypoint CLI admin. Nantinya dipakai untuk command seperti tambah device, cek status, queue status, reload, dan validasi config.                 |
| `internal/models/telemetry.go` | Tempat definisi **canonical telemetry format**. Ini format data netral internal agent setelah data dinormalisasi.                                  |
| `internal/collectors/port.go`  | Kontrak/interface untuk collector. Nantinya SNMP collector dan ICMP collector harus mengikuti interface ini.                                       |
| `internal/collectors/dummy_collector.go` | Dummy collector Phase 3. Menghasilkan raw sample deterministik untuk demo pipeline tanpa SNMP/ICMP.                                        |
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
| `internal/processors/port.go`  | Kontrak/interface untuk preprocessing dan normalization. Misalnya hitung throughput, latency, packet loss, jitter, lalu ubah ke telemetry standar. |
| `internal/processors/passthrough_processor.go` | Processor Phase 3 (passthrough). Memetakan RawSample dummy menjadi canonical telemetry sederhana.                               |
| `internal/processors/preprocess_threshold_processor.go` | Processor Phase 7: preprocessing minimal + evaluasi threshold untuk menambahkan tag status.                          |
| `internal/processors/preprocess_threshold_processor_test.go` | Unit test processor threshold: status critical/warning dan wildcard tag match.                                   |
| `internal/queue/port.go`       | Kontrak/interface untuk local queue. Nantinya implementasi SQLite store-and-forward harus mengikuti interface ini.                                 |
| `internal/queue/memory_queue.go` | Queue stub Phase 3 (in-memory). Untuk demo store-and-forward tanpa SQLite (tidak durable).                                                   |
| `internal/queue/sqlite_queue.go` | Implementasi queue durable Phase 4A berbasis SQLite. Menyimpan telemetry sebagai JSON dan melacak retry_count.                                |
| `internal/queue/sqlite_queue_stats.go` | Helper status read-only untuk SQLite queue: pending count dan max retry_count.                                                           |
| `internal/queue/sqlite_queue_test.go` | Test queue SQLite: memastikan data persist setelah restart + retry_count bertambah via MarkFailed.                                      |
| `internal/adapters/port.go`    | Kontrak/interface untuk output adapter. ThingsBoard, Generic MQTT, Terminal, Zabbix, Prometheus harus mengikuti interface ini.                     |
| `internal/adapters/terminal_adapter.go` | Terminal adapter Phase 3. Mencetak canonical telemetry ke stdout untuk debugging/demo.                                                   |
| `internal/core/pipeline.go`    | Orchestrator atau pengatur alur utama. File ini menghubungkan collector, processor, queue, dan adapter sesuai flow agent.                          |
| `internal/core/pipeline_sqlite_test.go` | Test integrasi pipeline+SQLite queue. Membuktikan data di-enqueue sebelum send dan tetap persist setelah restart.                      |
| `configs/agent.yml`            | Contoh konfigurasi utama agent (MVP). Mendefinisikan interval polling dan path file konfigurasi lainnya.                                           |
| `configs/devices.d/example-router.yml` | Contoh inventory device (MVP) termasuk toggle collector `icmp.enabled` / `snmp.enabled` untuk Phase 5.                                          |
| `configs/thresholds.yml`       | Konfigurasi threshold Phase 7 (metric/operator/warning/critical/tags) untuk evaluasi status.                                                     |
| `configs/adapters.yml`         | Placeholder konfigurasi adapter (Phase 8). Di Phase 2 hanya diload dan dicek struktur top-level key.                                              |
| `profiles/standard.yml`        | Profile SNMP standar (uptime + interface metrics) untuk semua device.                                                                           |
| `profiles/vendor-example.yml`  | Contoh profile vendor default untuk `vendor: example`.                                                                                          |
| `internal/config/types.go`     | Definisi struct config (root agent.yml, device entry, placeholders thresholds/adapters) untuk YAML loader dan validasi.                            |
| `internal/config/loader.go`    | Loader konfigurasi YAML. Membaca `agent.yml`, memuat `devices.d/*.yml`, thresholds, adapters, dan resolve path dengan `filepath`.                   |
| `internal/config/validate.go`  | Validasi dasar konfigurasi (field wajib, duplikasi device id, dan sanity-check struktur YAML thresholds/adapters).                                 |
| `internal/config/loader_test.go` | Unit test loader: path relatif + load devices directory.                                                                                        |
| `internal/config/validate_test.go` | Unit test validator: field wajib + duplikasi device id.                                                                                        |
| `cmd/nms-agentctl/validate.go` | Implementasi command `nms-agentctl validate` untuk load+validate config dan exit code yang sesuai.                                                |
| `cmd/nms-agentctl/queue_status.go` | Implementasi `nms-agentctl queue status` untuk menampilkan ringkasan queue SQLite (pending + max retry).                                    |
| `cmd/nms-agentctl/queue_retry.go` | Implementasi `nms-agentctl queue retry` untuk mencoba mengirim batch pending dari SQLite queue lalu ack (delivered) atau increment retry (failed). |
| `cmd/nms-agentctl/queue_retry_test.go` | Test CLI queue retry: seed SQLite queue lalu pastikan item pending terkirim dan terhapus.                                                |
| `go.sum`                       | Lockfile dependency Go modules hasil `go mod tidy`.                                                                                                |
| `Makefile`                     | Target build sederhana untuk fmt/test/vet/build/check (utama untuk environment non-Windows).                                                       |
| `make.bat`                     | Shim `make` untuk Windows. Mendukung target fmt/test/vet/build/check dengan memanggil perintah Go.                                                 |
| `docs/FLOW.md`                 | Diagram arsitektur dan alur runtime agent (CLI, config load/validate, pipeline, SQLite queue, adapter send, retry).                                |
