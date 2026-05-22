## Architecture Rules

The agent follows this strict flow:

`collect → preprocess → normalize → queue → adapter send`

Core logic must stay platform-agnostic. Do not add direct dependencies on ThingsBoard, Zabbix, Prometheus, or other platforms inside collectors, processors, queue, scheduler, or models.

All platform-specific behavior must live inside adapters.

Collectors and device profiles only gather raw device data. Core services validate, preprocess, and normalize it into canonical telemetry. Telemetry must be persisted to the local queue before sending. If adapter delivery fails, the data remains pending for retry.

Update docs/KNOWLEDGE.md after file changes
---
Update `docs/KNOWLEDGE.md`.

Task:
$ARGUMENTS

Rules:
- Only update the table "FILE KNOWLEDGE TABLE".
- Add new files that are not documented yet.
- Update moved, renamed, or deleted files.
- Keep descriptions short and clear.
- Explain each file's role in the agent architecture.
- Do not modify source code.    


## Development Status Rules

Every implementation task must update `docs/DEVELOPMENT_STAGES.md`.

The agent must:
- update the relevant task status;
- add a new entry to the Development Log;
- list validation commands that were run;
- avoid marking a task as DONE unless build/test succeeds;
- update `docs/KNOWLEDGE.md` when new files are created, renamed, moved, or deleted.