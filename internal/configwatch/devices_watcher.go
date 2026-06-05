package configwatch

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// DevicesWatcher observes changes in a devices directory and triggers
// a callback when a relevant device file (`.yml` / `.yaml`) is created,
// modified, or removed.
type DevicesWatcher struct {
	mu       sync.Mutex
	dir      string
	watcher  *fsnotify.Watcher
	ch       chan struct{}
	cancel   chan struct{}
	debounce time.Duration
	running  bool
}

// NewDevicesWatcher creates a watcher for the given devices directory.
// Events are debounced to avoid multiple reloads from a single atomic write.
func NewDevicesWatcher(devicesDir string, debounce time.Duration) *DevicesWatcher {
	if debounce <= 0 {
		debounce = 1 * time.Second
	}
	return &DevicesWatcher{
		dir:      devicesDir,
		ch:       make(chan struct{}, 1),
		cancel:   make(chan struct{}),
		debounce: debounce,
	}
}

// Start begins watching the devices directory.
func (w *DevicesWatcher) Start() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.running {
		return nil
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("create watcher: %w", err)
	}
	if err := os.MkdirAll(w.dir, 0o755); err != nil {
		watcher.Close()
		return fmt.Errorf("ensure watch directory %s: %w", w.dir, err)
	}

	if err := watcher.Add(w.dir); err != nil {
		watcher.Close()
		return fmt.Errorf("watch directory %s: %w", w.dir, err)
	}

	w.watcher = watcher
	w.running = true

	go w.run()
	return nil
}

func (w *DevicesWatcher) run() {
	var timer *time.Timer
	var timerCh <-chan time.Time
	for {
		select {
		case <-w.cancel:
			if timer != nil {
				timer.Stop()
			}
			return
		case event, ok := <-w.watcher.Events:
			if !ok {
				if timer != nil {
					timer.Stop()
				}
				return
			}
			if w.shouldReload(event) {
				if timer == nil {
					timer = time.NewTimer(w.debounce)
					timerCh = timer.C
					continue
				}
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				timer.Reset(w.debounce)
				timerCh = timer.C
			}
		case <-timerCh:
			select {
			case w.ch <- struct{}{}:
			default:
			}
			if timer != nil {
				timer.Stop()
			}
			timer = nil
			timerCh = nil
		case err, ok := <-w.watcher.Errors:
			if !ok {
				if timer != nil {
					timer.Stop()
				}
				return
			}
			fmt.Fprintf(os.Stderr, "devices watcher error: %v\n", err)
		}
	}
}

func (w *DevicesWatcher) shouldReload(event fsnotify.Event) bool {
	name := filepath.Base(event.Name)
	ext := strings.ToLower(filepath.Ext(name))
	if ext != ".yml" && ext != ".yaml" {
		return false
	}
	return event.Has(fsnotify.Create) ||
		event.Has(fsnotify.Write) ||
		event.Has(fsnotify.Remove) ||
		event.Has(fsnotify.Rename)
}

// Changes returns a channel that fires when a relevant device file changes.
func (w *DevicesWatcher) Changes() <-chan struct{} {
	return w.ch
}

// Stop closes the watcher and stops the background goroutine.
func (w *DevicesWatcher) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.running {
		return
	}
	close(w.cancel)
	w.watcher.Close()
	w.running = false
}
