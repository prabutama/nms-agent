package queue

import (
	"context"
	"database/sql"
)

// Stats contains a minimal queue status snapshot.
type Stats struct {
	PendingCount int
	MaxRetry     int
}

// Stats returns pending count and max retry_count for the SQLite-backed queue.
// It does not modify queue state.
func (q *SQLiteQueue) Stats(ctx context.Context) (Stats, error) {
	var s Stats

	if q == nil || q.db == nil {
		return Stats{}, sql.ErrConnDone
	}

	if err := q.db.QueryRowContext(ctx, "SELECT COUNT(1) FROM queue_items WHERE status='pending'").Scan(&s.PendingCount); err != nil {
		return Stats{}, err
	}
	// MAX() returns NULL on empty tables.
	var max sql.NullInt64
	if err := q.db.QueryRowContext(ctx, "SELECT MAX(retry_count) FROM queue_items WHERE status='pending'").Scan(&max); err != nil {
		return Stats{}, err
	}
	if max.Valid {
		s.MaxRetry = int(max.Int64)
	}
	return s, nil
}
