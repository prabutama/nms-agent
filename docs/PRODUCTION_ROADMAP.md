# Production Readiness Roadmap

> Dokumen ini berisi rencana bertahap untuk membawa nms-agent dari kondisi
> "berjalan sesuai kebutuhan" ke tingkat **production-stable** tanpa merusak
> flow inti `collect → preprocess → normalize → queue → adapter send`.
>
> Roadmap only. Isi file ini adalah target hardening dan rollout, bukan jaminan
> bahwa semua item sudah enforced oleh implementasi saat ini.

---

## Prinsip Eksekusi

1. Jangan ubah kontrak core domain/ports.
2. Jangan ubah format canonical telemetry.
3. Jangan ubah adapter behavior default.
4. Semua fitur baru wajib punya default aman: `disabled` atau backward-compatible.
5. Tiap fase wajib diverifikasi sebelum lanjut ke fase berikutnya.

## Guardrail Teknis

- `internal/core/pipeline.go` tetap sebagai pusat flow — jangan dirombak total.
- `internal/queue/sqlite_queue.go` hanya ditambah kemampuan kecil; tidak perlu
  rewrite.
- Konfig baru harus opsional dengan nilai default yang setara perilaku lama.
- Perubahan observability tidak boleh mengubah hasil delivery.
- Perubahan retry tidak boleh aktif agresif sebelum lulus test outage.

---

## Phase 0: Baseline & Safety Net

**Tujuan:** Bekukan dan dokumentasikan perilaku sekarang sebagai acuan.

### Scope

- Petakan flow runtime utama (`cmd/nms-agent/main.go` loop).
- Catat baseline untuk skenario:
  - Poll normal
  - Adapter down
  - Restart daemon (SIGTERM)
  - Reload config (SIGHUP / SIGUSR1)
  - `nms-agentctl queue retry`
- Definisikan invariants yang tidak boleh dilanggar:
  - Telemetry masuk queue **sebelum** delivery.
  - Item gagal tetap `pending`.
  - Item sukses hilang dari queue.
  - Reload tidak membutuhkan restart service.

### Output

- Baseline behavior matrix (file di `docs/`).
- Daftar invariant yang akan diuji otomatis nanti.

### Verifikasi

```bash
make check
nms-agentctl validate --config configs/agent.yml
nms-agentctl queue status --config configs/agent.yml
nms-agentctl adapter health --config configs/agent.yml
```

### Risiko

Rendah. Belum ada perubahan kode.

---

## Phase 1: Observability Minimum

**Tujuan:** Agent bisa dipantau tanpa menyentuh flow data inti.

### Scope

- **Structured logging** (level + timestamp + key=value).
  - Level bawaan: `info` (cocok output sekarang).
  - Format: `text` (default, seperti sekarang) dan `json` (opsional).
- **Event penting per cycle:**
  - Poll start/end
  - Collect result (jumlah sample, error per device)
  - Normalize count
  - Enqueue count
  - Delivery batch result (success/fail, jumlah item)
  - Reload success/fail
- **Error classification dasar** di log: `transient` vs `permanent`.
- **Queue telemetry** `pending` + `max_retry` otomatis tiap cycle.

### Config baru

```yaml
agent:
  logging:
    level: info       # debug | info | warn | error
    format: text      # text | json
```

Default: `level=info`, `format=text` — tidak mengubah output sekarang.

### File yang disentuh

- `cmd/nms-agent/main.go` — tambah log event.
- `internal/core/pipeline.go` — bungkus event pipeline.
- Package baru `internal/telemetry/` opsional untuk log helpers.

### Verifikasi

- Log masih muncul di stdout/journald.
- `--collector-mode auto` tetap jalan.
- Jumlah queue item tidak berubah karena logging.
- Semua test existing `go test ./...` hijau.

### Risiko

- Volume log naik. Mitigasi: level `info` default, `debug` untuk troubleshooting.

---

## Phase 2: Health & Runtime Introspection

**Tujuan:** Operator bisa cek status agent secara real-time tanpa baca log
mentah.

### Scope

- **Runtime status** via `nms-agentctl status`:
  - Active adapter
  - Poll interval
  - Last cycle: success/fail, timestamp, duration
  - Queue pending count
  - Queue max retry
  - Uptime daemon
- **Health surface** ringan:
  - Liveness: proses hidup.
  - Readiness: config valid, queue bisa dibuka, adapter terdaftar.

### Pendekatan

Gunakan Unix socket `/run/nms-agent/status.sock` yang sudah ada (viewer hub)
atau tambah endpoint status terpisah. HTTP localhost bisa jadi opsi nanti.

### File yang disentuh

- `cmd/nms-agent/main.go` — publish status snapshot.
- `cmd/nms-agentctl/status.go` — command baru.
- `internal/viewer/` — extend hub jika dipakai.

### Verifikasi

- `nms-agentctl status` mengembalikan data valid saat agent hidup.
- Status tetap aman dibaca saat adapter mati.
- Tidak mengganggu cycle polling.

### Risiko

- Race kondisi baca state runtime. Mitigasi: snapshot dengan mutex read-only.

---

## Phase 3: CI & Quality Gate

**Tujuan:** Mencegah regressions umum sebelum masuk staging/production.

### Scope

Pipeline CI minimal:

| Step | Command |
|------|---------|
| Format | `gofmt -l .` (fail jika ada diff) |
| Test | `go test ./...` |
| Race | `go test -race ./...` |
| Vet | `go vet ./...` |
| Lint | `staticcheck ./...` |
| Vuln | `govulncheck ./...` |
| Coverage | `go test -coverprofile=coverage.out ./...` |

### Update Makefile

```makefile
.PHONY: fmt test vet lint vuln race check-all

race:
	go test -race ./...

lint:
	staticcheck ./...

vuln:
	govulncheck ./...

check-all: fmt test race vet lint vuln
```

### Verifikasi

- `make check-all` hijau di Linux (target deployment).
- Race detector tidak menemukan data race di queue/watcher/viewer.
- Coverage minimal tidak turun drastis tanpa alasan.

### Risiko

- `staticcheck` atau `govulncheck` mungkin false positive. Mitigasi:
  diperiksa manual, bukan auto-block.

---

## Phase 4: Queue Reliability Hardening (Non-Disruptive)

**Tujuan:** Tambah retry backoff dan sinkronisasi tanpa merusak behavior
lama.

### Scope kecil (4a — schema & code path)

- Kolom baru di `queue_items`:
  - `next_attempt_at TEXT` (RFC3339Nano)
  - `last_attempt_at TEXT` (RFC3339Nano) opsional
- `PendingBatch()` hanya ambil item yang `next_attempt_at <= now()`.
- Saat `MarkFailed`, set `next_attempt_at` sesuai backoff.
- Code path **lama tetap jalan** jika fitur retry disabled.

### Config baru

```yaml
agent:
  delivery:
    retry:
      enabled: false        # default false = behavior lama
      base_backoff: 10s
      max_backoff: 300s
```

### Tahapan rollout

| Sub-fase | Aktivitas |
|----------|-----------|
| 4a | Schema extend + code path diam (off). Test migration. |
| 4b | Aktifkan di staging. Outage simulation. |
| 4c | Rollout production bertahap. |

### Tidak dikerjakan sekarang

- Dead-letter queue (Phase 5).
- TTL/expiry drop.
- Deduplikasi.

### Verifikasi

- Item gagal tetap `pending`.
- Item sukses tetap `DELETE`.
- Restart tetap preserve `retry_count` dan `next_attempt_at`.
- Queue drain normal saat adapter pulih.

### Risiko

- Query `next_attempt_at` salah bisa bikin queue macet.
  Mitigasi: fallback ke `status='pending'` tanpa filter waktu jika fitur
  disabled atau error.

---

## Phase 5: Dead-Letter & Queue Retention

**Tujuan:** Mencegah retry tanpa ujung dan pertumbuhan queue tak terbatas.

### Scope

- **Max retry limit:** item melebihi `max_retries` pindah ke `dead_letter`.
- **State baru:** `dead_letter` atau tabel `dead_letter_items`.
- **Cleanup policy:** hapus item `dead_letter` atau `pending` yang sudah
  lebih dari `retention_days`.
- **CLI status tambah:**
  - `dead_letter_count`
  - `oldest_pending_age`

### Config baru

```yaml
agent:
  delivery:
    retry:
      max_retries: 10
      dead_letter_enabled: false    # default off
    retention_days: 30
```

### Verifikasi

- Item permanent error tidak muter selamanya.
- Item dead-letter tidak ikut `PendingBatch()`.
- Operator bisa inspect dead-letter via CLI.
- `retention_days` membatasi umur item di DB.

### Risiko

- False positive permanent error bisa buang data. Mitigasi: klasifikasi
  error konservatif dulu — kebanyakan error dianggap transient.

---

## Phase 6: Failure Isolation Per Device

**Tujuan:** Satu device bermasalah tidak menggagalkan polling semua device
lain.

### Scope

- Collector `best-effort` yang lebih kuat:
  - `ICMPCollector` dan `SNMPCollector` lanjut ke device berikutnya walau
    satu device timeout.
- Error agregat per cycle, bukan fail global.
- `combinedCollector()` jangan stop di error pertama.
- Hasil valid tetap lanjut ke `Normalize` dan `Enqueue`.

### File yang disentuh

- `internal/collectors/icmp_collector.go`
- `internal/collectors/snmp_collector.go`
- `internal/routes/collector.go`
- `internal/collectors/dummy_collector.go`
- `cmd/nms-agent/main.go` — bagian `buildCollector`

### Strategi aman

- Jangan ubah interface `collectors.Collector`. Ubah implementasi internal.
- Tambah `PartialResult` yang bisa bawa error per-device + data valid.
- Pipeline tetap lihat array sample seperti sekarang.

### Verifikasi

- 1 device SNMP timeout, device lain tetap terkirim.
- Queue berisi data valid dari device yang sukses.
- Error per-device tercatat di log.

### Risiko

- Test existing mungkin expect all-or-nothing. Mitigasi: update test
  ekspektasi partial-success.

---

## Phase 7: Systemd Hardening & Resource Limits

**Tujuan:** Operasi host lebih aman dan stabil.

### Scope

Unit service diperkuat:

```ini
[Service]
Restart=always
RestartSec=5
TimeoutStopSec=30
StartLimitIntervalSec=300
StartLimitBurst=5
MemoryMax=256M
LimitNOFILE=65536
StateDirectory=nms-agent
LogsDirectory=nms-agent
```

Pastikan `ReadWritePaths` sesuai dengan path queue dan runtime socket yang
aktual.

### File yang disentuh

- `packaging/systemd/nms-agent.service`
- `packaging/systemd/install.sh` — jika ada perubahan path.

### Verifikasi

- `systemctl start nms-agent` sukses.
- `systemctl reload nms-agent` tetap jalan.
- Queue DB writable di path baru.
- Viewer socket `/run/nms-agent/view.sock` bisa diakses.

### Risiko

- `MemoryMax` terlalu ketat bisa OOM kill.
  Mitigasi: pantau memory baseline dulu, atur sesuai kebutuhan.
- `ProtectSystem=strict` blokir write path. Mitigasi: verifikasi
  `ReadWritePaths` lengkap.

---

## Phase 8: Security Hardening

**Tujuan:** Mengurangi risiko kebocoran credential dan akses ilegal.

### Scope

Item di section ini adalah target-state recommendation kecuali dokumen
operasional current-state menyebutkan bahwa perilaku tersebut sudah diterapkan.

| Area | Tindakan |
|------|----------|
| Config secret | Audit sumber secret. Jangan simpan di YAML. Wajib env. |
| File permission | `0600` untuk `nms-agent.env`, `0640` untuk YAML config. |
| MQTT TLS | Validasi CA chain. Tolak `tcp://` tanpa TLS di production. |
| SNMP v3 | Prioritaskan SNMPv3. Beri warning keras untuk community `public`. |
| Unix socket | Batasi akses user/group ke `/run/nms-agent/view.sock`. |
| Threat model | Dokumentasi ringkas: credential leak, queue data leak, config tamper. |

### Strategi

- Mulai dari **warning dan dokumentasi**.
- Enforcement keras (fail/block) belakangan, pakai opt-in flag.

### Verifikasi

- Deploy lama masih bisa jalan (backward-compatible).
- Warning muncul jelas saat config insecure terdeteksi.
- Config docs mencantumkan praktik aman.

### Risiko

- Enforcement terlalu cepat bisa merusak site lama.
  Mitigasi: warning first, enforce later.

---

## Phase 9: Load, Soak & Rollout

**Tujuan:** Bukti stabil di kondisi nyata sebelum deployment luas.

### Test wajib

| Test | Minimum |
|------|---------|
| Broker down 30 menit | Queue tidak drop. Drain pulih otomatis. |
| Broker down 6 jam | Disk usage terkendali. |
| Restart saat backlog | Queue persisten. Tidak ada duplikat massal. |
| Reload saat traffic | Siklus saat ini selesai. Pipeline baru lanjut. |
| Soak 24-72 jam | Memory & CPU stabil. |
| Scale test | N device × M metrics sesuai target site. |

### Output

- Kapasitas resmi terdokumentasi:
  - Maksimum device per site.
  - Rata-rata metric rate.
  - Rekomendasi disk minimum.
  - Rekomendasi `poll_interval` minimum.

---

## Urutan Implementasi Paling Aman

```
Phase 0  ─►  Phase 1  ─►  Phase 2  ─►  Phase 3  ─►  Phase 7
                                                    │
                                                    ▼
                                              Phase 4 (4a → 4b → 4c)
                                                    │
                                                    ▼
                                              Phase 5
                                                    │
                                                    ▼
                                              Phase 6
                                                    │
                                                    ▼
                                              Phase 8
                                                    │
                                                    ▼
                                              Phase 9
```

**Alasan:**
- Observability dulu agar setiap perubahan selanjutnya bisa dipantau.
- Quality gate sebelum perubahan sensitif.
- Queue semantics (retry, dead-letter) paling riskan, dikerjakan setelah
  visibility dan safety net siap.
- Security hardening dan load test di akhir karena paling sedikit dampak
  pada code flow.

---

## Acceptance Criteria Ringkas

| Fase | Kriteria |
|------|----------|
| 1 | Log cukup untuk jawab: "Apakah cycle terakhir sukses? Berapa item enqueue? Berapa item gagal kirim?" |
| 2 | Operator bisa cek runtime state tanpa baca log mentah. |
| 3 | CI bisa mendeteksi regression umum (race, vet, test gagal). |
| 4 | Adapter outage tidak spam retry terus-menerus tanpa jeda. |
| 5 | Queue tidak tumbuh tak terbatas karena item rusak permanen. |
| 6 | Device error lokal tidak merusak site-wide collection. |
| 7 | Service survive restart/reload dengan proteksi resource. |
| 8 | Secret dan access surface lebih aman dari baseline. |
| 9 | Ada bukti soak & load, bukan asumsi. |

---

## Tradeoff yang Perlu Diingat

- Backoff = stabilitas broker, tapi tambah latency drain.
- Dead-letter = queue aman, tapi data "hilang" tanpa alert.
- HTTP health endpoint = integrasi mudah, tapi tambah attack surface.
- JSON log = parsing enak, tapi `journalctl` manual jadi kurang nyaman.

---

## Referensi File

| File | Kaitan |
|------|--------|
| `docs/QUEUE_DESIGN.md` | Queue data model, retry policy |
| `docs/ADAPTER_CONTRACT.md` | Adapter interface termasuk HealthChecker |
| `internal/queue/sqlite_queue.go` | SQLite queue implementasi |
| `internal/core/pipeline.go` | Pipeline flow |
| `cmd/nms-agent/main.go` | Entrypoint + loop |
| `cmd/nms-agentctl/` | CLI commands |
| `packaging/systemd/nms-agent.service` | systemd unit |
| `docs/CONFIG_SCHEMA.md` | Konfigurasi YAML |
