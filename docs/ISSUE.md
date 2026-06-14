# Known Issues

## Auto Discovery

### 1. SNMP v2c only, tanpa auth

`internal/discovery/snmp_probe.go:26-28` hardcode `Version2c` dengan community "public" fallback. Tidak ada dukungan SNMP v3 (username, auth protocol, encryption). Device dengan SNMP v3 strict tidak terdeteksi, community string dikirim plaintext.

### 2. Command injection via config

`internal/discovery/providers/active/provider.go:153` — `exec.CommandContext(ctx, "ping", pingArgs(address, timeout)...)`. Parameter `address` dan `timeout` berasal dari config/subnet scan. `loaded.Root.Discovery.Interface` dan `Subnet` langsung dari YAML tanpa sanitasi. Jika attacker bisa modifikasi config → arbitrary command execution.

### 3. Device YAML world-readable

`internal/discovery/promote.go:73` — `os.WriteFile(tmpPath, b, 0o644)`. File device berisi IP, vendor, model — readable oleh semua user di sistem.

### 4. Network scanning tanpa consent

Active provider melakukan ICMP sweep ke seluruh subnet. Bisa dianggap port scan / hostile activity oleh IDS/IPS.

### 5. No granular rate limiting

`probeCandidates` parallel workers melakukan N koneksi SNMP simultan ke N IP berbeda. Bisa membanjiri network kecil.

### 6. Candidate list in-memory penuh

`internal/discovery/providers/active/provider.go:141` — `hostsInSubnet` generate semua IP sekaligus dalam `[]string`. Subnet /16 = 65.534 entry dalam slice. Tidak ada streaming atau chunking.

### 7. O(n*m) profile matching

`internal/profiles/validate.go:85` — `SelectProfile` iterate linear atas semua profile untuk setiap fingerprint. 1000 candidates × 50 profiles = 50.000 iterasi per cycle.

### 8. SNMP connection per-probe tanpa pool

`internal/discovery/snmp_probe.go:51` — `client.Connect()` + Get + Close untuk setiap candidate. Tidak ada connection reuse atau persistent session. Overhead TCP handshake untuk ribuan device.

### 9. Reload full config tiap discovery change

`cmd/nms-agent/main.go:325` — setelah promote, reload entire config (YAML parsing + validate) untuk satu device baru. Dengan 10.000 device existing, O(n) full parse setiap kali.

### 10. Duplicate profile loading

`internal/discovery/service.go:52-57` — load profiles sekali, lalu setelah exploration (line 110-114) load lagi. Dua kali I/O dalam satu run.

### 11. Filter existing by brute force

`internal/discovery/service.go:47-49` — iterate `loaded.Devices` + `strings.ToLower` tiap candidate. O(d × c) per cycle. Tidak ada hash map untuk lookup address.

### 12. No incremental discovery

Setiap cycle re-scan seluruh subnet dari nol. Tidak ada cache ARP/neighbor atau diff-based scanning.

### 13. No retry on SNMP probe failure

`snmp_probe.go:59-62` — jika SNMP Get gagal, `fp` dikembalikan apa adanya (SNMPOK=false). Tidak ada retry. Network glitch transient menyebabkan false negative.

### 14. Partial exploration tidak di-rollback

Jika exploration berhasil generate profile tapi `writePromotedDevice` gagal (disk full, permission), profile YAML sudah tertulis tapi device tidak dipromote. State inconsistent.

### 15. Devices watcher race condition

`configwatch.DevicesWatcher` trigger reload saat file berubah. Discovery juga write file lalu trigger watcher. Ada potensi reload bersamaan antara perubahan dari discovery dan perubahan manual user.
