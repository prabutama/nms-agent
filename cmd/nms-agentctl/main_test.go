package main

import (
	"os"
	"testing"
)

func TestIsHelpArg(t *testing.T) {
	tests := []struct {
		arg    string
		expect bool
	}{
		{"--help", true},
		{"-h", true},
		{"help", true},
		{"validate", false},
		{"device", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.arg, func(t *testing.T) {
			if got := isHelpArg(tt.arg); got != tt.expect {
				t.Fatalf("isHelpArg(%q) = %v, want %v", tt.arg, got, tt.expect)
			}
		})
	}
}

func TestRunDevice_Help(t *testing.T) {
	oldStderr := os.Stderr
	defer func() { os.Stderr = oldStderr }()

	r, w, _ := os.Pipe()
	os.Stderr = w

	code := runDevice([]string{"--help"})
	w.Close()

	os.Stderr = oldStderr

	if code != 0 {
		t.Fatalf("runDevice([]string{\"--help\"}) exit code=%d, want 0", code)
	}

	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	if len(output) == 0 {
		t.Fatal("expected help output, got empty string")
	}
}

func TestRunAdapter_Help(t *testing.T) {
	oldStderr := os.Stderr
	defer func() { os.Stderr = oldStderr }()

	r, w, _ := os.Pipe()
	os.Stderr = w

	code := runAdapter([]string{"--help"})
	w.Close()

	os.Stderr = oldStderr

	if code != 0 {
		t.Fatalf("runAdapter([]string{\"--help\"}) exit code=%d, want 0", code)
	}

	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	if len(output) == 0 {
		t.Fatal("expected help output, got empty string")
	}
}

func TestRunDiscovery_Help(t *testing.T) {
	oldStderr := os.Stderr
	defer func() { os.Stderr = oldStderr }()

	r, w, _ := os.Pipe()
	os.Stderr = w

	code := runDiscovery([]string{"--help"})
	w.Close()

	os.Stderr = oldStderr

	if code != 0 {
		t.Fatalf("runDiscovery([]string{\"--help\"}) exit code=%d, want 0", code)
	}

	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	if len(output) == 0 {
		t.Fatal("expected help output, got empty string")
	}
}

func TestRunThreshold_Help(t *testing.T) {
	oldStderr := os.Stderr
	defer func() { os.Stderr = oldStderr }()

	r, w, _ := os.Pipe()
	os.Stderr = w

	code := runThreshold([]string{"--help"})
	w.Close()

	os.Stderr = oldStderr

	if code != 0 {
		t.Fatalf("runThreshold([]string{\"--help\"}) exit code=%d, want 0", code)
	}

	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	if len(output) == 0 {
		t.Fatal("expected help output, got empty string")
	}
}
