# Adapter Contract

Adapters convert canonical telemetry into platform-specific format.

## Rules

- Adapter must not collect data.
- Adapter must not modify queue directly.
- Adapter returns success or failure to queue worker.
- Adapter must not drop telemetry silently.

## MVP Adapters

- Terminal
- ThingsBoard MQTT
- Generic MQTT