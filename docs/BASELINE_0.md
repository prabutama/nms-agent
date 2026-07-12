# Phase 0 — Baseline & Safety Net

> Dokumentasi keadaan sekarang sebagai acuan sebelum perubahan production.
>
> Historical baseline snapshot only.
> Tidak dijamin cocok dengan layout package, versi Go, jumlah package, atau
> angka coverage saat ini. Untuk perilaku operasional saat ini, ikuti
> `README.md`, `docs/CLI_COMMANDS.md`, `docs/CONFIG_SCHEMA.md`, dan code.

## Snapshot Environment

| Item | Value |
|------|-------|
| Go version | 1.26.0 (snapshot) |
| OS | Windows (dev), Linux (deploy) |
| SQLite | `modernc.org/sqlite` (pure Go, CGO_ENABLED=0) |
| Build | `go build ./...` OK |

## Test Baseline Snapshot (22 paket)

Angka di bawah adalah snapshot baseline, bukan metrik live yang dijaga terus.

| Package | Status | Coverage |
|---------|--------|----------|
| `cmd/nms-agent` | [no test files] | 0.0% |
| `cmd/nms-agentctl` | ok | 42.7% |
| `internal/adapters` | ok | 41.7% |
| `internal/adapters/base` | [no test files] | 0.0% |
| `internal/adapters/generic_mqtt` | ok | 32.1% |
| `internal/adapters/terminal` | [no test files] | 0.0% |
| `internal/adapters/thingsboard_mqtt` | ok | 55.1% |
| `internal/adapters/tui` | [no test files] | 0.0% |
| `internal/collectors` | ok | 74.9% |
| `internal/config` | ok | 38.1% |
| `internal/configwatch` | ok | 12.7% |
| `internal/core` | ok | 69.8% |
| `internal/discovery` | ok | 68.0% |
| `internal/discovery/explorer` | ok | 24.5% |
| `internal/discovery/providers` | ok | 100.0% |
| `internal/discovery/providers/active` | ok | 18.8% |
| `internal/discovery/providers/netlink` | [no test files] | 0.0% |
| `internal/integrations/thingsboard` | ok | 26.0% |
| `internal/models` | [no test files] | 0.0% |
| `internal/processors` | ok | 88.0% |
| `internal/profiles` | ok | 78.5% |
| `internal/queue` | ok | 42.1% |
| `internal/routes` | ok | 62.6% |
| `internal/viewer` | ok | 31.4% |
| **Total** | **22 ok, 0 fail** | |

### Race Detector

- `go test -race ./...` tidak bisa jalan di Windows karena `modernc.org/sqlite`
  membutuhkan `CGO_ENABLED=0`.
- Race test wajib dijalankan di Linux (CI) sebelum deploy.
- Platform Windows hanya untuk development.

Tabel package di atas bisa memuat nama package yang berubah pada fase berikutnya.
Perlakukan sebagai bukti historis, bukan inventaris current-state.

## Code Stats Snapshot

| Metric | Value |
|--------|-------|
| Go files | 115 (snapshot) |
| Total bytes | ~420 KB (snapshot) |

## Invariants (Tidak Boleh Dilanggar)

1. Telemetry masuk queue **sebelum** adapter delivery.
2. Item gagal tetap `pending` (tidak hilang).
3. Item sukses di-`DELETE` dari queue.
4. `SIGHUP`/reload tidak membutuhkan restart process.
5. Queue tidak boleh kehilangan data saat restart daemon.
6. Config invalid tidak boleh menghentikan service yang sedang berjalan
   (reload gagal, runtime lama lanjut).

## Baseline Behavior

### Skema Normal

```
Poll ─► Collect ─► Normalize ─► Enqueue ─► PendingBatch ─► SendBatch
                                           └── sukses → MarkDelivered
                                           └── gagal  → MarkFailed
```

### Adapter Down

- `SendBatch` gagal → `MarkFailed` (retry_count+1, last_error diisi).
- Queue tetap `pending`. Siklus berikutnya dicoba lagi.
- Jika `stop_on_error=true`, siklus berhenti di error pertama.
- Jika `drain_enabled=true`, siklus lanjut ke batch berikutnya.

### Restart Daemon

- Queue SQLite persisten di disk.
- Item `pending` tetap ada setelah restart.
- Item `delivered` sudah di-`DELETE`, tidak dikirim ulang.

### Reload Config (SIGHUP)

- Pipeline dibangun ulang: collector, processor, adapter baru.
- Queue object (SQLite) tetap sama — tidak ditutup.
- Ticker interval bisa berubah.
- Watcher devices.d juga di-reinit jika path berubah.

### CLI Verifikasi

```bash
nms-agentctl validate            # config OK
nms-agentctl queue status        # pending + max_retry
nms-agentctl adapter health      # adapter OK/error
```

## Catatan untuk CI (Phase 3)

- Race test dan full staticcheck hanya bisa di Linux.
- Windows cukup `go test ./...` + `go vet ./...` + `go build ./...`.
