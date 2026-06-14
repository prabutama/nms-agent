---
description: Run project validation and summarize failures
---

Run project validation.

Read first:
- AGENTS.md
- docs/AI_CONTEXT.md

Commands:
- make fmt
- make test
- make vet
- make build
- make check

Rules:
- Do not edit files unless explicitly asked.
- Do not hide or ignore validation failures.
- Do not mark the project as valid if any command fails.
- If a command is missing, report it clearly and continue with the remaining validation commands.
- Keep the diagnosis focused on the smallest likely fix.

If any command fails:
- show which command failed
- summarize the failure
- identify likely files/functions
- propose the smallest fix
- mention whether docs or contracts may need updates

Return:
- validation summary
- command results
- failures and likely causes
- smallest recommended fix
