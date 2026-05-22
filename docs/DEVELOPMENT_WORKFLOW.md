# Development Workflow

## AI Agent Rules

- Read AGENTS.md and docs/AI_CONTEXT.md before planning.
- Do not scan the whole repository unless required.
- Use Serena to inspect relevant files only.
- Keep each task small and focused.
- Prefer modifying 3–5 files maximum per task.
- Do not change public contracts unless required.
- If changing a contract, update the related docs.

## Development Flow

1. Plan the task.
2. Identify relevant files.
3. Implement small changes.
4. Run validation commands.
5. Summarize changed files, test results, and risks.

## Validation Commands

- make fmt 
- make test 
- make vet 
- make build 