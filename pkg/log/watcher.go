package log

import (
	"context"
	"github.com/fsnotify/fsnotify"
	"go.uber.org/zap"
	"os"
)

const logWatcherName = "log-watcher"

// WatchOrExit setups up watch of the log config path provided.  If the watcher fails to set up properly an error will be returned and the
// process will exit with exit value 128.  The method will block until the context is canceled.
func WatchOrExit(ctx context.Context, logConfigFilePath string) {
	if err := Watch(ctx, logConfigFilePath); err != nil {
		Get().Named(logWatcherName).Sugar().With(zap.Error(err)).Errorf("failed to setup log file watching")
		os.Exit(128)
	}
}

func Watch(ctx context.Context, logConfigFilePath string) error {
	logger := defaultLogger.Named(logWatcherName).Sugar()
	logger.Infof("setting up log watcher for config path (%s)", logConfigFilePath)
	// create a new file watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logger.With(zap.Error(err)).Errorf("failed to create log file watcher")
		return err
	}

	closeFunc := func() {
		if err := watcher.Close(); err != nil {
			logger.Warnf("failed to close log watcher")
		}
	}
	defer closeFunc()

	watcherFunc := func() {
		for {
			select {
			// watch for events
			case event := <-watcher.Events:
				logger.Infof("log watch event, location (%s), op (%s)", event.Name, event.Op)
				newLogger, err := CreateLoggerFromFile(event.Name)
				if err != nil {
					logger.Warnf("after log file change event, unable to create new logger from config defined at location (%s)", event.Name)
				} else {
					setLogger(newLogger)
				}

			// watch for errors
			case err := <-watcher.Errors:
				logger.With(zap.Error(err)).Errorf("log watch error event")

			case <-ctx.Done():
				logger.Infof("log watcher event loop exiting")
				return
			}
		}
	}
	go watcherFunc()

	if err := watcher.Add(logConfigFilePath); err != nil {
		logger.With(zap.Error(err)).Errorf("failed to watch log file (%s)", logConfigFilePath)
	}

	select {
	case <-ctx.Done():
		logger.Infof("log watcher exiting")
	}
	return nil
}
