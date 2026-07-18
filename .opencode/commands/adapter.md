---
description: Work on output adapters
---

Read first:
- AGENTS.md
- docs/AI_CONTEXT.md
- docs/ADAPTER_CONTRACT.md
- docs/DATA_CONTRACT.md
- docs/QUEUE_DESIGN.md

Adapter task:
$ARGUMENTS

Rules:
- Use semantic search first.
- Do not scan the whole repository.
- Adapter must receive canonical telemetry only.
- Adapter must not collect data.
- Adapter must not access device profile directly.
- Adapter must not modify collector, processor, or queue behavior unless explicitly required.
- Adapter must return clear success or failure.
- If send fails, queue worker must keep data pending.
- Do not bypass store-and-forward semantics.
- Do not log secrets, tokens, API keys, SNMP communities, or passwords.
- Keep changes small and testable.
- If adapter contract changes, update docs/ADAPTER_CONTRACT.md.
- If telemetry/attribute payload shape changes, update docs/DATA_CONTRACT.md.
- If queue behavior changes, update docs/QUEUE_DESIGN.md.
- If config fields change, update docs/CONFIG_SCHEMA.md.
- If creating, renaming, moving, or deleting files, update docs/KNOWLEDGE.md.
- Always update docs/DEVELOPMENT_STAGES.md with a development log entry.
- After implementation, remind user to run `make check`.

Return:
- adapter behavior implemented
- changed files
- contract/docs updated
- remaining risks
