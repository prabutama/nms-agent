---
description: Work on config loader, CLI config update, env expansion, validation, and hot reload
---

Read first:
- AGENTS.md
- docs/AI_CONTEXT.md
- docs/CONFIG_SCHEMA.md
- docs/ARCHITECTURE.md
- docs/DEVELOPMENT_WORKFLOW.md

Config task:
$ARGUMENTS

Rules:
- Use semantic search first.
- Do not scan the whole repository.
- Validate config before saving or applying.
- Do not hardcode secrets.
- Keep sensitive values in environment variables or .env examples only.
- Do not log secrets, tokens, API keys, SNMP communities, or passwords.
- Env expansion must be deterministic and testable.
- Invalid config must not break a running agent.
- Runtime reload should use the existing reload mechanism, such as SIGHUP/systemctl reload, if available.
- Preserve backward compatibility unless explicitly asked to break it.
- If config schema changes, update docs/CONFIG_SCHEMA.md.
- If adapter behavior changes because of config, update docs/ADAPTER_CONTRACT.md.
- If creating, renaming, moving, or deleting files, update docs/KNOWLEDGE.md.
- Always update docs/DEVELOPMENT_STAGES.md with a development log entry.
- After implementation, remind user to run `make check`.

Return:
- config behavior implemented
- changed files
- docs updated
- remaining risks
