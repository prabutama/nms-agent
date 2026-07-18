---
description: Work on SQLite queue and store-and-forward logic
---

Read first:
- AGENTS.md
- docs/AI_CONTEXT.md
- docs/QUEUE_DESIGN.md
- docs/DATA_CONTRACT.md
- docs/ADAPTER_CONTRACT.md

Queue task:
$ARGUMENTS

Rules:
- Use semantic search first.
- Do not scan the whole repository.
- Queue is the source of truth before adapter send.
- Telemetry must be saved before sending.
- Failed sends must remain pending.
- Successful sends may be marked sent only after adapter success.
- Use event_id to reduce duplicate risk where applicable.
- Never drop pending telemetry silently.
- Never mark failed sends as successful.
- Do not modify adapters unless necessary.
- Preserve backward compatibility for existing queue data unless explicitly asked.
- If queue schema changes, update docs/QUEUE_DESIGN.md.
- If data shape changes, update docs/DATA_CONTRACT.md.
- If adapter contract changes, update docs/ADAPTER_CONTRACT.md.
- If creating, renaming, moving, or deleting files, update docs/KNOWLEDGE.md.
- Always update docs/DEVELOPMENT_STAGES.md with a development log entry.
- After implementation, remind user to run `make check`.

Return:
- queue behavior implemented
- changed files
- schema/migration impact
- tests added or updated
- remaining risks
