## Core Architecture Rules

- The agent follows a strict hexagonal flow:  
  `collect → preprocess → normalize → queue → adapter send`.

- Business logic must stay inside core services. All external integrations must be placed behind ports/adapters.

- The core agent must remain platform-agnostic. It must not directly depend on ThingsBoard, Zabbix, Prometheus, or any specific monitoring platform.

- Platform-specific behavior must only exist inside adapter implementations.

- Data handling must use a canonical telemetry contract after normalization. This contract decouples collectors, device profiles, queue, and adapters.

- Reliability is based on store-and-forward. Telemetry must be persisted to the local queue before delivery. If sending fails, the data must remain pending for retry.

- Responsibilities are split by layer:
  - collectors and profiles gather device/vendor data;
  - core services handle validation, preprocessing, and normalization;
  - queue manages local durability and retry state;
  - adapters translate and send telemetry to target platforms.

- Route inventory follows the same canonical flow. Core emits only canonical route records; any split of route summary vs route attributes/snapshots must happen in adapters or external gateway converters, not in route collectors or core services.
