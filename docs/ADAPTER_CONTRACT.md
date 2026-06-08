# Adapter Contract

Adapters convert canonical telemetry into platform-specific format.

## Rules

- Adapter must not collect data.
- Adapter must not modify queue directly.
- Adapter returns success or failure to queue worker.
- Adapter must not drop telemetry silently.
- Adapter may project canonical string records into platform-specific attribute channels, but only after canonical records have already passed through the queue.

## MVP Adapters

- Terminal
- TUI (Bubbletea-based terminal UI)
- ThingsBoard MQTT
- Generic MQTT
