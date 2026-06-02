# Security Notes

Security guidelines for deploying and operating the NMS Agent.

## Credentials and Secrets

### MQTT Access Tokens

- Store ThingsBoard access tokens securely in `adapters.yml`
- Never commit `adapters.yml` with real tokens to version control
- Use environment variables for sensitive values where possible
- Rotate tokens regularly through the ThingsBoard dashboard

### SNMP Community Strings

- Use SNMPv3 with authentication and encryption when available
- For SNMPv2c, use complex community strings instead of default `public`/`private`
- Restrict SNMP access via firewall rules to known gateway IPs only
- Do not share community strings across environments

### File Permissions

- Config files may contain credentials — restrict file access:

```bash
chmod 600 /etc/nms-agent/adapters.yml
chmod 600 /etc/nms-agent/agent.yml
chmod 600 /etc/nms-agent/devices.d/*.yml
```

- Queue database contains telemetry data:

```bash
chmod 600 /var/lib/nms-agent/queue.db
```

- Agent user should have minimal required permissions:

```bash
useradd -r -s /sbin/nologin nms-agent
chown -R nms-agent:nms-agent /opt/nms-agent
chown -R nms-agent:nms-agent /etc/nms-agent
chown -R nms-agent:nms-agent /var/lib/nms-agent
```

## Network Security

### Outbound Connections

The agent makes outbound connections to:
- SNMP agents on UDP port 161
- ICMP ping to target devices
- MQTT broker on configured port (default 1883 or 8883 for TLS)

Ensure firewall rules only allow necessary connections.

### TLS for MQTT

- Use `tls://` or `mqtts://` scheme in broker URL for encrypted connections
- Configure CA certificate verification in adapter config
- Disable `InsecureSkipVerify` in production unless absolutely necessary

### Inbound Connections

The agent does not listen for inbound connections. No additional firewall rules are needed for inbound traffic.

## Environment Variables

- Use `${ENV_VAR}` placeholders in config files for sensitive values
- Document required environment variables in deployment guides
- Never hardcode secrets in config files

## Queue Data

- SQLite queue stores raw and processed telemetry data
- Queue data may contain network topology information
- Ensure queue storage volume is encrypted at rest where required
- Implement queue data retention policies if needed

## Binary Distribution

- Verify binary integrity using checksums or signatures
- Build from source in production environments when possible
- Use `go build` with reproducible builds for auditability

## Audit and Monitoring

- Monitor agent logs for authentication failures
- Alert on adapter connection failures
- Track queue depth for delivery health
- Review device config changes regularly

## Known Security Considerations

### SNMP v2c Limitations

SNMP v2c transmits community strings in cleartext. Use SNMPv3 in production environments that require it.

### ICMP Exposure

ICMP ping requires network-level access to target devices. Ensure ICMP is not exposed to unauthorized networks.

### SQLite Security

SQLite stores data in a single file with no built-in encryption. For sensitive deployments, use encrypted filesystems or LUKS volumes.
