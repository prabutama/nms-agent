# Soak & Load Test Plan

> Dokumen ini berisi rencana pengujian untuk memvalidasi stabilitas agent
> dalam kondisi operasi nyata sebelum deployment luas.

---

## Prasyarat

Sebelum menjalankan pengujian, pastikan:

- [ ] Agent versi terbaru sudah di-build dan di-instal di host pengujian.
- [ ] `nms-agentctl validate` lulus tanpa error.
- [ ] `nms-agentctl status` menunjukkan agent berjalan normal.
- [ ] Broker/destination tersedia dan bisa di-reach.
- [ ] Disk tersedia minimal 5GB untuk queue data.
- [ ] Monitor CPU dan memory diaktifkan (htop, journalctl, atau alat vendor).

---

## Test: Baseline (10 menit)

Tujuan: Pastikan agent berjalan normal tanpa beban khusus.

Langkah:
1. Start agent: `nms-agent run --config /etc/nms-agent/agent.yml --collector-mode real`
2. Biarkan berjalan 10 menit.
3. Pantau:
   - Log: tidak ada error berulang.
   - Status: `nms-agentctl status --watch` — cycle ok berturut-turut.
   - Queue: pending `0` setelah beberapa siklus.

Kriteria lulus:
- Semua cycle OK.
- Queue pending `0` di akhir.
- Tidak ada restart tidak terduga.

---

## Test: Broker Down 30 Menit

Tujuan: Validasi store-and-forward bekerja saat adapter tidak reachable.

Langkah:
1. Agent berjalan normal dengan MQTT adapter.
2. Matikan broker (stop MQTT service / blokir port).
3. Tunggu 30 menit.
4. Hidupkan kembali broker.
5. Pantau:
   - Queue pending bertambah selama broker down.
   - Setelah broker pulih, pending menurun ke 0.
   - Setiap item tetap di-queue (tidak hilang).
   - Retry_count bertambah.

Kriteria lulus:
- Queue pending kembali ke 0 dalam waktu ≤ 5 menit setelah broker pulih.
- Tidak ada telemetry yang hilang (diverifikasi dari jumlah item yang diterima broker).
- Log menunjukkan `delivery_ok` setelah broker pulih.

---

## Test: Restart Saat Queue Backlog

Tujuan: Validasi queue persistence saat daemon restart dengan item pending.

Langkah:
1. Broker dimatikan.
2. Tunggu hingga queue memiliki minimal 100 pending items.
3. Restart agent: `systemctl restart nms-agent`
4. Nyalakan broker.
5. Pantau:
   - Queue pending tetap ada setelah restart.
   - Items dikirim setelah broker pulih.
   - Tidak ada duplikasi massal.

Kriteria lulus:
- Jumlah pending sebelum restart = jumlah yang terkirim + dead-letter (jika ada).
- Log menunjukkan `cycle_start` dan `cycle_end` normal setelah restart.

---

## Test: Reload Config Saat Traffic Aktif

Tujuan: Validasi hot reload tidak mengganggu siklus yang sedang berjalan.

Langkah:
1. Agent berjalan dengan beberapa device.
2. Edit `devices.d/device.yml` (contoh: ubah interval atau collector flag).
3. Kirim SIGHUP: `nms-agentctl reload --pid $(pidof nms-agent)` atau `systemctl reload nms-agent`.
4. Pantau:
   - Log: `reload_completed` atau `reload_failed`.
   - Cycle berlanjut normal setelah reload.
   - Queue tidak kehilangan items saat reload.

Kriteria lulus:
- Reload sukses.
- Cycle berikutnya berjalan dengan config baru.
- Tidak ada error di log terkait reload.

---

## Test: Soak 24 Jam

Tujuan: Validasi stabilitas jangka panjang.

Langkah:
1. Agent berjalan normal selama 24 jam dengan device dan broker real.
2. Pantau secara periodik (setiap 1 jam):
   - Memory usage (`MemoryMax` di systemd = 256M).
   - CPU usage.
   - Queue pending (harus 0 atau mendekati 0).
   - Cycle success rate.
3. Catat:
   - Total cycle count.
   - Total items delivered.
   - Total failures.
   - Memory max, min, avg.
   - CPU max, min, avg.
   - Disk usage nms-agent.db.

Kriteria lulus:
- Agent tidak restart/crash selama 24 jam.
- Memory stabil (tidak growth terus).
- Queue pending `0` di akhir (asumsi broker selalu up).

---

## Test: Scale — Multi Device

Tujuan: Validasi performa dengan jumlah device realistic.

Langkah:
1. Konfigurasi 20+ device.
2. Jalankan agent 1 jam.
3. Catat:
   - Waktu per cycle.
   - Jumlah metrics per cycle.
   - Adapter delivery time.
   - Queue depth maksimum.

Kriteria lulus:
- Cycle time < 80% poll_interval.
- Tidak ada timeout/error massal.
- Semua device terwakili dalam telemetry.

---

## Test: Dead-Letter dan Retention

Tujuan: Validasi dead-letter dan cleanup policy.

Langkah:
1. Aktifkan retry dengan `max_retries: 3`.
2. Matikan broker permanen.
3. Tunggu hingga items mencapai `dead_letter`.
4. Pantau:
   - Status: `queue_dead_letter` bertambah.
   - Log: item dipindahkan ke `dead_letter`.
   - `queue status` menunjukkan dead_letter count.
5. Nyalakan broker dan verifikasi dead-letter items tidak dikirim ulang.
6. Verifikasi `CleanupDeleted` menghapus items setelah retention_days.

Kriteria lulus:
- Items dengan retry_count >= max_retries masuk `dead_letter`.
- Items dead-letter tidak di-PendingBatch.
- Items dead-letter dihapus setelah retention_days.

---

## Test: Failure Isolation Per Device

Tujuan: Validasi satu device error tidak menggagalkan cycle.

Langkah:
1. Konfigurasi 5 device, 1 device di antaranya unreachable.
2. Jalankan agent.
3. Pantau:
   - Device unreachable: log error per device (bukan fail global).
   - Device lain tetap menghasilkan telemetry.
   - Queue menerima data dari device yang reachable.
4. Perbaiki device unreachable.
5. Pantau semua device kembali normal.

Kriteria lulus:
- Cycle tidak gagal total.
- Queue berisi data dari device sehat.
- Log mencatat error per-device.

---

## Ringkasan Hasil

Setelah semua test selesai, catat:

| Test | Status | Catatan |
|------|--------|---------|
| Baseline | | |
| Broker Down 30m | | |
| Restart Backlog | | |
| Reload Traffic | | |
| Soak 24h | | |
| Scale 20+ device | | |
| Dead-letter | | |
| Failure Isolation | | |

## Rekomendasi Kapasitas

Berdasarkan hasil pengujian, isi:

- Maksimum device per site: ___
- Rata-rata metric rate: ___
- Disk minimum: ___
- Rekomendasi `poll_interval` minimum: ___
- Memory baseline: ___
