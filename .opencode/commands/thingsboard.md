---
description: Work on ThingsBoard integration, REST API, MQTT Gateway, alarms, topology, relations, dashboards, customers, users, assets, devices, and notifications
---

Read first:
- AGENTS.md
- docs/AI_CONTEXT.md
- docs/ARCHITECTURE.md
- docs/ADAPTER_CONTRACT.md
- docs/DATA_CONTRACT.md
- docs/CONFIG_SCHEMA.md
- docs/QUEUE_DESIGN.md
- docs/api/thingsboard/README.md
- docs/api/thingsboard/nms-agent-usecases.md
- docs/api/thingsboard/openapi.json

ThingsBoard task:
$ARGUMENTS

Architecture rules:
- Keep core platform-agnostic.
- Keep ThingsBoard-specific logic inside the ThingsBoard adapter/integration layer.
- Do not put REST API calls inside collectors, processors, queue, or core.
- MQTT Gateway is the data plane for telemetry and device attributes.
- REST API is the control plane for asset, device, relation, alarm, topology, dashboard, customer, user, and notification automation.
- Do not replace the MQTT Gateway telemetry flow unless explicitly requested.
- Do not bypass queue/store-and-forward behavior for telemetry delivery.

Authentication and security rules:
- Use API key auth header:
  X-Authorization: ApiKey ${TB_API_KEY}
- Use a dedicated tenant integration API key for management automation.
- Do not use customer-level API key for tenant management tasks.
- Do not log API keys, tokens, credentials, SNMP communities, or secrets.
- Do not commit real secrets.
- Prefer environment variables for API keys and base URLs.

Implementation rules:
- Use semantic search first.
- Do not scan the whole repository.
- Touch only files related to this task.
- Prefer idempotent reconciliation over blind create.
- Check current ThingsBoard state before creating/updating entities.
- Do not implement delete operations unless explicitly requested.
- For relations, use the model:
  ASSET(site) --Contains--> DEVICE
- Relation create must be idempotent.
- Alarm create/update must avoid spam and should clear alarms when conditions return to normal.
- Dashboard automation should prefer template-based provisioning, not generating complex widget layout from scratch each run.
- Customer/user automation must be explicit and guarded.
- Notification automation must be explicit and guarded.

Docs and contract rules:
- If adding/changing REST config fields, update docs/CONFIG_SCHEMA.md.
- If changing adapter behavior, update docs/ADAPTER_CONTRACT.md.
- If changing telemetry/attribute/alarm/topology data shape, update docs/DATA_CONTRACT.md.
- If creating, renaming, moving, or deleting files, update docs/KNOWLEDGE.md.
- Always update docs/DEVELOPMENT_STAGES.md with a development log entry.
- After implementation, remind user to run `make check`.

Return:
- behavior implemented
- changed files
- ThingsBoard API endpoints used
- config/docs/contracts updated
- remaining risks
