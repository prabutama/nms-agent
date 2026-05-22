---
description: Implement a focused feature with minimal file changes
---

Read AGENTS.md and docs/AI_CONTEXT.md.

Feature to implement:
$ARGUMENTS

Rules:
- Use semantic search first.
- Do not scan the whole repository.
- Touch only files related to this feature.
- Keep changes small and testable.
- Do not change public contracts unless required.
- If changing contracts, update relevant docs.
- If you create, rename, move, or delete any file, update `docs/KNOWLEDGE.md`.
- Update the table "FILE KNOWLEDGE TABLE" with the file path and its responsibility.

After implementation, run:
- make fmt
- make test
- make build

Return:
- changed files
- validation result
- remaining risks