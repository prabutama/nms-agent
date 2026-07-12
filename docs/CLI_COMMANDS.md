# NMS Agent CLI Command Examples

Use `--config <path>` on every command. Current subcommands do not all share same built-in default config path.

## Current Default Config Paths

- `nms-agentctl status` -> `/etc/nms-agent/agent.yml`
- `nms-agentctl adapter health` -> `/etc/nms-agent/agent.yml`
- `nms-agentctl discovery ...` -> `/etc/nms-agent/agent.yml`
- `nms-agentctl view` -> `/etc/nms-agent/agent.yml`
- `nms-agentctl queue retry` -> `configs/agent.yml`

## Validate Config
nms-agentctl validate --config configs/agent.yml

## Reload Agent

Send SIGHUP to a running `nms-agent` process after validating config:

nms-agentctl reload --config configs/agent.yml --pid <pid>

## Device Commands
nms-agentctl device list --config configs/agent.yml

### Add a device
nms-agentctl device add --config configs/agent.yml \
  --id router-2 \
  --address 192.0.2.2 \
  --vendor mikrotik \
  --model routeros \
  --snmp=true \
  --icmp=true

### Update a device
nms-agentctl device update --config configs/agent.yml \
  --id router-2 \
  --address 192.0.2.22 \
  --icmp=false

### Remove a device
nms-agentctl device remove --config configs/agent.yml --id router-2

### Test a device (SNMP/ICMP)
nms-agentctl device test --config configs/agent.yml --id router-2

## Queue Commands
nms-agentctl queue status --config configs/agent.yml
nms-agentctl queue retry --config configs/agent.yml --limit 100

Notes:

- `queue status` reads queue metadata from configured SQLite path.
- `queue retry` is currently limited. When active adapter is empty or `tui`, it uses no-op delivery path. For other adapters, real redelivery is not implemented in this CLI yet.
- Treat `queue retry` as diagnostic/development helper until real adapter delivery support is added.

## Adapter Commands
nms-agentctl adapter health --config configs/agent.yml

## Discovery Commands

### Show discovery config status
nms-agentctl discovery status --config configs/agent.yml

### Preview discovery result without writing devices/profiles
nms-agentctl discovery preview --config configs/agent.yml --subnet 192.168.10.0/24 --max-new-devices 50

### Run one discovery cycle immediately
nms-agentctl discovery run --config configs/agent.yml --subnet 192.168.10.0/24 --max-new-devices 50

### Run discovery on Linux/systemd installs
/opt/nms-agent/nms-agentctl discovery run --config /etc/nms-agent/agent.yml --subnet 192.168.10.0/24 --max-new-devices 50

### Override interface/provider/SNMP community during discovery
nms-agentctl discovery run --config configs/agent.yml --subnet 192.168.10.0/24 --interface eth0 --provider active --snmp-community '${SNMP_COMMUNITY}'

### Max new device semantics
- Omit `--max-new-devices` or set `0` to use default `50`.
- Use a positive value for an explicit promotion limit.
- Use `-1` for unlimited promotion.
- Manual discovery always requires SNMP OK, `sysObjectID`, and known profile match before writing devices.
- SNMP probe failures are shown as `SNMP_PROBE_FAILED` and skipped.
- On Linux/systemd installs, discovery may be run as `root`; promoted device/profile files are re-owned to service user `nms-agent` before becoming final files.

## View Command

### View live daemon state summary
nms-agentctl view --config /etc/nms-agent/agent.yml --mode summary

Summary mode shows:

- device totals: `total`, `up`, `down`, `unknown`
- alert totals: `warning`, `critical`
- metric total for latest telemetry batch: `Metrics: total=<n>`
- last update timestamp
- per-device table with `DEVICE`, `STATUS`, `LAST SEEN`, `LATENCY`, `LOSS`, `METRICS`, `ALERTS`
- per-device `METRICS` count is number of telemetry metrics produced by that device in latest batch only, not cumulative

### View raw live telemetry stream
nms-agentctl view --config /etc/nms-agent/agent.yml --mode raw

Notes:

- `view` connects to local daemon socket `/run/nms-agent/view.sock`.
- `view` requires running `nms-agent` daemon with viewer socket available.
- This command is intended for Linux/Unix-style runtime environments.

## Threshold Commands

### List all threshold rules
nms-agentctl threshold list --config configs/agent.yml

### Set / upsert threshold (global metric)
nms-agentctl threshold set --config configs/agent.yml \
  --metric snmp.if.rx.utilization_pct \
  --operator ">" \
  --warning 70 \
  --critical 90

### Set / upsert threshold (tag-specific rule)
nms-agentctl threshold set --config configs/agent.yml \
  --metric icmp.latency_ms \
  --operator ">" \
  --warning 100 \
  --critical 250 \
  --tags source=ping,device_id=router-01

### Update existing rule (same metric + same tags = update)
nms-agentctl threshold set --config configs/agent.yml \
  --metric icmp.latency_ms \
  --operator ">" \
  --warning 120 \
  --critical 300 \
  --tags source=ping,device_id=router-01
