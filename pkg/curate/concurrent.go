package curate

import (
	"context"
	ollama "github.com/ollama/ollama/api"
	"sync/atomic"
	"time"
)

type WordWorker struct {
	maxConcurrency  int
	wordChannel     <-chan string
	resultChannel   chan<- CurateResult
	client          *ollama.Client
	reportFrequency int32

	concurrentChannel chan interface{}
	readComplete      atomic.Bool
	inProcess         atomic.Int32

	startTime    time.Time
	processCount atomic.Int32
}

func NewWordWorker(maxConcurrency int, wordChannel <-chan string, resultChannel chan<- CurateResult, client *ollama.Client) *WordWorker {
	return &WordWorker{
		maxConcurrency: maxConcurrency,
		wordChannel:    wordChannel, resultChannel: resultChannel, client: client,
		reportFrequency:   200,
		concurrentChannel: setupConcurrentChannel(maxConcurrency),
	}
}

func (w *WordWorker) isComplete() bool {
	return w.readComplete.Load() && w.inProcess.Load() <= 0
}

func (w *WordWorker) markReadComplete() {
	w.readComplete.Store(true)
}

func (w *WordWorker) incrementInProcess() {
	w.inProcess.Add(1)
}

func (w *WordWorker) decrementInProcess() {
	w.inProcess.Add(-1)
}

func (w *WordWorker) incrementProcessCount() {
	v := w.processCount.Add(1)
	if v%w.reportFrequency == 0 {
		elapsed := time.Since(w.startTime)
		getLogger().Infof("words processed (%d), elapsed (%s), average (%f)", v, elapsed, elapsed.Milliseconds()/int64(v))
	}
}

func (w *WordWorker) sendTerminalMessageToResultProcesserIfNecessary() {
	if w.isComplete() {
		w.resultChannel <- NewTerminalCurateResult()
		getLogger().Warnf("terminal result message sent")
	}
}

func (w *WordWorker) processWordChannel(ctx context.Context) bool {
	logger := getLogger()
	defer logger.Warnf("word channel processing complete")

	logger.Infof("starting word processing")
	w.startTime = time.Now()

	for {
		select {
		case <-ctx.Done():
			w.markReadComplete()
			return false
		case word, open := <-w.wordChannel:
			if !open {
				logger.Infof("word channel closed")
				w.markReadComplete()
				// handle cleanup
				return true
			} else {
				logger.Debugf("word from channel (%s)", word)
				w.incrementInProcess()
				w.processWord(ctx, word)
			}
		}
	}

	return true
}

func (w *WordWorker) processWord(ctx context.Context, word string) bool {
	select {
	case <-ctx.Done():
		w.markReadComplete()
		w.decrementInProcess()
		return false
	case token := <-w.concurrentChannel:
		go w.curateWord(ctx, token, word)
		return true
	}

	return false
}

func (w *WordWorker) curateWord(ctx context.Context, token interface{}, word string) bool {
	logger := getLogger()

	writeTokenToChannel := func() {
		w.concurrentChannel <- token
	}
	defer writeTokenToChannel()

	start := time.Now()
	rareOrObscure := IsWordRareOrObscure(ctx, w.client, word)
	elapsed := time.Since(start)
	logger.Debugf("word (%s), is rare or obscure (%t), elapsed (%s)", word, rareOrObscure, elapsed)

	result := NewCurateResult(word, rareOrObscure)
	w.resultChannel <- result

	w.incrementProcessCount()
	w.decrementInProcess()
	w.sendTerminalMessageToResultProcesserIfNecessary()

	return rareOrObscure
}

func setupConcurrentChannel(size int) chan interface{} {
	c := make(chan interface{}, size)
	populateChan(c, size)
	return c
}

func populateChan(c chan<- interface{}, size int) {
	for i := range size {
		c <- i
	}
}
