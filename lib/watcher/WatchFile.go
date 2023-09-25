// Package watcher that handles file renames
package watcher

import (
	"log/slog"
	"time"

	"github.com/fsnotify/fsnotify"
)

// debounce timer that invokes the callback after no more changes are received
const watcherDebounceDelay = 50 // msec

// WatchFile is a resilient file watcher that handles file renames
// Special features:
//  1. This debounces multiple quick changes before invoking the callback
//  2. After the callback, resubscribe to the file to handle file renames that change the file inode
//     path to watch
//     handler to invoke on change
//
// This returns the fsnotify watcher. Close it when done.
func WatchFile(path string,
	handler func() error) (*fsnotify.Watcher, error) {

	watcher, _ := fsnotify.NewWatcher()
	// The callback timer debounces multiple changes to the config file
	callbackTimer := time.AfterFunc(0, func() {
		//slog.Info("WatchFile.Watch: trigger, invoking callback...")
		_ = handler()

		// file renames change the inode of the filename, resubscribe
		_ = watcher.Remove(path)
		err := watcher.Add(path)
		if err != nil {
			slog.Error("failed adding file to watch", "filename", path, "err", err)
		}
	})
	callbackTimer.Stop() // don't start yet

	err := watcher.Add(path)
	if err != nil {
		slog.Error("unable to watch for changes", "err", err)
		return watcher, err
	}
	//slog.Info("WatchFile added", "path", path)

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					slog.Debug("No more events. Ending watch for file.",
						"path", path, "event", event.String())
					callbackTimer.Stop()
					return
				}
				// don't really care what the change it, 50msec after the last event the file will reload
				//slog.Info("File Event", "event", event.String(), "file", event.ID)
				callbackTimer.Reset(time.Millisecond * watcherDebounceDelay)
			case err2, ok := <-watcher.Errors:
				if !ok && err2 != nil {
					slog.Error("Unexpected error", "err", err2)
					return
				}
				// end of watcher.
				//slog.Error("Error", "err",err2)
			}
		}
	}()
	return watcher, nil
}
