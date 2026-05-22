---
description: Work on output adapters
---

Read:
- docs/ADAPTER_CONTRACT.md
- docs/DATA_CONTRACT.md
- docs/AI_CONTEXT.md

Adapter task:
$ARGUMENTS

Rules:
- Adapter must receive canonical telemetry only.
- Adapter must not collect data.
- Adapter must not access device profile.
- Adapter must return success or failure.
- If send fails, queue worker must keep data pending.

Run:
- make test
- make build

Return:
- adapter behavior
- changed files
- validation result