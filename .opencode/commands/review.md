---
description: Review code changes without editing files
---

Review the current changes for:
- architecture violations
- platform coupling
- queue/store-and-forward correctness
- config validation issues
- adapter contract violations
- data contract changes
- missing tests
- missing documentation updates
- secret leakage risk

Read first:
- AGENTS.md
- docs/AI_CONTEXT.md
- docs/ARCHITECTURE.md
- docs/DATA_CONTRACT.md
- docs/ADAPTER_CONTRACT.md
- docs/QUEUE_DESIGN.md
- docs/CONFIG_SCHEMA.md

Rules:
- Do not edit files.
- Do not run destructive commands.
- Be specific.
- Mention file names and functions when possible.
- Prioritize critical issues first.
- Do not nitpick formatting unless it affects correctness or maintainability.

Output:
- Critical issues
- Major issues
- Minor issues
- Missing tests
- Missing docs/contract updates
- Suggested minimal fixes
- Overall recommendation
