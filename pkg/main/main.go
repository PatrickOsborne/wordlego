package main

import (
	"context"
	"errors"
	"go.uber.org/zap"
	"os"
	"os/signal"
	"ozzysoft.net/wordle/pkg/curate"
	"ozzysoft.net/wordle/pkg/llama"
	"ozzysoft.net/wordle/pkg/log"
	"syscall"
	"time"
)

const logConfigPath = "config/logging/logging.yaml"

func init() {
	log.SetFromFile(logConfigPath)
}

func main() {
	logger := log.Get().Sugar().Named("main")
	logger.Infof("running")

	ctx := log.WithCtx(context.Background(), logger.Desugar())
	ctx, contextCancelFunc := context.WithCancel(ctx)

	go log.WatchOrExit(ctx, logConfigPath)

	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	go cancelContextOnSignal(ctx, signalChannel, contextCancelFunc, logger)

	client, err := llama.CreateClient()
	if err != nil {
		contextCancelFunc()
	} else {
		err := curate.Curate(ctx, client, "data/words_five.txt")
		if err != nil {
		}
		contextCancelFunc()
	}

	// block to exit signal
	logger.Infof("waiting for complete")
	select {
	case <-ctx.Done():
		err := ctx.Err()
		if errors.Is(err, context.Canceled) {
			logger.Infof("context canceled")
		} else {
			logger.With(zap.Error(err)).Errorf("context error")
		}
	}

	time.Sleep(100 * time.Millisecond)
	logger.Infof("exiting")
}

func cancelContextOnSignal(ctx context.Context, channel chan os.Signal, cancelFunc context.CancelFunc, logger *zap.SugaredLogger) {
	select {
	case sig := <-channel:
		logger.Infof("signal encountered (%s)", sig)
		cancelFunc()
	case <-ctx.Done():
		err := ctx.Err()
		if errors.Is(err, context.Canceled) {
			logger.Infof("context canceled")
		} else {
			logger.With(zap.Error(err)).Errorf("context error")
		}
	}
}
