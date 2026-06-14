# AI Context Guide

Read this before choosing context.

## For queue/store-and-forward tasks
Read:
- docs/QUEUE_DESIGN.md
- docs/DATA_CONTRACT.md
- internal/queue/*

## For adapter tasks — development, refactor, or adding a new adapter
Read:
- docs/ADAPTER_CONTRACT.md
- docs/DATA_CONTRACT.md
- internal/adapters/base/port.go
- internal/adapters/base/format.go
- internal/adapters/base/output_timezone.go
- internal/adapters/factory.go
- internal/adapters/<closest-existing-adapter>/adapter.go

## For config or CLI tasks
Read:
- docs/CONFIG_SCHEMA.md
- docs/DEVELOPMENT_WORKFLOW.md
- cmd/nms-agentctl/*

## For SNMP/vendor profile tasks
Read:
- docs/DEVICE_PROFILE.md
- internal/profiles/*
- internal/collectors/*
