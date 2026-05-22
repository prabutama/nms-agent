---
description: Work on config loader, CLI config update, and hot reload
---

Read:
- docs/CONFIG_SCHEMA.md
- docs/ARCHITECTURE.md
- docs/AI_CONTEXT.md

Task:
$ARGUMENTS

Rules:
- Validate config before saving.
- Do not hardcode secrets.
- Keep .env for sensitive values.
- Reload should use SIGHUP/systemctl reload.
- Invalid config must not break running agent.

Run:
- make test
- make build

Return:
- config behavior
- changed files
- validation result