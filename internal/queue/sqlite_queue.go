package queue

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	_ "modernc.org/sqlite"

	"nms-agent/internal/models"
)

// SQLiteQueue is the durable queue implementation (Phase 4A).
// It persists canonical telemetry as JSON for store-and-forward delivery.
type SQLiteQueue struct {
	db *sql.DB
}

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
			"updated_at TEXT NOT NULL\n" +
			");",
		"CREATE INDEX IF NOT EXISTS idx_queue_items_status_created ON queue_items(status, created_at);",
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
	rows, err := q.db.QueryContext(ctx,
		"SELECT id, payload_json, retry_count, created_at FROM queue_items WHERE status='pending' ORDER BY created_at ASC LIMIT ?",
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []QueueItem
	for rows.Next() {
		var (
			id         string
			payload    string
			retryCount int
			createdAt  string
		)
		if err := rows.Scan(&id, &payload, &retryCount, &createdAt); err != nil {
			return nil, err
		}
		var t models.Telemetry
		if err := json.Unmarshal([]byte(payload), &t); err != nil {
			return nil, err
		}
		ct, _ := time.Parse(time.RFC3339Nano, createdAt)
		out = append(out, QueueItem{ID: id, Telemetry: t, RetryCount: retryCount, CreatedAt: ct})
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
		stmt, err := tx.PrepareContext(ctx, "DELETE FROM queue_items WHERE id=?")
		if err != nil {
			return err
		}
		defer stmt.Close()
		for _, id := range ids {
			if _, err := stmt.ExecContext(ctx, id); err != nil {
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
	now := time.Now().UTC().Format(time.RFC3339Nano)
	return execInTx(ctx, q.db, func(tx *sql.Tx) error {
		stmt, err := tx.PrepareContext(ctx, "UPDATE queue_items SET retry_count=retry_count+1, last_error=?, updated_at=? WHERE id=?")
		if err != nil {
			return err
		}
		defer stmt.Close()
		for _, id := range ids {
			if _, err := stmt.ExecContext(ctx, reason, now, id); err != nil {
				return err
			}
		}
		return nil
	})
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

func newQueueID() string {
	// UUID avoids time-resolution collisions under high-frequency enqueue.
	return "q-" + uuid.NewString()
}
