# NMS Agent

A platform-agnostic Go monitoring agent that collects device data via SNMP and ICMP, preprocesses and normalizes it, stores it in a local SQLite queue for store-and-forward, and sends it to consumer platforms (ThingsBoard, Generic MQTT, TUI) via adapters.

## Quick Start

```bash
# Validate configuration
nms-agentctl validate --config configs/agent.yml

# Run agent (uses dummy collector when no real targets enabled)
nms-agent run --config configs/agent.yml --collector-mode dummy

# Run agent with real SNMP/ICMP collectors
nms-agent run --config configs/agent.yml --collector-mode real

# Check queue status
nms-agentctl queue status --config configs/agent.yml

# Retry pending queue items
nms-agentctl queue retry --config configs/agent.yml
```

## Installation

### From Source

```bash
git clone https://github.com/prabutama/nms-agent
cd nms-agent
go build -o nms-agent ./cmd/nms-agent
go build -o nms-agentctl ./cmd/nms-agentctl
```

### systemd Deployment

```bash
sudo ./packaging/systemd/install.sh
sudo systemctl status nms-agent
sudo systemctl reload nms-agent
```

**Reinstall / Upgrade:**
Config files (`agent.yml`, `adapters.yml`, `thresholds.yml`) are preserved. Updated sample configs are deployed as `*.dist` files for reference.

## System Requirements

- Linux server recommended for production deployment
- Go 1.24.x for build from source
- systemd for service-based deployment
- Network access to:
  - target devices via ICMP and SNMP (UDP/161)
  - MQTT / ThingsBoard broker
- Writable local storage for SQLite queue
- WSL on Windows acceptable for development and testing

## Required Packages

Ubuntu/Debian:

```bash
sudo apt update
sudo apt install -y golang-go git iputils-ping ca-certificates
```

For systemd deployment:

```bash
sudo apt install -y systemd
```

Optional troubleshooting packages:

```bash
sudo apt install -y snmp mosquitto-clients sqlite3
```

## Configuration

### agent.yml

```yaml
agent:
  poll_interval: 60s
  output:
    timezone: UTC+7
  delivery:
    max_batch: 200
    drain_enabled: true
    max_batches_per_cycle: 20
    stop_on_error: true

paths:
  devices_dir: devices.d
  thresholds_file: thresholds.yml
  profiles_dir: profiles
  adapters_file: adapters.yml
  nms_agent_db: data/nms-agent.db
  view_socket: data/view.sock
```

### devices.d/<id>.yml

```yaml
id: device-1
address: 10.0.0.1
vendor: linux
model: proxmox
icmp:
  enabled: true
snmp:
  enabled: true
```

### adapters.yml

```yaml
adapters:
  active: thingsboard_mqtt
  configs:
    broker: tcp://thingsboard.local:1883
    qos: 1
    retain: false
    strict_queue_mode: true
    provisioning:
      base_url: http://thingsboard.local:8080
      device_key: ${TB_PROVISION_DEVICE_KEY}
      device_secret: ${TB_PROVISION_DEVICE_SECRET}
```

### thresholds.yml

```yaml
thresholds:
  - metric: snmp.if.rx_utilization_pct
    operator: ">"
    warning: 70
    critical: 90
    tags:
      ifIndex: "*"
```

## CLI Commands

### nms-agent

```bash
nms-agent run --config configs/agent.yml --collector-mode dummy
```

### nms-agentctl

```bash
# Validate config
nms-agentctl validate --config configs/agent.yml

# Reload agent
nms-agentctl reload --config configs/agent.yml --pid <pid>

# Device management
nms-agentctl device list --config configs/agent.yml
nms-agentctl device add --config configs/agent.yml --id <id> --address <host> --vendor <v> --model <m>
nms-agentctl device update --config configs/agent.yml --id <id> [--address <host>] [--vendor <v>]
nms-agentctl device remove --config configs/agent.yml --id <id>
nms-agentctl device test --config configs/agent.yml --id <id> --snmp=true --icmp=true

# Queue management
nms-agentctl queue status --config configs/agent.yml
nms-agentctl queue retry --config configs/agent.yml [--limit 100]

# Threshold management
nms-agentctl threshold list --config configs/agent.yml
nms-agentctl threshold set --config configs/agent.yml --metric <name> --operator <op> --warning <val> --critical <val> --tags k=v

# Adapter health check
nms-agentctl adapter health --config configs/agent.yml

# Discovery commands
nms-agentctl discovery status --config configs/agent.yml
nms-agentctl discovery preview --config configs/agent.yml
nms-agentctl discovery run --config configs/agent.yml
```

Notes:

- Always pass `--config` explicitly. Current subcommands do not all use same built-in default path.
- `nms-agentctl queue retry` is currently limited. For empty or `tui` active adapter it uses no-op delivery; real adapter redelivery is not implemented for other adapters yet.
- `nms-agentctl view` uses `paths.view_socket` when set; production defaults to `/run/nms-agent/view.sock`, while WSL/dev configs can use `data/view.sock`.

## Architecture

```
SNMP / ICMP Collector
  -> Device Profile / Vendor Profile
  -> Preprocessing Engine
  -> Canonical Telemetry Format
  -> SQLite Local Queue
  -> Output Adapter (terminal / TUI / Generic MQTT / ThingsBoard MQTT)
  -> Consumer Platform
```

## Demo Guide

1. **Validate config:**
   ```bash
   nms-agentctl validate --config configs/agent.yml
   ```

2. **Run with dummy collector (no real devices needed):**
   ```bash
   nms-agent run --config configs/agent.yml --collector-mode dummy
   ```

3. **Run with TUI adapter for live dashboard:**
   - Set `adapters.yml` to `active: tui`
   - Run: `nms-agent run --config configs/agent.yml --collector-mode dummy`
   - Use arrow keys to navigate, `q` to quit

4. **Run with real devices:**
   - Add device configs to `devices.d/`
   - Set `adapters.yml` to your preferred adapter
   - Run: `nms-agent run --config configs/agent.yml --collector-mode real`

## Troubleshooting

See [docs/TROUBLESHOOTING.md](docs/TROUBLESHOOTING.md) for common issues and fixes.

## Security

See [docs/SECURITY.md](docs/SECURITY.md) for security guidelines.

## Known Limitations

- `nms-agentctl queue retry` does not yet perform real redelivery for non-`tui` adapters. Treat it as limited diagnostic/development helper, not full production replay mechanism.
- `nms-agentctl view` requires a running daemon with viewer socket available. Use `paths.view_socket` for WSL/dev paths or default `/run/nms-agent/view.sock` for production Linux/systemd.
- Default `--config` behavior is not yet uniform across all `nms-agentctl` subcommands. Use `--config <path>` on every command to avoid ambiguity.

## Platform Notes

- Linux with systemd is primary production target.
- Windows and WSL are acceptable for development, configuration work, and most tests.
- Local viewer socket features such as `nms-agentctl view` require Unix sockets; configure `paths.view_socket` for WSL/dev runs.

## License

Private / Internal Use Only
