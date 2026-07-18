package queue

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"

	_ "modernc.org/sqlite"

	"nms-agent/internal/models"
)

// RetryConfig controls exponential backoff and dead-letter behavior for failed queue items.
// When Enabled is false, all pending items are retried immediately (legacy behavior).
type RetryConfig struct {
	Enabled     bool
	BaseBackoff time.Duration
	MaxBackoff  time.Duration
	MaxRetries  int
}

// SQLiteQueue is the durable queue implementation (Phase 4A).
// It persists canonical telemetry as JSON for store-and-forward delivery.
type SQLiteQueue struct {
	db       *sql.DB
	retryCfg RetryConfig
}

const sqliteIDChunkSize = 900

func OpenSQLite(path string) (*SQLiteQueue, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	// Single writer typical for embedded SQLite.
	db.SetMaxOpenConns(1)

	q := &SQLiteQueue{db: db}
	if err := q.init(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	return q, nil
}

// SetRetryConfig configures exponential backoff for the queue.
// Call before the first cycle; safe to call at any time.
func (q *SQLiteQueue) SetRetryConfig(cfg RetryConfig) {
	q.retryCfg = cfg
}

func (q *SQLiteQueue) Close() error {
	if q == nil || q.db == nil {
		return nil
	}
	return q.db.Close()
}

func (q *SQLiteQueue) init(ctx context.Context) error {
	stmts := []string{
		"PRAGMA journal_mode=WAL;",
		"PRAGMA foreign_keys=ON;",
		"CREATE TABLE IF NOT EXISTS queue_items (\n" +
			"id TEXT PRIMARY KEY,\n" +
			"payload_json TEXT NOT NULL,\n" +
			"status TEXT NOT NULL,\n" +
			"retry_count INTEGER NOT NULL DEFAULT 0,\n" +
			"last_error TEXT,\n" +
			"created_at TEXT NOT NULL,\n" +
			"updated_at TEXT NOT NULL,\n" +
			"next_attempt_at TEXT,\n" +
			"last_attempt_at TEXT\n" +
			");",
		"CREATE INDEX IF NOT EXISTS idx_queue_items_status_created ON queue_items(status, created_at);",
		"CREATE TABLE IF NOT EXISTS thingsboard_device_tokens (\n" +
			"device_id TEXT PRIMARY KEY,\n" +
			"access_token TEXT NOT NULL,\n" +
			"credentials_type TEXT NOT NULL DEFAULT 'ACCESS_TOKEN',\n" +
			"provisioned_at TEXT NOT NULL,\n" +
			"updated_at TEXT NOT NULL,\n" +
			"last_used_at TEXT\n" +
			");",
	}
	// Phase 4 migration: add retry columns if missing.
	for _, alter := range []string{
		"ALTER TABLE queue_items ADD COLUMN next_attempt_at TEXT;",
		"ALTER TABLE queue_items ADD COLUMN last_attempt_at TEXT;",
	} {
		if _, err := q.db.ExecContext(ctx, alter); err != nil {
			// Column already exists — ignore error.
		}
	}
	for _, s := range stmts {
		if _, err := q.db.ExecContext(ctx, s); err != nil {
			return err
		}
	}
	return nil
}

func (q *SQLiteQueue) EnqueueBatch(ctx context.Context, batch []models.Telemetry) error {
	if len(batch) == 0 {
		return nil
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)

	tx, err := q.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.PrepareContext(ctx, "INSERT INTO queue_items(id, payload_json, status, retry_count, last_error, created_at, updated_at) VALUES(?, ?, 'pending', 0, NULL, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, t := range batch {
		payload, err := json.Marshal(t)
		if err != nil {
			return err
		}
		id := newQueueID()
		if _, err := stmt.ExecContext(ctx, id, string(payload), now, now); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (q *SQLiteQueue) PendingBatch(ctx context.Context, limit int) ([]QueueItem, error) {
	if limit <= 0 {
		limit = 100
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	var query string
	var args []any
	if q.retryCfg.Enabled {
		query = "SELECT id, payload_json, retry_count, created_at, next_attempt_at FROM queue_items WHERE status='pending' AND (next_attempt_at IS NULL OR next_attempt_at <= ?) ORDER BY created_at ASC LIMIT ?"
		args = []any{now, limit}
	} else {
		query = "SELECT id, payload_json, retry_count, created_at, next_attempt_at FROM queue_items WHERE status='pending' ORDER BY created_at ASC LIMIT ?"
		args = []any{limit}
	}
	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []QueueItem
	for rows.Next() {
		var (
			id          string
			payload     string
			retryCount  int
			createdAt   string
			nextAttempt sql.NullString
		)
		if err := rows.Scan(&id, &payload, &retryCount, &createdAt, &nextAttempt); err != nil {
			return nil, err
		}
		var t models.Telemetry
		if err := json.Unmarshal([]byte(payload), &t); err != nil {
			return nil, err
		}
		ct, _ := time.Parse(time.RFC3339Nano, createdAt)
		item := QueueItem{ID: id, Telemetry: t, RetryCount: retryCount, CreatedAt: ct}
		if nextAttempt.Valid {
			if t, err := time.Parse(time.RFC3339Nano, nextAttempt.String); err == nil {
				item.NextAttemptAt = t
			}
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (q *SQLiteQueue) MarkDelivered(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	return execInTx(ctx, q.db, func(tx *sql.Tx) error {
		for _, chunk := range chunkStrings(ids, sqliteIDChunkSize) {
			args := make([]any, 0, len(chunk))
			for _, id := range chunk {
				args = append(args, id)
			}
			query := "DELETE FROM queue_items WHERE id IN (" + placeholders(len(chunk)) + ")"
			if _, err := tx.ExecContext(ctx, query, args...); err != nil {
				return err
			}
		}
		return nil
	})
}

func (q *SQLiteQueue) MarkFailed(ctx context.Context, ids []string, reason string) error {
	if len(ids) == 0 {
		return nil
	}
	now := time.Now().UTC()
	nowStr := now.Format(time.RFC3339Nano)
	return execInTx(ctx, q.db, func(tx *sql.Tx) error {
		for _, chunk := range chunkStrings(ids, sqliteIDChunkSize) {
			if q.retryCfg.Enabled {
				base := q.retryCfg.BaseBackoff
				if base <= 0 {
					base = 10 * time.Second
				}
				maxBackoff := q.retryCfg.MaxBackoff
				if maxBackoff <= 0 {
					maxBackoff = 300 * time.Second
				}
				maxRetries := q.retryCfg.MaxRetries
				if maxRetries <= 0 {
					maxRetries = 10
				}
				// Read current retry_count for each id
				ph := placeholders(len(chunk))
				rows, err := tx.QueryContext(ctx, "SELECT id, retry_count FROM queue_items WHERE id IN ("+ph+")", strToAny(chunk)...)
				if err != nil {
					return fmt.Errorf("read retry_count: %w", err)
				}
				counts := make(map[string]int, len(chunk))
				for rows.Next() {
					var id string
					var c int
					if err := rows.Scan(&id, &c); err != nil {
						rows.Close()
						return fmt.Errorf("scan retry_count: %w", err)
					}
					counts[id] = c
				}
				rows.Close()

				for _, id := range chunk {
					c := counts[id]
					newCount := c + 1

					if maxRetries > 0 && newCount >= maxRetries {
						// Move to dead_letter — stop retrying.
						if _, err := tx.ExecContext(ctx,
							"UPDATE queue_items SET retry_count=retry_count+1, status='dead_letter', last_error=?, updated_at=?, last_attempt_at=? WHERE id=?",
							reason, nowStr, nowStr, id,
						); err != nil {
							return fmt.Errorf("mark dead_letter: %w", err)
						}
					} else {
						// Exponential backoff.
						backoff := time.Duration(math.Min(float64(base)*math.Pow(2, float64(newCount)), float64(maxBackoff)))
						nextAttempt := now.Add(backoff).Format(time.RFC3339Nano)
						if _, err := tx.ExecContext(ctx,
							"UPDATE queue_items SET retry_count=retry_count+1, last_error=?, updated_at=?, last_attempt_at=?, next_attempt_at=? WHERE id=?",
							reason, nowStr, nowStr, nextAttempt, id,
						); err != nil {
							return fmt.Errorf("mark failed with backoff: %w", err)
						}
					}
				}
			} else {
				args := make([]any, 0, len(chunk)+3)
				args = append(args, reason, nowStr, nowStr)
				for _, id := range chunk {
					args = append(args, id)
				}
				query := "UPDATE queue_items SET retry_count=retry_count+1, last_error=?, updated_at=?, last_attempt_at=? WHERE id IN (" + placeholders(len(chunk)) + ")"
				if _, err := tx.ExecContext(ctx, query, args...); err != nil {
					return fmt.Errorf("mark failed: %w", err)
				}
			}
		}
		return nil
	})
}

func strToAny(s []string) []any {
	out := make([]any, len(s))
	for i, v := range s {
		out[i] = v
	}
	return out
}

// Snapshot returns the latest queue items for local viewing.
// It is read-only and does not change queue state.
func (q *SQLiteQueue) Snapshot(ctx context.Context, limit int) ([]QueueItem, error) {
	if limit <= 0 {
		limit = 200
	}
	rows, err := q.db.QueryContext(ctx,
		"SELECT id, payload_json, retry_count, created_at, next_attempt_at FROM queue_items ORDER BY created_at DESC LIMIT ?",
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []QueueItem
	for rows.Next() {
		var (
			id          string
			payload     string
			retryCount  int
			createdAt   string
			nextAttempt sql.NullString
		)
		if err := rows.Scan(&id, &payload, &retryCount, &createdAt, &nextAttempt); err != nil {
			return nil, err
		}
		var t models.Telemetry
		if err := json.Unmarshal([]byte(payload), &t); err != nil {
			return nil, err
		}
		ct, _ := time.Parse(time.RFC3339Nano, createdAt)
		item := QueueItem{ID: id, Telemetry: t, RetryCount: retryCount, CreatedAt: ct}
		if nextAttempt.Valid {
			if t, err := time.Parse(time.RFC3339Nano, nextAttempt.String); err == nil {
				item.NextAttemptAt = t
			}
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func execInTx(ctx context.Context, db *sql.DB, fn func(*sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}

func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.TrimRight(strings.Repeat("?,", n), ",")
}

func chunkStrings(values []string, size int) [][]string {
	if size <= 0 || size >= len(values) {
		return [][]string{values}
	}
	chunks := make([][]string, 0, (len(values)+size-1)/size)
	for start := 0; start < len(values); start += size {
		end := start + size
		if end > len(values) {
			end = len(values)
		}
		chunks = append(chunks, values[start:end])
	}
	return chunks
}

// CleanupDeleted removes items that are dead_letter or delivered (status not 'pending')
// and older than the given age. Returns number of rows deleted.
func (q *SQLiteQueue) CleanupDeleted(ctx context.Context, olderThan time.Duration) (int64, error) {
	if q == nil || q.db == nil {
		return 0, sql.ErrConnDone
	}
	cutoff := time.Now().UTC().Add(-olderThan).Format(time.RFC3339Nano)
	res, err := q.db.ExecContext(ctx, "DELETE FROM queue_items WHERE status != 'pending' AND updated_at < ?", cutoff)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func newQueueID() string {
	// UUID avoids time-resolution collisions under high-frequency enqueue.
	return "q-" + uuid.NewString()
}
