# Canonical Telemetry Contract

All collectors must output canonical telemetry.
Adapters must only accept canonical telemetry.

## Required Fields

- metric.name
- metric.value_type
- metric.value_number
- metric.value_string
- metric.unit
- timestamp
- source.protocol

## Threshold Tags (Phase 7)

When a threshold rule matches, the following tags are added to telemetry:

- threshold.status: ok|warning|critical
- threshold.matched: true
- threshold.rule: <metric>#<index>

## Derived Interface Metrics (Phase 7)

- snmp.if.rx_bps
- snmp.if.tx_bps
- snmp.if.rx_utilization_pct
- snmp.if.tx_utilization_pct

Utilization requires interface speed. Effective speed is resolved with this
priority:
1. snmp.if.speed_bps (if > 0)
2. snmp.if.high_speed_mbps × 1,000,000 (fallback if speed_bps is 0)

## ICMP Metrics (Phase 5)

- icmp.reachable
- icmp.latency_ms
- icmp.jitter_ms
- icmp.packet_loss_pct

## Device Resource Metrics (Phase 6)

- snmp.host.cpu.load_pct
- snmp.host.memory.size_kb

Optional Linux/Proxmox (UCD-SNMP-MIB) memory breakdown (all in kB):

- snmp.host.memory.free_kb
- snmp.host.memory.available_kb
- snmp.host.memory.shared_kb
- snmp.host.memory.buffer_kb
- snmp.host.memory.cached_kb
- snmp.host.swap.total_kb
- snmp.host.swap.free_kb

## Host Storage Metrics (hrStorage, Phase 6)

Indexed (tag `ifIndex` is the hrStorage index):

- snmp.host.storage.type
- snmp.host.storage.description
- snmp.host.storage.allocation_units
- snmp.host.storage.size_units
- snmp.host.storage.used_units

## Route Inventory Metrics (Phase 14)

Summary metrics:

- route.ipv4.supported
- route.ipv4.route_count
- route.ipv4.default_route_count
- route.ipv4.connected_route_count
- route.ipv4.remote_route_count
- route.ipv4.changed

Detail/default-route/snapshot records remain canonical string-valued records:

- route.ipv4.default.destination
- route.ipv4.default.next_hop
- route.ipv4.default.interface_id
- route.ipv4.default.interface_name
- route.ipv4.default.protocol
- route.ipv4.default.route_type
- route.ipv4.source
- route.ipv4.snapshot

Route inventory is built-in for SNMP-enabled devices. Unsupported route tables must emit `route.ipv4.supported=0` without failing the main polling cycle.

## Physical Interface Filtering

Classifier global (semua device) menggunakan multi-signal:

1. **ifName virtual pattern** → drop  
   Proxmox: vmbr*, tap*, veth*, fwbr*, fwpr*, fwln*  
   Docker/K8s: docker*, br-*, cni*, flannel*, cali*  
   Lainnya: lo, dummy, tun, bond, team, sit, gre, wg, tailscale, zt, vnet, virbr*

2. **ifConnectorPresent=true(1)** → keep (strong physical signal)

3. **ifConnectorPresent=false(2)** → drop (explicit non-physical)

4. **ifType allowlist (6=ethernet, 71=wifi)** → keep

5. **ifType known but not in allowlist** → drop

6. **Semua sinyal tidak diketahui** → keep (safe default)

Derived metrics (rx_bps, tx_bps, utilization) ikut terfilter otomatis.

### Metrics yang digunakan

- snmp.if.type (OID 1.3.6.1.2.1.2.2.1.3)
- snmp.if.name (OID 1.3.6.1.2.1.31.1.1.1.1)
- snmp.if.connector_present (OID 1.3.6.1.2.1.31.1.1.1.17)

## Device Identity Metrics (Phase 7)

- snmp.system.description
- snmp.system.name
- snmp.if.name

## Metric Normalization (Phase 7)

Normalization diterapkan setelah derived metrics, sebelum threshold evaluation. Hanya untuk `value_type=number`:

| Aturan | Metric pattern | Clamp | Unit default |
|---|---|---|---|
| Persentase | `*_pct` | `[0, 100]` | `pct` |
| Milidetik | `*_ms` | `≥ 0` | `ms` |
| Detik | `*_seconds` | `≥ 0` | `s` |
| Bits per detik | `*_bps` | `≥ 0` | `bps` |
| Reachability | `icmp.reachable` | `0` atau `1` | — |

Unit default hanya diisi jika tag `unit` belum ada.

## Metric Value Types

- value_type: number|string (required)
- value_number: float (required when value_type=number)
- value_string: string (required when value_type=string)
