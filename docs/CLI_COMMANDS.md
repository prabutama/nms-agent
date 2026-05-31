# NMS Agent CLI Command Examples

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

## Adapter Commands
nms-agentctl adapter health --config configs/agent.yml

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
