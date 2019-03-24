package cleanuphandler

import (
	"context"
	"io"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

var logger = log.New(func() io.Writer { f, _ := os.OpenFile(os.DevNull, os.O_WRONLY, 0666); return f }(), "", 0)

var sigs = []os.Signal{syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL, syscall.SIGABRT, syscall.SIGQUIT}

var done = make(chan struct{})

var cancelWorker context.CancelFunc

type CleanupHandler func(*log.Logger)

var handlers []CleanupHandler

func HandleSignals(signal ...os.Signal) { sigs = signal; reloadWorker() }

func Wait() { <-done }

func AddCleanupHandlers(newHandlers ...CleanupHandler) { handlers = append(handlers, newHandlers...) }

func init() {
	ctx, cancel := context.WithCancel(context.Background())
	go worker(ctx)
	cancelWorker = cancel
}

func worker(ctx context.Context) {
	var sigChan chan os.Signal
	signal.Ignore(sigs...)
	signal.Notify(sigChan, sigs...)

	select {
	case sig := <-sigChan:
		logger.Println("catch signal :", sig, ", exit")
		executeHandlers()
		logger.Println("cleaning up finished")
		return
	case <-ctx.Done():
		return
	}
}

func reloadWorker() {
	cancelWorker()
	ctx, cancel := context.WithCancel(context.Background())
	go worker(ctx)
	cancelWorker = cancel
}

func executeHandlers() {
	wg := new(sync.WaitGroup)
	work := func(handler CleanupHandler) {
		wg.Add(1)
		defer wg.Done()
		handler(logger)
	}
	for _, v := range handlers {
		go work(v)
	}
	wg.Wait()
}