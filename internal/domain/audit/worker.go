package audit

import (
	"context"
	"log"

	"github.com/Val-senseisama/payments/types"
)

type Worker struct {
	ch    chan *types.AuditLog
	store types.AuditStore
}

func NewWorker(store types.AuditStore, bufSize int) *Worker {
	w := &Worker{
		ch:    make(chan *types.AuditLog, bufSize),
		store: store,
	}

	go w.run()
	return w
}

func (w *Worker) Send(logEntry *types.AuditLog) {
	select {
	case w.ch <- logEntry:
	default:
		log.Println("audit worker: buffer full, dropping log")
	}

}

func (w *Worker) run() {
	for logEntry := range w.ch {
		if err := w.store.CreateAuditLog(context.Background(), logEntry); err != nil {
			log.Println("audit worker: failed to write log: ", err)
		}
	}
}
