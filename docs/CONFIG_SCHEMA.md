# Phase 2 — Configuration Schema (MVP)

The configuration is YAML-based and split into multiple files.

## File Layout

- `configs/agent.yml`: main entrypoint config.
- `configs/devices.d/*.yml`: device inventory entries.
- `configs/thresholds.yml`: placeholder for Phase 7 (loaded/validated for structure only in Phase 2).
- `configs/adapters.yml`: placeholder for adapter selection/config (loaded/validated for structure only in Phase 2).

## configs/agent.yml

```yaml
agent:
  poll_interval: 60s

  # Presentation timezone for adapter output only.
  # Canonical telemetry timestamps are still stored as absolute instants.
  output:
    timezone: UTC

  delivery:
    max_batch: 200
    drain_enabled: true
    max_batches_per_cycle: 20
    stop_on_error: true

paths:
  devices_dir: devices.d
  thresholds_file: thresholds.yml
  adapters_file: adapters.yml
  queue_db: data/queue/queue.db

discovery:
  enabled: false
  interval: 10m
  interface: eth0
  subnet: 192.168.10.0/24
  provider: netlink

  active_probe:
    timeout: 1s
    concurrency: 64

  snmp:
    version: v2c
    community: ${SNMP_COMMUNITY}
    timeout: 2s
    retries: 1
    concurrency: 32

  auto_promote:
    enabled: true
    require_snmp_ok: true
    require_sys_object_id: true
    require_profile_match: true
    max_new_devices_per_cycle: 10
    device_id_template: "{{vendor}}-{{sys_name}}"
    write_to: devices.d

  exploration:
    enabled: true
    run_when: no_profile_match
    safe_only: true
    auto_approve_generated_profile: true
    auto_promote_after_generate: true
    max_oids_per_device: 300
    timeout: 3s
    output_dir: profiles
```

Notes:
- `poll_interval` uses Go duration format (e.g. `10s`, `1m`).
- `agent.output.timezone` controls presentation timezone for adapter output (TUI/MQTT). Default is `UTC`.
  Supported values: IANA (e.g. `Asia/Jakarta`) or fixed offsets like `UTC+7`, `UTC+07:00`.
- `delivery.*` configures the queue delivery drain loop (Phase 8):
  - `max_batch`: max items fetched per batch (default 100).
  - `drain_enabled`: when true, keep delivering until queue empty or max_batches_per_cycle reached.
  - `max_batches_per_cycle`: max delivery rounds per single poll cycle (default 10).
  - `stop_on_error`: when true, abort on first send failure; otherwise continue draining.
- `paths.*` may be relative to the directory containing `agent.yml`.
- `${ENV_VAR}` expansion is supported for path strings via the current process environment.
- `paths.queue_db` is the SQLite DB file path for the local durable queue. Its parent directory is created at runtime if missing.
- `discovery.*` is optional. Discovery supports passive `netlink` candidates and active ICMP subnet probing via `active`.
- `discovery.interface` is the local network interface name on the gateway host (for example `eth0`, `ens18`, `br-lan`).
- `discovery.subnet` filters candidates to a single target CIDR.
- `discovery.provider` supports:
  - `netlink`: passive Linux neighbor/ARP discovery. Host biasanya baru terlihat setelah ada traffic/ARP entry.
  - `active`: active ICMP probe ke seluruh host dalam subnet agar candidate baru bisa muncul tanpa ping manual terlebih dahulu.
- `discovery.active_probe.timeout` membatasi satu ICMP probe per host (default `1s`).
- `discovery.active_probe.concurrency` mengatur jumlah probe ICMP paralel (default `64`).
- `discovery.snmp.community` supports `${ENV_VAR}` expansion through the current process environment.
- `discovery.exploration.*` is active in Milestone B using a static safe OID catalog (system, interfaces, host-resources). It is not a full arbitrary-tree SNMP walk.
- Perubahan file `devices.d/*.yml` sekarang dipantau daemon; jika file device ditambah/diubah/dihapus dan config valid, runtime akan reload otomatis tanpa restart service.

## configs/devices.d/*.yml

```yaml
id: router-1
address: 192.0.2.1
vendor: mikrotik
model: routeros

icmp:
  enabled: true
snmp:
  enabled: true
```

Notes:
- `id` must be unique.
- `address` is a host/IP string (protocol-specific validation is deferred).
- `icmp.enabled` and `snmp.enabled` toggle which real collectors are used in `--collector-mode auto|real`.

## configs/thresholds.yml

```yaml
thresholds:
  - metric: icmp.latency_ms
    operator: ">"
    warning: 50
    critical: 100
    tags:
      source: icmp
```

## configs/adapters.yml

```yaml
adapters:
  active: tui
  configs: {}
```

Supported adapter names:

- `tui`: interactive Bubbletea-based TUI with device health, alerts, and interface throughput (requires TTY).  
  Optional configs:
  - `refresh_interval`: TUI refresh rate (e.g. `1s`, default `1s`).
  - `alt_screen`: use terminal alternate screen buffer (default `true`).
  - `discard_output`: discard TUI output and disable input (headless/testing, default `false`).
  - `disable_renderer`: disable Bubble Tea renderer (advanced/testing, default `false`).

- `generic_mqtt`: publish canonical telemetry JSON to an MQTT broker.
  Required configs:
  - `broker`: broker URL or host:port (e.g. `tcp://127.0.0.1:1883` or `127.0.0.1:1883`).
  - `topic`: publish topic (e.g. `nms-agent/telemetry`).
  Optional configs:
  - `client_id`: MQTT client ID.
  - `username`, `password`: broker auth.
  - `qos`: `0|1|2` (default `1`).
  - `retain`: retained publish flag (default `false`).
  - `auto_reconnect`: enable auto reconnect (default `true`).
  - `strict_queue_mode`: when true, fail-fast on disconnect so SQLite pending reflects broker outages (default `false`).
  - `connect_timeout`: Go duration (default `5s`).
  - `publish_timeout`: Go duration (default `5s`).

Example:

```yaml
adapters:
  active: generic_mqtt
  configs:
    broker: tcp://127.0.0.1:1883
    topic: nms-agent/telemetry
    qos: 1
    retain: false
    strict_queue_mode: true
```

- `thingsboard_mqtt`: publish ThingsBoard-shaped telemetry either directly to ThingsBoard Gateway MQTT API or indirectly via broker for ThingsBoard Gateway connector ingestion.
  Required configs:
  - `broker`: broker URL (e.g. `tcp://thingsboard.local:1883`).
  Optional configs:
  - `mode`: `direct` or `gateway` (default `direct`).
    - `direct`: MQTT auth uses ThingsBoard gateway `access_token`, default topic `v1/gateway/telemetry`.
    - `gateway`: publish to a regular broker/topic for ThingsBoard Gateway connector consumption, default topic `nms-agent/thingsboard/telemetry`.
  - `access_token`: ThingsBoard gateway device access token (required in `direct` mode).
  - `username`: broker username for `gateway` mode if broker auth is enabled.
  - `password`: broker password for `gateway` mode if broker auth is enabled.
  - `topic`: telemetry topic (default `v1/gateway/telemetry`).
  - `client_id`: MQTT client ID.
  - `qos`: `0|1|2` (default `1`).
  - `retain`: (default `false`).
  - `auto_reconnect`: (default `true`).
  - `strict_queue_mode`: fail-fast on disconnect so SQLite pending reflects broker outages (default `false`).
  - `connect_timeout`: Go duration (default `5s`).
  - `publish_timeout`: Go duration (default `5s`).
  Notes:
  - Adapter publishes payload grouped by device and timestamp in ThingsBoard-style `{"device":[{"ts":...,"values":{...}}]}` format.
  - Adapter will publish metric value plus metadata keys: `<metric>__value_type` and `<metric>__tags` (includes threshold tags like `threshold.status`).
  - In `gateway` mode this payload is intended to be consumed by a ThingsBoard Gateway MQTT connector with a thin/pass-through custom converter.

Example:

```yaml
adapters:
  active: thingsboard_mqtt
  configs:
    broker: tcp://127.0.0.1:1883
    access_token: YOUR_GATEWAY_TOKEN
    strict_queue_mode: true
```

Gateway mode example:

```yaml
adapters:
  active: thingsboard_mqtt
  configs:
    broker: tcp://127.0.0.1:1883
    mode: gateway
    topic: nms-agent/thingsboard/telemetry
    strict_queue_mode: true
```

## .env usage

Phase 2 does not load `.env` files yet. Only `${ENV_VAR}` expansion is supported for path strings.
