---
description: Plan a small implementation task without editing files
---

Read first:
- AGENTS.md
- docs/AI_CONTEXT.md
- docs/ARCHITECTURE.md
- docs/DEVELOPMENT_WORKFLOW.md

Task to plan:
$ARGUMENTS

Rules:
- Do not edit files.
- Do not create files.
- Do not run destructive commands.
- Use semantic search first.
- Do not scan the whole repository.
- Identify only relevant docs and files.
- Keep the plan small and incremental.
- Avoid large refactors unless the task explicitly requires them.
- Mention whether contracts/docs likely need updates.
- Mention validation commands to run after implementation.
- Ask for confirmation before implementation.

Return:
- concise understanding of the task
- relevant files/docs to inspect
- proposed implementation steps
- expected files to edit
- expected tests
- validation commands
- risks/open questions
