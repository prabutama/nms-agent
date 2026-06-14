# Optimized opencode command files for nms-agent

Recommended location:

```text
.opencode/command/
```

Files:

```text
adapter.md      Work on output adapters
check.md        Run validation and summarize failures
config.md       Work on config, env expansion, reload, validation
implement.md    Implement focused feature
plan.md         Plan small task without editing
pr.md           Summarize changes for commit/PR
queue.md        Work on SQLite queue/store-and-forward
review.md       Review current changes without editing
thingsboard.md  Work on ThingsBoard REST/MQTT integration
```

Validation:

```bash
make check
```

Project rules embedded in these commands:

- Read `AGENTS.md` and `docs/AI_CONTEXT.md` first.
- Keep core platform-agnostic.
- Keep ThingsBoard-specific logic inside the adapter/integration layer.
- Preserve queue/store-and-forward semantics.
- Do not log secrets.
- Update contract docs when contracts change.
- Update `docs/KNOWLEDGE.md` when files are created/renamed/moved/deleted.
- Always update `docs/DEVELOPMENT_STAGES.md` after implementation.
- After implementation, remind user to run `make check`.
