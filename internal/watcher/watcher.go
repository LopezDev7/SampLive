// Package watcher monitors project files and emits debounced change events.
package watcher

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Event is a debounced change notification for a file.
type Event struct {
	Path string
}

// Watcher watches a set of directories recursively for files with the
// configured extensions. Events are debounced because editors do messy
// things on save (write, rename, write again), and we want exactly one
// compile per save.
type Watcher struct {
	fw       *fsnotify.Watcher
	exts     map[string]bool
	debounce time.Duration
	events   chan Event
	done     chan struct{}
	mu       sync.Mutex
	last     map[string]time.Time
}

// New creates a Watcher. dirs are watched recursively; exts is a list of
// lower-case file extensions (e.g. ".pwn", ".inc") that trigger events.
func New(dirs []string, exts []string, debounce time.Duration) (*Watcher, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	for _, d := range dirs {
		if err := walkAdd(fw, d); err != nil {
			fw.Close()
			return nil, err
		}
	}
	em := make(map[string]bool, len(exts))
	for _, e := range exts {
		em[strings.ToLower(e)] = true
	}
	return &Watcher{
		fw:       fw,
		exts:     em,
		debounce: debounce,
		events:   make(chan Event, 64),
		done:     make(chan struct{}),
		last:     make(map[string]time.Time),
	}, nil
}

func walkAdd(fw *fsnotify.Watcher, root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return fw.Add(path)
		}
		return nil
	})
}

// Events returns the channel of debounced change notifications.
func (w *Watcher) Events() <-chan Event { return w.events }

// Start begins pumping file system events.
func (w *Watcher) Start() { go w.loop() }

// Close stops the watcher and releases resources.
func (w *Watcher) Close() error {
	close(w.done)
	return w.fw.Close()
}

func (w *Watcher) loop() {
	for {
		select {
		case <-w.done:
			return
		case ev, ok := <-w.fw.Events:
			if !ok {
				return
			}
			if w.matches(ev.Name) {
				w.pump(ev.Name)
			}
		case <-w.fw.Errors:
			// surface watcher errors on the events channel would pollute the
			// pipeline; they are dropped for the MVP.
		}
	}
}

func (w *Watcher) matches(path string) bool {
	return w.exts[strings.ToLower(filepath.Ext(path))]
}

// pump registers a pending delivery for path (debounce window refresh).
func (w *Watcher) pump(path string) {
	w.mu.Lock()
	_, pending := w.last[path]
	w.last[path] = time.Now()
	w.mu.Unlock()
	if !pending {
		go w.deliver(path)
	}
}

// deliver waits until path has been quiet for the debounce window, then
// emits a single event.
func (w *Watcher) deliver(path string) {
	for {
		time.Sleep(w.debounce)
		w.mu.Lock()
		last, ok := w.last[path]
		quiet := ok && time.Since(last) >= w.debounce
		if quiet {
			delete(w.last, path)
		}
		w.mu.Unlock()
		if quiet {
			select {
			case w.events <- Event{Path: path}:
				return
			case <-w.done:
				return
			}
		}
	}
}
