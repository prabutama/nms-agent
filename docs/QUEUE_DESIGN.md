# Queue Design (Phase 4)

This document defines the local store-and-forward queue.

## Goals

- Persist canonical telemetry locally before adapter delivery.
- Support retries without dropping data on transient failures.
- Allow precise acknowledgements using queue item IDs.

## Data Model

The queue stores canonical telemetry serialized as JSON, with queue metadata.

### SQLite Table

Table: `queue_items`

- `id` (TEXT, PRIMARY KEY): queue item ID.
- `payload_json` (TEXT, NOT NULL): JSON-encoded canonical telemetry.
- `status` (TEXT, NOT NULL): `pending` for Phase 4A.
- `retry_count` (INTEGER, NOT NULL): number of failed send attempts.
- `last_error` (TEXT, NULL): last adapter error string.
- `created_at` (TEXT, NOT NULL): RFC3339Nano.
- `updated_at` (TEXT, NOT NULL): RFC3339Nano.

Index:
- `(status, created_at)` for efficient oldest-first pending reads.

## API Behavior (Phase 4A)

- Enqueue: insert one row per telemetry record.
- PendingBatch: returns the oldest `pending` items.
- MarkDelivered(ids): deletes the rows by ID.
- MarkFailed(ids, reason): increments `retry_count` and stores `last_error`.

## Retry Policy

Phase 4A only tracks `retry_count` and `last_error`.
Backoff/TTL/deduplication are deferred to later tasks.

## Expected Downtime Behavior

If the adapter is unavailable, items remain `pending` in SQLite and will be retried on the next run.
