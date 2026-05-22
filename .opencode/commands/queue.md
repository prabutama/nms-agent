---
description: Work on SQLite queue and store-and-forward logic
---

Read:
- docs/QUEUE_DESIGN.md
- docs/DATA_CONTRACT.md
- docs/AI_CONTEXT.md

Task:
$ARGUMENTS

Rules:
- Queue is the source of truth before adapter send.
- Telemetry must be saved before sending.
- Failed send must remain pending.
- Use event_id to reduce duplicate risk.
- Do not modify adapters unless necessary.

Run:
- make test
- make build

Return:
- queue behavior implemented
- changed files
- tests added or updated