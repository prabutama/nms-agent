package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAdapterHealth_TerminalOK(t *testing.T) {
	tmp := t.TempDir()
	agentYml := writeAgentFiles(t, tmp, filepath.Join(tmp, "queue.db"), "thresholds: []\n")

	// Override adapters.yml to tui.
	if err := os.WriteFile(filepath.Join(tmp, "adapters.yml"), []byte("adapters:\n  active: tui\n  configs: {}\n"), 0o644); err != nil {
		t.Fatalf("WriteFile adapters: %v", err)
	}

	code := runAdapterHealth([]string{"--config", agentYml})
	if code != 0 {
		t.Fatalf("runAdapterHealth exit code=%d", code)
	}
}

func TestAdapterHealth_UnknownAdapterFails(t *testing.T) {
	tmp := t.TempDir()
	agentYml := writeAgentFiles(t, tmp, filepath.Join(tmp, "queue.db"), "thresholds: []\n")

	if err := os.WriteFile(filepath.Join(tmp, "adapters.yml"), []byte("adapters:\n  active: unknown_adapter\n  configs: {}\n"), 0o644); err != nil {
		t.Fatalf("WriteFile adapters: %v", err)
	}

	code := runAdapterHealth([]string{"--config", agentYml})
	if code == 0 {
		t.Fatalf("expected non-zero exit code")
	}
}
