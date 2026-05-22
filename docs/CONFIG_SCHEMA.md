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

paths:
  devices_dir: devices.d
  thresholds_file: thresholds.yml
  adapters_file: adapters.yml
```

Notes:
- `poll_interval` uses Go duration format (e.g. `10s`, `1m`).
- `paths.*` may be relative to the directory containing `agent.yml`.
- `${ENV_VAR}` expansion is supported for path strings via the current process environment.

## configs/devices.d/*.yml

```yaml
id: router-1
address: 192.0.2.1
vendor: example
model: example-router
```

Notes:
- `id` must be unique.
- `address` is a host/IP string (protocol-specific validation is deferred).

## configs/thresholds.yml (placeholder)

```yaml
thresholds: []
```

## configs/adapters.yml (placeholder)

```yaml
adapters:
  active: terminal
  configs: {}
```

## .env usage

Phase 2 does not load `.env` files yet. Only `${ENV_VAR}` expansion is supported for path strings.
