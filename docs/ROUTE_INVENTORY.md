# Route Inventory

Route inventory is a built-in SNMP capability for all devices with `snmp.enabled=true`.

## Provider Order

The IPv4 route provider uses this order:

1. `ipCidrRouteTable` (primary MVP)
2. `inetCidrRouteTable` (best-effort only)
3. `ipRouteTable` (legacy fallback)

If all route tables are unsupported or empty, the agent publishes `route.ipv4.supported=0` and continues the normal polling cycle.

## Canonical Output

Summary metrics are numeric canonical records:

- `route.ipv4.supported`
- `route.ipv4.route_count`
- `route.ipv4.default_route_count`
- `route.ipv4.connected_route_count`
- `route.ipv4.remote_route_count`
- `route.ipv4.changed`

Default-route and snapshot details remain canonical string records:

- `route.ipv4.default.destination`
- `route.ipv4.default.next_hop`
- `route.ipv4.default.interface_id`
- `route.ipv4.default.interface_name`
- `route.ipv4.default.protocol`
- `route.ipv4.default.route_type`
- `route.ipv4.source`
- `route.ipv4.snapshot`

Core does not introduce a separate attribute contract. Target adapters or gateway converters may project these string records into attributes if the platform supports it.

## Interface Resolution

The provider builds an `ifIndex -> ifName` map from IF-MIB `ifName`, with `ifDescr` fallback.

If a route row reports interface `0` or empty but has a non-zero next-hop, the resolver finds the most specific connected route that contains that next-hop and copies its interface metadata. In that case `interface_resolved_by` is set to `next_hop_connected_route`.

## Future Logical Topology

This task does not build topology yet. Future topology builders can use:

- device default next-hop
- connected routes/subnets
- interface id/interface name
- source table
- collection timestamp

Virtual/container routes remain in the route inventory. Future topology logic may ignore them when inferring logical edges.
