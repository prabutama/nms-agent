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
```

Notes:
- `poll_interval` uses Go duration format (e.g. `10s`, `1m`).
- `delivery.*` configures the queue delivery drain loop (Phase 8):
  - `max_batch`: max items fetched per batch (default 100).
  - `drain_enabled`: when true, keep delivering until queue empty or max_batches_per_cycle reached.
  - `max_batches_per_cycle`: max delivery rounds per single poll cycle (default 10).
  - `stop_on_error`: when true, abort on first send failure; otherwise continue draining.
- `paths.*` may be relative to the directory containing `agent.yml`.
- `${ENV_VAR}` expansion is supported for path strings via the current process environment.
- `paths.queue_db` is the SQLite DB file path for the local durable queue. Its parent directory is created at runtime if missing.

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
  active: terminal
  configs: {}
```

Supported adapter names:

- `terminal`: print telemetry as log lines to stdout.
- `tui`: interactive Bubbletea-based TUI with device health, alerts, and interface throughput (requires TTY).  
  Optional configs:
  - `refresh_interval`: TUI refresh rate (e.g. `1s`, default `1s`).
  - `alt_screen`: use terminal alternate screen buffer (default `true`).
  - `discard_output`: discard TUI output and disable input (headless/testing, default `false`).
  - `disable_renderer`: disable Bubble Tea renderer (advanced/testing, default `false`).

## .env usage

Phase 2 does not load `.env` files yet. Only `${ENV_VAR}` expansion is supported for path strings.
