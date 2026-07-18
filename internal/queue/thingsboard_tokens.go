package queue

import (
	"context"
	"database/sql"
	"strings"
	"time"
)

func (q *SQLiteQueue) GetThingsBoardToken(ctx context.Context, deviceID string) (string, bool, error) {
	if q == nil || q.db == nil {
		return "", false, sql.ErrConnDone
	}
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return "", false, nil
	}
	var token string
	err := q.db.QueryRowContext(ctx, "SELECT access_token FROM thingsboard_device_tokens WHERE device_id=?", deviceID).Scan(&token)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return token, true, nil
}

func (q *SQLiteQueue) SaveThingsBoardToken(ctx context.Context, deviceID, token string) error {
	if q == nil || q.db == nil {
		return sql.ErrConnDone
	}
	deviceID = strings.TrimSpace(deviceID)
	token = strings.TrimSpace(token)
	if deviceID == "" || token == "" {
		return nil
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := q.db.ExecContext(ctx, "INSERT INTO thingsboard_device_tokens(device_id, access_token, credentials_type, provisioned_at, updated_at) VALUES(?, ?, 'ACCESS_TOKEN', ?, ?) ON CONFLICT(device_id) DO UPDATE SET access_token=excluded.access_token, credentials_type=excluded.credentials_type, updated_at=excluded.updated_at", deviceID, token, now, now)
	return err
}

func (q *SQLiteQueue) MarkThingsBoardTokenUsed(ctx context.Context, deviceID string) error {
	if q == nil || q.db == nil {
		return sql.ErrConnDone
	}
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return nil
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := q.db.ExecContext(ctx, "UPDATE thingsboard_device_tokens SET last_used_at=?, updated_at=? WHERE device_id=?", now, now, deviceID)
	return err
}
