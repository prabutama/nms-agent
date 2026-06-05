package configwatch

import (
	"testing"

	"github.com/fsnotify/fsnotify"
)

func TestDevicesWatcherShouldReload(t *testing.T) {
	w := NewDevicesWatcher("devices.d", 0)
	tests := []struct {
		name  string
		event fsnotify.Event
		want  bool
	}{
		{name: "create yml", event: fsnotify.Event{Name: "devices.d/a.yml", Op: fsnotify.Create}, want: true},
		{name: "write yaml", event: fsnotify.Event{Name: "devices.d/a.yaml", Op: fsnotify.Write}, want: true},
		{name: "rename yml", event: fsnotify.Event{Name: "devices.d/a.yml", Op: fsnotify.Rename}, want: true},
		{name: "remove txt", event: fsnotify.Event{Name: "devices.d/a.txt", Op: fsnotify.Remove}, want: false},
		{name: "chmod yml", event: fsnotify.Event{Name: "devices.d/a.yml", Op: fsnotify.Chmod}, want: false},
	}
	for _, tt := range tests {
		if got := w.shouldReload(tt.event); got != tt.want {
			t.Fatalf("%s: got %v want %v", tt.name, got, tt.want)
		}
	}
}
