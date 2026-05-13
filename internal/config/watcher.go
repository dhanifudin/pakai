package config

import (
	"log/slog"
	"time"

	"github.com/fsnotify/fsnotify"
)

type Watcher struct {
	stopCh chan struct{}
}

func NewWatcher() *Watcher {
	return &Watcher{
		stopCh: make(chan struct{}),
	}
}

func (w *Watcher) Start() {
	watchPath := resolveConfigPath()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		slog.Warn("fsnotify unavailable, falling back to 5s poll", "error", err)
		go w.fallbackPoll()
		return
	}

	if err := watcher.Add(watchPath); err != nil {
		slog.Warn("fsnotify watch failed, falling back to 5s poll", "path", watchPath, "error", err)
		watcher.Close()
		go w.fallbackPoll()
		return
	}

	go w.watchLoop(watcher)
}

func (w *Watcher) Stop() {
	close(w.stopCh)
}

func (w *Watcher) IsRunning() bool {
	select {
	case <-w.stopCh:
		return false
	default:
		return true
	}
}

func (w *Watcher) watchLoop(watcher *fsnotify.Watcher) {
	defer watcher.Close()

	for {
		select {
		case <-w.stopCh:
			return
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Op&(fsnotify.Write|fsnotify.Create) != 0 {
				invalidateConfig()
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			slog.Warn("fsnotify error", "error", err)
		}
	}
}

func (w *Watcher) fallbackPoll() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopCh:
			return
		case <-ticker.C:
			invalidateConfig()
		}
	}
}
