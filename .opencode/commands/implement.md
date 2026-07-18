---
description: Implement a focused feature with minimal file changes
---

Read first:
- AGENTS.md
- docs/AI_CONTEXT.md
- docs/ARCHITECTURE.md
- docs/DEVELOPMENT_WORKFLOW.md

Feature to implement:
$ARGUMENTS

Rules:
- Use semantic search first.
- Do not scan the whole repository.
- Touch only files related to this feature.
- Keep changes small and testable.
- Prefer focused implementation over broad refactor.
- Do not change public contracts unless required.
- Do not introduce platform coupling into core packages.
- Do not bypass queue/store-and-forward behavior.
- Do not log secrets, tokens, API keys, SNMP communities, or passwords.
- If changing telemetry/attribute/alarm data shape, update docs/DATA_CONTRACT.md.
- If changing adapter behavior, update docs/ADAPTER_CONTRACT.md.
- If changing queue behavior/schema, update docs/QUEUE_DESIGN.md.
- If changing config fields, update docs/CONFIG_SCHEMA.md.
- If changing device profile format, update docs/DEVICE_PROFILE.md.
- If creating, renaming, moving, or deleting files, update docs/KNOWLEDGE.md.
- Always update docs/DEVELOPMENT_STAGES.md with a development log entry.
- After implementation, remind user to run `make check`.

Return:
- behavior implemented
- changed files
- docs/contracts updated
- tests added or updated
- remaining risks
