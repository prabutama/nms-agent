package queue

import (
	"context"
	"path/filepath"
	"testing"
)

func TestSQLiteQueueThingsBoardTokens(t *testing.T) {
	q, err := OpenSQLite(filepath.Join(t.TempDir(), "nms-agent.db"))
	if err != nil {
		t.Fatalf("OpenSQLite: %v", err)
	}
	defer q.Close()

	ctx := context.Background()
	if token, ok, err := q.GetThingsBoardToken(ctx, "d1"); err != nil || ok || token != "" {
		t.Fatalf("empty lookup token=%q ok=%v err=%v", token, ok, err)
	}
	if err := q.SaveThingsBoardToken(ctx, "d1", "token-1"); err != nil {
		t.Fatalf("SaveThingsBoardToken: %v", err)
	}
	if token, ok, err := q.GetThingsBoardToken(ctx, "d1"); err != nil || !ok || token != "token-1" {
		t.Fatalf("lookup token=%q ok=%v err=%v", token, ok, err)
	}
	if err := q.SaveThingsBoardToken(ctx, "d1", "token-2"); err != nil {
		t.Fatalf("SaveThingsBoardToken update: %v", err)
	}
	if token, ok, err := q.GetThingsBoardToken(ctx, "d1"); err != nil || !ok || token != "token-2" {
		t.Fatalf("lookup updated token=%q ok=%v err=%v", token, ok, err)
	}
	if err := q.MarkThingsBoardTokenUsed(ctx, "d1"); err != nil {
		t.Fatalf("MarkThingsBoardTokenUsed: %v", err)
	}
}
