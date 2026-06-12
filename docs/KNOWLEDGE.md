## FILE KNOWLEDGE TABLE

| File                           | Peran                                                                                                                                              |
| ------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------- |
| `go.mod`                       | Identitas modul Go. File ini menyimpan nama module, versi Go, dan dependency yang digunakan.                                                       |
| `cmd/nms-agent/main.go`        | Entrypoint agent service. Memilih collector runtime via `--collector-mode`, adapter via factory (`tui`/`generic_mqtt`/`thingsboard_mqtt`), menjalankan pipeline periodik, auto-reload `devices.d` via watcher, dan discovery loop dengan provider terpilih (`netlink`/`active`). |
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
| `internal/config/types.go`     | Definisi struct config (root agent.yml, device entry, thresholds/adapters, discovery termasuk `active_probe`) + `ResolvePath()` untuk relative path + env expansion, `Delivery.WithDefaults()`. |
| `internal/config/loader.go`    | Loader konfigurasi YAML. Membaca `agent.yml`, memuat `devices.d/*.yml`, thresholds, adapters, discovery settings, dan resolve path dengan `filepath`.                   |
| `internal/config/validate.go`  | Validasi dasar konfigurasi (field wajib, duplikasi device id, sanity-check YAML), termasuk blok discovery (`provider=netlink|active` dan `active_probe`), + `ValidateThresholdRules()` untuk isolated rule validation.    |
| `internal/configwatch/devices_watcher.go` | Watcher runtime untuk direktori `devices.d`: debounce perubahan file `.yml/.yaml` lalu memicu reload config daemon tanpa restart service. |
| `internal/configwatch/devices_watcher_test.go` | Unit test filter event watcher `devices.d` agar hanya perubahan file device yang memicu reload. |
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
| `cmd/nms-agentctl/device.go`   | Implementasi `nms-agentctl device` subcommands: list/add/update/remove/test (validasi + atomic write/rollback, sanitasi input wizard, validasi id/address/vendor/model). |
| `cmd/nms-agentctl/device_test.go` | Unit test device CLI: add/update/remove (write file baru, update field, hapus file) dan cek duplikasi id.                                       |
| `internal/adapters/factory.go` | Factory adapter berdasarkan nama (`tui`/`generic_mqtt`/`thingsboard_mqtt`), dipanggil dari `cmd/nms-agent/main.go` gantikan hardcoded `NewTerminalAdapter()`. |
| `internal/adapters/factory_test.go` | Unit test factory adapter: pastikan adapter yang didukung bisa dibuat (TUI headless) dan unknown name mengembalikan error.                     |
| `internal/adapters/output_timezone.go` | Konfigurasi global timezone untuk output adapter (terminal/TUI/MQTT) berdasarkan `agent.output.timezone`.                                      |
| `internal/adapters/mqtt_generic_adapter.go` | Generic MQTT adapter Phase 8: publish canonical telemetry JSON ke broker MQTT (config: broker/topic/qos/retain/auth/timeout + `strict_queue_mode`). |
| `internal/adapters/thingsboard_mqtt_adapter.go` | ThingsBoard MQTT adapter Phase 8: mendukung mode `direct` (Gateway API token auth) dan `gateway` (publish ke broker untuk ThingsBoard Gateway connector), mengagregasi payload per device+timestamp, merapikan key `snmp.if.*`/`snmp.host.storage.*`, di direct mode dapat memproyeksikan route string records ke attributes, serta menjalankan management side-effects (relation reconciler + topology publisher + alarm manager) dengan stderr diagnostic logging. |
| `internal/adapters/thingsboard_mqtt_route_test.go` | Unit test projection route inventory di ThingsBoard direct mode: route summary tetap telemetry sementara route string detail dipublish sebagai attributes. |
| `internal/integrations/thingsboard/models.go` | Model integration ThingsBoard untuk site context, relation, alarm, device info, dan topology snapshot site-local. |
| `internal/integrations/thingsboard/client.go` | REST client ThingsBoard berbasis tenant API key untuk check auth, lookup relation/device, create/clear alarm, dan publish SERVER_SCOPE attributes asset site. |
| `internal/integrations/thingsboard/alarm_manager.go` | Orchestrator alarm lifecycle berbasis threshold telemetry: mapping metric→alarm type, create/update saat critical/warning, clear saat normal, dan cache alarm per device+metric. |
| `internal/integrations/thingsboard/site_context.go` | Validasi site context single-site per agent (`thingsboard.site`) agar integration hybrid tetap sederhana saat deploy per site. |
| `internal/integrations/thingsboard/relation_reconciler.go` | Reconciler warning-only untuk memastikan relation `ASSET(site) --Contains--> DEVICE` tanpa mengganggu telemetry utama; graceful continue saat device lookup gagal. |
| `internal/integrations/thingsboard/topology_builder.go` | Builder topology logis IPv4 site-local dari route snapshot canonical: device, subnet, dan external gateway nodes + edges. |
| `internal/integrations/thingsboard/topology_publisher.go` | Publisher topology site-local ke SERVER_SCOPE attributes asset site hanya saat fingerprint berubah. |
| `internal/integrations/thingsboard/site_context_test.go` | Unit test validasi site context dan build topology site-local minimal untuk integration layer ThingsBoard. |
| `internal/adapters/tui_adapter.go` | Adapter TUI: parsing config, start Bubble Tea program, `SendBatch()` inject telemetry via `Program.Send()`, `Close()` quit.                      |
| `internal/adapters/tui_model.go` | Model TUI: state per-device/per-interface, simpan health ICMP (reachable/latency/jitter/loss), dedup alerts, filter `snmp.if.*`, memory ala `free` (UCD) dengan fallback hrStorage. |
| `internal/adapters/tui_view.go` | View/layout TUI: 2-pane (device list + detail), truncation/MaxWidth anti-overlap, render Health (reachable/latency/jitter/loss), resources+Mem/Swap ala `free`. |
| `internal/adapters/tui_theme.go` | Tema/styling TUI (lipgloss styles) untuk header/cards/status warna.                                                                          |
| `internal/adapters/tui_keys.go` | Keymap global TUI + integrasi help bubble (short/full).                                                                                      |
| `internal/adapters/tui_format.go` | Helper format untuk throughput (bps -> K/M/Gbps) dan memory (KB -> Ki/Mi/Gi seperti `free -h`).                                                 |
| `internal/adapters/tui_adapter_test.go` | Smoke test TUI adapter (headless): SendBatch berbagai metric, multiple batch, close tanpa send.                                         |
| `internal/adapters/tui_state.go` | State shared antara TUI adapter dan CLI summary: reducer ApplyBatch, helper DeviceCounts/AlertCounts/SortedDevices/dll.                              |
| `internal/discovery/types.go` | Tipe inti discovery Milestone A: candidate, fingerprint, result, provider, dan prober untuk auto-discovery pasif.                              |
| `internal/discovery/service.go` | Orchestrator discovery sekali jalan: ambil kandidat dari provider, probe SNMP, resolve profile, dan auto-promote device baru.                              |
| `internal/discovery/snmp_probe.go` | Probe SNMP ringan untuk fingerprint discovery (`sysObjectID`, `sysName`, `sysDescr`) dengan config community/timeout/retries, termasuk normalisasi nama OID response sebelum mapping field fingerprint.                              |
| `internal/discovery/snmp_probe_test.go` | Unit test parser dan mapping fingerprint discovery untuk `sysObjectID` `ObjectIdentifier` serta normalisasi OID (`iso.` dan leading dot pada value/nama response).                              |
| `internal/discovery/resolver.go` | Resolver fingerprint -> vendor/model berbasis `sysObjectID` dan heuristic `sysDescr` untuk profile matching discovery.                              |
| `internal/discovery/promote.go` | Renderer `device_id_template`, collision suffix, dan atomic writer untuk auto-promote device hasil discovery ke `devices.d`.                              |
| `internal/discovery/service_test.go` | Unit test discovery service: promote known profile, collision suffix, promotion limit, dan skip unknown-standard-only profile.                              |
| `internal/discovery/explorer/explorer.go` | Safe exploration Milestone B: probe katalog OID aman, generate profile YAML, dan simpan ke `profiles/` untuk auto-approve/auto-promote.                              |
| `internal/discovery/explorer/explorer_test.go` | Unit test helper exploration: generated match fallback dan write generated profile file.                              |
| `internal/discovery/providers/factory.go` | Factory provider discovery berdasarkan config untuk memilih mode pasif `netlink` atau active ICMP probe.                              |
| `internal/discovery/providers/factory_test.go` | Unit test factory provider discovery agar mode config memetakan ke provider yang benar.                              |
| `internal/discovery/providers/active/provider.go` | Provider discovery aktif berbasis ICMP: enumerasi host subnet, probe reachability paralel, lalu hasilkan candidate tanpa menunggu ARP/neighbor table.                              |
| `internal/discovery/providers/active/provider_test.go` | Unit test helper provider active untuk memastikan host subnet mengabaikan network/broadcast/local IP.                              |
| `internal/discovery/providers/netlink/provider_linux.go` | Provider discovery Linux: baca interface + ARP/neighbor table via netlink lalu filter kandidat dalam subnet target.                              |
| `internal/discovery/providers/netlink/provider_stub.go` | Stub provider non-Linux agar build lintas platform tetap aman saat discovery provider `netlink` tidak tersedia.                              |
| `internal/adapters/mqtt_generic_adapter_test.go` | Unit test generic MQTT adapter: validasi config + publish sukses/gagal/timeout tanpa broker real (fake client/token).                     |
| `internal/adapters/thingsboard_mqtt_adapter_test.go` | Unit test ThingsBoard MQTT adapter: validasi config + payload shape + flattened `snmp.if.*` dan `snmp.host.storage.*` keys, sanitasi nama interface, guard indexed metric lain tetap generic, serta publish error/strict mode tanpa broker real.                      |
| `go.sum`                       | Lockfile dependency Go modules hasil `go mod tidy`.                                                                                                |
| `cmd/nms-agentctl/threshold_test.go` | Unit test threshold CLI: add new rule, update existing by metric+tags match, append for different tags, missing metric fail, list output, parseTags, tagsEqual, ValidateThresholdRules. |
| `Makefile`                     | Target build sederhana untuk fmt/test/vet/build/check (utama untuk environment non-Windows).                                                       |
| `make.bat`                     | Shim `make` untuk Windows. Mendukung target fmt/test/vet/build/check dengan memanggil perintah Go.                                                 |
| `docs/FLOW.md`                 | Diagram arsitektur dan alur runtime agent (CLI, config load/validate, pipeline, SQLite queue, adapter send, retry).                                |
| `docs/CLI_COMMANDS.md`         | Contoh command CLI `nms-agentctl` untuk validate, device list, queue status/retry, adapter health, threshold list/set.                            |
| `docs/DATA_CONTRACT.md`        | Kontrak canonical telemetry (field wajib, tags threshold, derived metrics, normalization, termasuk hrStorage dan optional UCD memory/swap breakdown). |
| `docs/CONFIG_SCHEMA.md`        | Dokumentasi schema config YAML (agent.yml/adapters.yml/devices/thresholds/discovery) dan opsi adapter/discovery.           |
| `docs/ADAPTER_CONTRACT.md`     | Kontrak adapter: aturan boundary adapter terhadap queue dan canonical telemetry, termasuk projection canonical string records ke attribute channel platform bila diperlukan.                                                |
| `docs/ROUTE_INVENTORY.md`      | Dokumen perilaku route inventory built-in: urutan provider, canonical outputs, resolusi interface, dan persiapan data untuk topology builder. |
| `docs/thingsboard.json`        | OpenAPI spec ThingsBoard yang dipakai sebagai referensi endpoint REST untuk asset, device, relation, attributes, dan otomasi manajemen. |
| `docs/DEVELOPMENT_STAGES.md`   | Checklist phase/stage pengembangan + development log + catatan validasi per task.                                                                  |
| `packaging/systemd/nms-agent.service` | Unit file systemd untuk menjalankan `nms-agent` sebagai service, termasuk `ExecReload` (SIGHUP).                                               |
| `packaging/systemd/nms-agent.env` | Contoh file environment untuk deployment systemd agar adapter config dapat memakai `${TB_URL}` dan API key ThingsBoard tanpa hardcode di YAML.                                                                 |
| `packaging/systemd/install.sh` | Script install systemd (build dari repo): setup user/dir, install config, install unit, enable+start service.                                       |
| `packaging/systemd/README.md`  | Panduan operasi systemd: install, start/stop/reload, dan cara melihat log journald.                                                                 |
| `packaging/systemd/agent.yml`  | Sample config untuk deployment systemd (path absolut `/etc`, `/var/lib`, dll).                                                                      |
| `packaging/systemd/adapters.yml` | Sample adapter config untuk systemd deployment.                                                                                                   |
| `packaging/systemd/thresholds.yml` | Sample thresholds file untuk systemd deployment.                                                                                                 |
| `packaging/systemd/devices.d/example-linux-proxmox.yml` | Sample device entry untuk deployment (Linux/Proxmox).                                                                                  |
| `cmd/nms-agentctl/adapter_health.go` | Implementasi `nms-agentctl adapter health` untuk cek konektivitas adapter aktif (MQTT connect) tanpa mengirim telemetry.                      |
| `cmd/nms-agentctl/adapter_health_test.go` | Unit test `adapter health`: terminal ok, unknown adapter fail (menggunakan config temp).                                                     |
| `cmd/nms-agentctl/discover.go` | CLI discovery manual: `status`, `preview`, dan `run` untuk observability/eksekusi discovery di luar loop daemon, memakai provider discovery yang sama dengan daemon.                                                     |
| `cmd/nms-agentctl/view.go`                 | Implementasi `nms-agentctl view` untuk connect ke daemon via Unix socket; mode `summary` sekarang menjaga state lokal, redraw in-place pada terminal interaktif, dan menampilkan tabel status per-device (status, last seen, latency, loss, alerts).                      |
| `cmd/nms-agentctl/device.go`                 | CLI device management dengan wizard interaktif otomatis saat flag tidak lengkap (deteksi TTY).                                             |
| `internal/viewer/message.go`                 | Tipe pesan JSON untuk viewer client (`snapshot` / `telemetry` / `status` dengan status + details).                                                                             |
| `internal/viewer/hub.go`                     | Hub lokal untuk menyimpan snapshot telemetry (merge kumulatif per device+metric+ifIndex) dan broadcast live update ke subscriber + status updates (StatusUpdate).                                                         |
| `internal/viewer/server.go`                  | Unix socket server untuk daemon: menerima koneksi viewer, kirim snapshot + stream live.                                                        |
| `internal/viewer/client.go`                  | Unix socket client untuk `nms-agentctl view`: connect, baca snapshot, subscribe live.                                                          |
| `internal/viewer/hub_test.go`                | Unit test merge snapshot viewer: multi-device merge, replace same key, different ifIndex.                                                          |
| `internal/routes/model.go`                   | Model canonical route inventory (`RouteEntry`, `RouteSnapshot`) dan konstanta source table SNMP untuk future logical topology. |
| `internal/routes/provider.go`                | Kontrak provider route inventory agar source pengambilan route tetap terpisah dari runtime collector dan platform adapter. |
| `internal/routes/snmp_provider.go`           | Provider SNMP route inventory IPv4 dengan prioritas `ipCidrRouteTable`, fallback legacy `ipRouteTable`, best-effort `inetCidrRouteTable`, plus lookup nama interface IF-MIB. |
| `internal/routes/resolver.go`                | Resolver route inventory: mapping protocol/type per source table, deteksi default route, urutan stabil, dan resolusi `ifIndex=0` via connected route. |
| `internal/routes/fingerprint.go`             | Fingerprint route snapshot dan cache perubahan per device/address-family untuk menghasilkan flag `route.ipv4.changed`. |
| `internal/routes/normalizer.go`              | Normalizer route snapshot menjadi canonical raw records summary/default-route/snapshot string tanpa menambah kontrak queue baru. |
| `internal/routes/collector.go`               | Collector built-in route inventory untuk semua device SNMP-enabled; unsupported route tables tidak menggagalkan cycle utama. |
| `internal/routes/routes_test.go`             | Unit test route inventory: parsing ipCidr/legacy, resolusi interface, fingerprint, snapshot limit, dan non-fatal unsupported behavior. |
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
