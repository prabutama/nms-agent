## FILE KNOWLEDGE TABLE

| File                           | Peran                                                                                                                                              |
| ------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------- |
| `go.mod`                       | Identitas modul Go. File ini menyimpan nama module, versi Go, dan dependency yang digunakan.                                                       |
| `cmd/nms-agent/main.go`        | Entrypoint utama agent service. Nantinya ini yang dijalankan oleh systemd sebagai `nms-agent`.                                                     |
| `cmd/nms-agentctl/main.go`     | Entrypoint CLI admin. Nantinya dipakai untuk command seperti tambah device, cek status, queue status, reload, dan validasi config.                 |
| `internal/models/telemetry.go` | Tempat definisi **canonical telemetry format**. Ini format data netral internal agent setelah data dinormalisasi.                                  |
| `internal/collectors/port.go`  | Kontrak/interface untuk collector. Nantinya SNMP collector dan ICMP collector harus mengikuti interface ini.                                       |
| `internal/collectors/dummy_collector.go` | Dummy collector Phase 3. Menghasilkan raw sample deterministik untuk demo pipeline tanpa SNMP/ICMP.                                        |
| `internal/processors/port.go`  | Kontrak/interface untuk preprocessing dan normalization. Misalnya hitung throughput, latency, packet loss, jitter, lalu ubah ke telemetry standar. |
| `internal/processors/passthrough_processor.go` | Processor Phase 3 (passthrough). Memetakan RawSample dummy menjadi canonical telemetry sederhana.                               |
| `internal/queue/port.go`       | Kontrak/interface untuk local queue. Nantinya implementasi SQLite store-and-forward harus mengikuti interface ini.                                 |
| `internal/queue/memory_queue.go` | Queue stub Phase 3 (in-memory). Untuk demo store-and-forward tanpa SQLite (tidak durable).                                                   |
| `internal/queue/sqlite_queue.go` | Implementasi queue durable Phase 4A berbasis SQLite. Menyimpan telemetry sebagai JSON dan melacak retry_count.                                |
| `internal/queue/sqlite_queue_test.go` | Test queue SQLite: memastikan data persist setelah restart + retry_count bertambah via MarkFailed.                                      |
| `internal/adapters/port.go`    | Kontrak/interface untuk output adapter. ThingsBoard, Generic MQTT, Terminal, Zabbix, Prometheus harus mengikuti interface ini.                     |
| `internal/adapters/terminal_adapter.go` | Terminal adapter Phase 3. Mencetak canonical telemetry ke stdout untuk debugging/demo.                                                   |
| `internal/core/pipeline.go`    | Orchestrator atau pengatur alur utama. File ini menghubungkan collector, processor, queue, dan adapter sesuai flow agent.                          |
| `internal/core/pipeline_sqlite_test.go` | Test integrasi pipeline+SQLite queue. Membuktikan data di-enqueue sebelum send dan tetap persist setelah restart.                      |
| `configs/agent.yml`            | Contoh konfigurasi utama agent (MVP). Mendefinisikan interval polling dan path file konfigurasi lainnya.                                           |
| `configs/devices.d/example-router.yml` | Contoh inventory device (MVP). Berisi `id`, `address`, dan metadata vendor/model untuk fase berikutnya.                                     |
| `configs/thresholds.yml`       | Placeholder konfigurasi threshold (Phase 7). Di Phase 2 hanya diload dan dicek struktur top-level key.                                            |
| `configs/adapters.yml`         | Placeholder konfigurasi adapter (Phase 8). Di Phase 2 hanya diload dan dicek struktur top-level key.                                              |
| `internal/config/types.go`     | Definisi struct config (root agent.yml, device entry, placeholders thresholds/adapters) untuk YAML loader dan validasi.                            |
| `internal/config/loader.go`    | Loader konfigurasi YAML. Membaca `agent.yml`, memuat `devices.d/*.yml`, thresholds, adapters, dan resolve path dengan `filepath`.                   |
| `internal/config/validate.go`  | Validasi dasar konfigurasi (field wajib, duplikasi device id, dan sanity-check struktur YAML thresholds/adapters).                                 |
| `internal/config/loader_test.go` | Unit test loader: path relatif + load devices directory.                                                                                        |
| `internal/config/validate_test.go` | Unit test validator: field wajib + duplikasi device id.                                                                                        |
| `cmd/nms-agentctl/validate.go` | Implementasi command `nms-agentctl validate` untuk load+validate config dan exit code yang sesuai.                                                |
| `go.sum`                       | Lockfile dependency Go modules hasil `go mod tidy`.                                                                                                |
| `Makefile`                     | Target build sederhana untuk fmt/test/vet/build/check (utama untuk environment non-Windows).                                                       |
| `make.bat`                     | Shim `make` untuk Windows. Mendukung target fmt/test/vet/build/check dengan memanggil perintah Go.                                                 |
