# Configuration Schema

The configuration is YAML-based and split into multiple files.

## File Layout

- `configs/agent.yml`: main entrypoint config.
- `configs/devices.d/*.yml`: device inventory entries.
- `configs/thresholds.yml`: active threshold rules consumed by runtime preprocessing.
- `configs/adapters.yml`: active adapter selection/config consumed by runtime delivery pipeline.

This document describes current configuration behavior, not phase-history snapshot.

## configs/agent.yml

```yaml
agent:
  env_file: local.env
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
  nms_agent_db: data/nms-agent.db
  view_socket: data/view.sock
```

Notes:
- `poll_interval` uses Go duration format (e.g. `10s`, `1m`).
- `agent.env_file` is optional. When set, it loads `KEY=value` pairs relative to `agent.yml` before `${ENV_VAR}` expansion. Existing process environment variables take precedence over values in the file.
- `agent.output.timezone` controls presentation timezone for adapter output (TUI/MQTT). Default is `UTC`.
  Supported values: IANA (e.g. `Asia/Jakarta`) or fixed offsets like `UTC+7`, `UTC+07:00`.
- `delivery.*` configures the queue delivery drain loop (Phase 8):
  - `max_batch`: max items fetched per batch (default 100).
  - `drain_enabled`: when true, keep delivering until queue empty or max_batches_per_cycle reached.
  - `max_batches_per_cycle`: max delivery rounds per single poll cycle (default 10).
  - `stop_on_error`: when true, abort on first send failure; otherwise continue draining.
- `paths.*` may be relative to the directory containing `agent.yml`.
- `${ENV_VAR}` expansion is supported for path strings via the current process environment.
- `paths.nms_agent_db` is the SQLite DB file path for local agent state. It stores queue items and generated ThingsBoard device tokens. Its parent directory is created at runtime if missing.
- `paths.view_socket` is the Unix socket path used by `nms-agentctl view`. If empty, it defaults to `/run/nms-agent/view.sock`. Use a relative path like `data/view.sock` for WSL/dev runs without systemd permissions.
- Discovery is manual-only. `agent.yml` does not require or store discovery configuration for daemon runtime.
- Discovery is executed only through `nms-agentctl discovery preview --subnet <CIDR>` and `nms-agentctl discovery run --subnet <CIDR>`.
- Running preview/run is explicit consent; manual discovery does not depend on `discovery.enabled`.
- The CLI auto-detects the local interface when `--interface` is omitted. If auto-detection fails, pass `--interface` explicitly.
- `--provider` supports:
  - `netlink`: passive Linux neighbor/ARP discovery. Host biasanya baru terlihat setelah ada traffic/ARP entry.
  - `active`: active ICMP probe ke seluruh host dalam subnet agar candidate baru bisa muncul tanpa ping manual terlebih dahulu.
- Active probe defaults: `timeout=1s`, `concurrency=64`.
- SNMP probe defaults: `version=v2c`, `community=public` when not provided, `timeout=2s`, `retries=1`, `concurrency=4`.
- `--snmp-community` supports `${ENV_VAR}` expansion.
- Manual discovery always requires SNMP OK, `sysObjectID`, and a known profile match before promotion.
- SNMP probe errors are reported as `SNMP_PROBE_FAILED` and never promote a device.
- `--max-new-devices`: `0` or omitted uses default `50`, positive values set an explicit limit, `-1` means unlimited.
- If SNMP community is missing, CLI prints a warning and continues with the default public community.
- Unknown profile matches are skipped with a warning and do not write `devices.d`.
- Known profile matches are promoted to `devices.d` with `0600` file permission.
- On Linux/systemd installs, discovery CLI may run as `root`; promoted device/profile artifacts are re-owned to service user `nms-agent` before rename so daemon can read them.
- Perubahan file `devices.d/*.yml` sekarang dipantau daemon; jika file device ditambah/diubah/dihapus dan config valid, runtime akan reload otomatis tanpa restart service.
- Current CLI subcommands do not all use same built-in default `--config` path. For consistent behavior, pass `--config <path>` explicitly.

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
- Route inventory MVP does not require extra config in `agent.yml`. Every SNMP-enabled device is automatically probed for IPv4 routes using `ipCidrRouteTable` as primary source, `ipRouteTable` as fallback, and `inetCidrRouteTable` as best-effort optional source.

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

This file is active in current runtime. Threshold rules are loaded and applied by preprocessing before adapter delivery.

## configs/adapters.yml

```yaml
adapters:
  active: tui
  configs: {}
```

This file is active in current runtime. Adapter selection and adapter config are loaded into pipeline startup and reload paths.

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

- `thingsboard_mqtt`: auto-provision ThingsBoard device tokens, cache them in SQLite, then publish telemetry through the ThingsBoard MQTT Device API.
  Required configs:
  - `broker`: broker URL (e.g. `tcp://thingsboard.local:1883`).
  - `provisioning.base_url`: ThingsBoard HTTP base URL for `POST /api/v1/provision`.
  - `provisioning.device_key`: provisioning key from the target ThingsBoard device profile.
  - `provisioning.device_secret`: provisioning secret from the target ThingsBoard device profile.
  Optional configs:
  - `client_id`: MQTT client ID.
  - `qos`: `0|1|2` (default `1`).
  - `retain`: (default `false`).
  - `auto_reconnect`: (default `true`).
  - `strict_queue_mode`: fail-fast on disconnect so SQLite pending reflects broker outages (default `false`).
  - `connect_timeout`: Go duration (default `5s`).
  - `publish_timeout`: Go duration (default `5s`).
  - `thingsboard.api.base_url`: base URL REST ThingsBoard untuk jalur manajemen hybrid.
  - `thingsboard.api.api_key`: API key REST tenant-scope untuk device lookup, relation, alarm, dan topology.
  - `thingsboard.site.key`: logical site key internal agent.
  - `thingsboard.site.asset_id`: target asset site untuk relation dan topology attributes.
  - `thingsboard.site.asset_name`: optional nama asset untuk audit/debug.
  Notes:
  - Adapter publishes per-device timestamped payloads to fixed topic `v1/devices/me/telemetry`.
  - Adapter publishes per-device attributes to fixed topic `v1/devices/me/attributes`.
  - Device tokens are stored in `thingsboard_device_tokens` inside `paths.nms_agent_db`.
  - Adapter will publish metric value plus metadata keys: `<metric>__value_type` and `<metric>__tags` (includes threshold tags like `threshold.status`).
  - Jalur hybrid management memakai REST API tenant-scope untuk relation `ASSET(site) --Contains--> DEVICE`, publish topology snapshot ke `SERVER_SCOPE` attribute asset site, serta alarm create/clear/assign. Gagal pada jalur management tidak boleh mematikan telemetry utama.

Example:

```yaml
adapters:
  active: thingsboard_mqtt
  configs:
    broker: tcp://127.0.0.1:1883
    strict_queue_mode: true
    provisioning:
      base_url: ${TB_URL}
      device_key: ${TB_PROVISION_DEVICE_KEY}
      device_secret: ${TB_PROVISION_DEVICE_SECRET}
    thingsboard:
      api:
        base_url: ${TB_URL}
        api_key: ${TB_API_KEY}
      site:
        key: branch-b
        asset_id: a75abc20-1839-11f1-a070-473c29007b79
        asset_name: Branch-B
```

## .env usage

The binary expands `${ENV_VAR}` from the current process environment and from optional `agent.env_file`.

Example `configs/local.env`:

```bash
TB_URL=https://thingsboard.example.com
TB_PROVISION_DEVICE_KEY=replace-me
TB_PROVISION_DEVICE_SECRET=replace-me
```

This expansion applies to:

- config paths in `agent.yml`
- string values inside `adapters.yml` configs

For service deployment, prefer setting environment via systemd `EnvironmentFile` (for example `/etc/nms-agent/nms-agent.env`).
