# Release Artifacts

## Build Artifacts

### Linux (amd64)

```bash
GOOS=linux GOARCH=amd64 go build -o dist/nms-agent-linux-amd64 ./cmd/nms-agent
GOOS=linux GOARCH=amd64 go build -o dist/nms-agentctl-linux-amd64 ./cmd/nms-agentctl
```

### Windows (amd64)

```bash
go build -o dist/nms-agent.exe ./cmd/nms-agent
go build -o dist/nms-agentctl.exe ./cmd/nms-agentctl
```

### Linux (arm64)

```bash
GOOS=linux GOARCH=arm64 go build -o dist/nms-agent-linux-arm64 ./cmd/nms-agent
GOOS=linux GOARCH=arm64 go build -o dist/nms-agentctl-linux-arm64 ./cmd/nms-agentctl
```

## Release Package Contents

Each release should include:

- `nms-agent` binary
- `nms-agentctl` binary
- `packaging/systemd/nms-agent.service`
- `packaging/systemd/install.sh`
- `packaging/systemd/README.md`
- Sample configs:
  - `configs/agent.yml`
  - `configs/adapters.yml`
  - `configs/thresholds.yml`
  - `configs/devices.d/` (example device files)
  - `profiles/` (SNMP device profiles)
- `README.md`
- `docs/TROUBLESHOOTING.md`
- `docs/SECURITY.md`

## Deployment Checklist

- [ ] Binary built and tested on target platform
- [ ] Config files reviewed and credentials updated
- [ ] systemd unit installed and enabled
- [ ] Network firewall rules configured
- [ ] MQTT broker connectivity verified
- [ ] SNMP community strings or v3 credentials configured
- [ ] Threshold rules reviewed for target environment
- [ ] Monitoring/alerting for agent health configured

## Verification Commands

```bash
# Validate binary
./nms-agentctl validate --config configs/agent.yml

# Check device list
./nms-agentctl device list --config configs/agent.yml

# Check queue status
./nms-agentctl queue status --config configs/agent.yml

# Check adapter health
./nms-agentctl adapter health --config configs/agent.yml

# Run with dummy collector (dry run)
./nms-agent run --config configs/agent.yml --collector-mode dummy
```
