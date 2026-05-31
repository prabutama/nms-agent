package main

import (
	"path/filepath"
	"testing"
)

func TestReload_MissingPIDFails(t *testing.T) {
	tmp := t.TempDir()
	agentYml := writeAgentFiles(t, tmp, filepath.Join(tmp, "queue.db"), "thresholds: []\n")

	code := runReload([]string{"--config", agentYml})
	if code == 0 {
		t.Fatalf("expected non-zero exit code")
	}
}
