# NMS Agent CLI Command Examples

## Validate Config
nms-agentctl validate --config configs/agent.yml

## Queue Commands
nms-agentctl queue status --config configs/agent.yml
nms-agentctl queue retry --config configs/agent.yml --limit 100

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
