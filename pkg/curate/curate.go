package curate

import (
	"bufio"
	"context"
	"fmt"
	ollama "github.com/ollama/ollama/api"
	"go.uber.org/zap"
	"os"
	"ozzysoft.net/wordle/pkg/log"
	"strings"
	"sync/atomic"
	"time"
)

type CurateResult struct {
	word    string
	exclude bool
	done    bool
}

func NewCurateResult(w string, exclude bool) CurateResult {
	return CurateResult{word: w, exclude: exclude}
}

func NewTerminalCurateResult() CurateResult {
	return CurateResult{done: true}
}

func Curate(ctx context.Context, client *ollama.Client, path string) error {
	logger := getLogger()
	processMax := -1
	maxConcurrency := 10

	logger.Infof("starting curation, process max (%d), concurrency max (%d)", processMax, maxConcurrency)

	curateResultChannel := make(chan CurateResult, 100)
	wordChannel := make(chan string, 100)
	worker := NewWordWorker(maxConcurrency, wordChannel, curateResultChannel, client)

	resultsDone := make(chan interface{})
	go handleResults(ctx, "data/curated.txt", "data/excluded.txt", curateResultChannel, resultsDone)

	go worker.processWordChannel(ctx)

	wordFile, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open word file (%s). %w", path, err)
	}
	defer doClose(wordFile)

	start := time.Now()
	count := 0
	scanner := bufio.NewScanner(wordFile)
loop:
	for scanner.Scan() {
		w := scanner.Text()
		w = strings.TrimSpace(w)
		count += 1
		if processMax < 0 || count <= processMax {
			logger.Debugf("read word (%s)", w)
			wordChannel <- w
		}

		select {
		case <-ctx.Done():
			logger.Infof("context closed, curation exiting")
			break loop
		default:
			// loop around
		}
	}

	logger.Infof("read file (%s), found words (%d)", path, count)
	close(wordChannel)

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read word file (%s). %w", path, err)
	}

	logger.Infof("waiting for curate results to be processed")
	select {
	case <-ctx.Done():
		logger.Infof("context closed, curation exiting")
	case <-resultsDone:
		logger.Infof("done channel closed")
	}

	elapsed := time.Since(start)
	avg := float64(elapsed.Milliseconds()) / float64(processMax)
	logger.Infof("elapsed (%s), count (%d), average elapsed milliseconds (%f)", elapsed, processMax, avg)
	return nil
}

func handleResults(ctx context.Context, path string, excludedPath string, c <-chan CurateResult, done chan<- interface{}) {
	logger := getLogger()
	logger.Infof("starting to handle results")

	curated, err := os.Create(path)
	if err != nil {
		logger.Errorf("failed to create curated file at path (%s)", path)
		return
	}
	defer doClose(curated)

	excluded, err := os.Create(excludedPath)
	if err != nil {
		logger.Errorf("failed to create excluded file at path (%s)", excludedPath)
		return
	}
	defer doClose(excluded)

	excludedCount := atomic.Int32{}
	curatedCount := atomic.Int32{}
	report := func() {
		logger.Infof("results processing completed, curated count (%d), excluded count (%d)", curatedCount.Load(), excludedCount.Load())
	}
	defer report()

	for {
		select {
		case <-ctx.Done():
			logger.Infof("context closed")
			return
		case result, open := <-c:
			if !open {
				logger.Infof("curate result channel closed, exiting")
				return
			}

			if result.done {
				// terminal result, exit the process.
				close(done)
				logger.Warnf("received terminal results message, exiting")
				return
			}

			if result.exclude {
				excludedCount.Add(1)
				_, err = excluded.WriteString(result.word + "\n")
				if err != nil {
					logger.Errorf("failed to write to excluded file, exiting")
					return
				}
			} else {
				curatedCount.Add(1)
				_, err = curated.WriteString(result.word + "\n")
				if err != nil {
					logger.Errorf("failed to write to curated file, exiting")
					return
				}
			}
		}
	}
}

func doClose(f *os.File) {
	err := f.Close()
	if err != nil {
		getLogger().Warnf("failed to close file (%s).  (%s)", f.Name(), err)
	}
}

func getLogger() *zap.SugaredLogger {
	return log.Get().Sugar().Named("curate")
}
