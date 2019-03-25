package cleanuphandler

import (
	"context"
	"io"
	"log"
	"os"
	"os/signal"
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

func AddCleanupHandlers(newHandlers ...CleanupHandler) {
	if len(newHandlers) != 0 {
		handlers = append(handlers, newHandlers...)
	}
}

func SetLogger(l *log.Logger) {
	if l != nil {
		logger = l
	}
}

func init() {
	ctx, cancel := context.WithCancel(context.Background())
	go worker(ctx)
	cancelWorker = cancel
}

func worker(ctx context.Context) {
	var sigChan = make(chan os.Signal)
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
	for _, v := range handlers {
		v(logger)
	}
	done <- struct{}{}
}
