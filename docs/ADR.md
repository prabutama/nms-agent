# ADR

## ADR-001: Use Go for agent implementation

Reason:
- single binary
- lightweight service
- strong concurrency
- good for systemd and CLI

## ADR-002: Store-and-forward is placed at agent level

Reason:
- previous downtime test returned only 4/10 records
- data must be buffered close to the source

## ADR-003: Use adapter pattern

Reason:
- core agent must be platform-agnostic