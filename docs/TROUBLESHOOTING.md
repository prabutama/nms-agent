# Troubleshooting Guide

Common issues and their fixes for the NMS Agent.

## Configuration Errors

### Config path not found

**Error:** `open configs/agent.yml: no such file or directory`

**Fix:** Ensure the config path is correct and accessible. Use absolute paths for production deployments.

```bash
nms-agentctl validate --config /full/path/to/agent.yml
```

### Invalid YAML syntax

**Error:** `yaml: ...` during validation

**Fix:** Check YAML indentation and syntax. Use a YAML linter or validator.

```bash
nms-agentctl validate --config configs/agent.yml
```

### Duplicate device ID

**Error:** `duplicate device id: <id>` during validation

**Fix:** Ensure each device file in `devices.d/` has a unique `id` field.

```bash
nms-agentctl device list --config configs/agent.yml
```

## Collector Errors

### SNMP authentication failed

**Error:** `SNMP walk failed: timeout` or `auth failure`

**Fix:**
- Verify SNMP community string or credentials in device config
- Check SNMP service is running on the target device
- Ensure firewall allows UDP port 161
- Verify SNMP version compatibility (v2c vs v3)

### ICMP ping not available

**Error:** `ping: command not found` or permission denied

**Fix:**
- Install ping utility: `apt install iputils-ping` or `yum install iputils`
- Run with appropriate permissions (root or CAP_NET_RAW capability)

```bash
sudo nms-agent run --config configs/agent.yml --collector-mode real
```

### SNMP timeout on specific device

**Error:** `SNMP walk timed out for device <id>`

**Fix:**
- Check network connectivity to the device
- Increase SNMP timeout in device profile if supported
- Verify device SNMP agent is responsive

```bash
nms-agentctl device test --config configs/agent.yml --id <device-id> --snmp=true
```

## Queue Errors

### Queue DB path permission denied

**Error:** `open data/queue/queue.db: permission denied`

**Fix:** Ensure the queue directory is writable by the agent process.

```bash
mkdir -p data/queue
chmod 755 data/queue
```

For systemd deployment, queue is stored at `/var/lib/nms-agent/queue.db`.

### Queue full or disk space

**Error:** `sqlite: disk full` or similar

**Fix:**
- Monitor disk space on the queue volume
- Review queue status regularly:

```bash
nms-agentctl queue status --config configs/agent.yml
```

- Retry pending items:

```bash
nms-agentctl queue retry --config configs/agent.yml
```

## Adapter Errors

### MQTT connection failed

**Error:** `MQTT connect failed: dial tcp <broker>:1883: connection refused`

**Fix:**
- Verify MQTT broker is running and accessible
- Check broker URL in `adapters.yml`
- Verify network connectivity and firewall rules
- For ThingsBoard, verify access token is correct

```bash
nms-agentctl adapter health --config configs/agent.yml
```

### ThingsBoard token invalid

**Error:** `ThingsBoard auth failed: invalid access token`

**Fix:**
- Verify access token in `adapters.yml` matches the gateway token in ThingsBoard
- Regenerate token in ThingsBoard dashboard if needed

## Reload Errors

### Reload failed: config validation error

**Error:** `reload failed: <validation error>`

**Fix:**
- Run validation before reload:

```bash
nms-agentctl validate --config configs/agent.yml
```

- Fix config errors, then retry reload

### Reload failed: PID not found

**Error:** `reload failed: process <pid> not found`

**Fix:**
- Ensure the agent process is running:

```bash
ps aux | grep nms-agent
```

- Get the correct PID:

```bash
nms-agentctl reload --config configs/agent.yml --pid <pid>
```

## systemd Errors

### Service failed to start

**Error:** `systemctl status nms-agent` shows `failed`

**Fix:**
- Check journal logs:

```bash
journalctl -u nms-agent -n 100 --no-pager
```

- Verify config files exist at expected paths
- Verify binary is installed at `/opt/nms-agent/nms-agent`

### Config not reloading after systemctl reload

**Fix:**
- Verify SIGHUP handler is compiled (Unix builds only)
- Check agent logs for reload messages
- Verify config validation passes before reload

## Performance Issues

### High CPU usage

**Possible causes:**
- Too many devices polled at short intervals
- SNMP walk returning excessive data

**Fix:**
- Increase `poll_interval` in `agent.yml`
- Review device profiles for unnecessary OIDs
- Use device profiles to limit collected metrics

### Slow adapter response

**Possible causes:**
- Network latency to broker/platform
- Large batch sizes

**Fix:**
- Reduce `max_batch` in `delivery` config
- Check broker/platform health
- Use `adapter health` command to check connectivity
