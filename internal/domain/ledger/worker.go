package ledger

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/Val-senseisama/payments/internal/domain/audit"
	"github.com/Val-senseisama/payments/types"
	"github.com/google/uuid"
)

type PostJob struct {
	TransactionID   uuid.UUID
	CompanyID       uuid.UUID
	UserID          uuid.UUID
	DebitAccountID  uuid.UUID
	CreditAccountID uuid.UUID
	Amount          int64
}

type PostWorker struct {
	store       types.LedgerStore
	txnStore    types.TransactionStore
	redisStore  types.RedisStore
	auditWorker *audit.Worker
	ctx         context.Context
	cancel      context.CancelFunc
}

func NewPostWorker(store types.LedgerStore, txnStore types.TransactionStore, redisStore types.RedisStore, auditWorker *audit.Worker) *PostWorker {
	ctx, cancel := context.WithCancel(context.Background())
	w := &PostWorker{
		store:       store,
		txnStore:    txnStore,
		redisStore:  redisStore,
		auditWorker: auditWorker,
		ctx:         ctx,
		cancel:      cancel,
	}
	go w.run()
	return w
}

func (w *PostWorker) Stop() {
	w.cancel()
}

func (w *PostWorker) Enqueue(ctx context.Context, job PostJob) error {
	payload, err := json.Marshal(job)
	if err != nil {
		return err
	}
	return w.redisStore.EnqueueLedgerJob(ctx, payload)
}

func (w *PostWorker) run() {
	for {
		select {
		case <-w.ctx.Done():
			log.Println("post worker: shutting down gracefully")
			return
		default:
		}

		ctx := w.ctx
		data, err := w.redisStore.DequeueLedgerJob(ctx, 3*time.Second)
		if err != nil {
			// Timeout or empty queue is normal, loop again
			continue
		}

		var job PostJob
		if err := json.Unmarshal(data, &job); err != nil {
			log.Printf("post worker: failed to unmarshal job: %v", err)
			continue
		}

		txnIDStr := job.TransactionID.String()

		// 1. Acquire Redis lock (10 second TTL)
		acquired, err := w.redisStore.AcquirePostingLock(ctx, txnIDStr, 10*time.Second)
		if err != nil || !acquired {
			log.Printf("post worker: failed to acquire lock for txn %s (already processing)", txnIDStr)
			continue
		}

		// 2. Perform atomic DB ledger post with company_id guard
		postErr := w.store.PostTransaction(ctx, job.TransactionID, job.CompanyID, job.DebitAccountID, job.CreditAccountID, job.Amount)

		// 3. Release Redis lock
		_ = w.redisStore.ReleasePostingLock(ctx, txnIDStr)

		if postErr != nil {
			log.Printf("post worker: error posting transaction %s: %v. Moving to DLQ.", txnIDStr, postErr)

			// Mark transaction as failed in DB
			_ = w.txnStore.UpdateTransactionStatus(ctx, job.TransactionID, types.TxnStatusFailed)

			// Push job to Dead-Letter Queue (DLQ) in Redis
			_ = w.redisStore.EnqueueDLQJob(ctx, data)

			// Broadcast failed status
			failMsg, _ := json.Marshal(map[string]string{
				"transaction_id": txnIDStr,
				"status":         string(types.TxnStatusFailed),
				"error":          postErr.Error(),
			})
			_ = w.redisStore.PublishTransactionUpdate(ctx, job.CompanyID.String(), failMsg)
			continue
		}

		// 4. Broadcast completion to Redis Pub/Sub
		updateMsg, _ := json.Marshal(map[string]string{
			"transaction_id": txnIDStr,
			"status":         string(types.TxnStatusCompleted),
		})
		_ = w.redisStore.PublishTransactionUpdate(ctx, job.CompanyID.String(), updateMsg)

		// 5. Send async audit log
		w.auditWorker.Send(&types.AuditLog{
			CompanyID:  &job.CompanyID,
			EntityType: "ledger",
			EntityID:   &job.TransactionID,
			Action:     "ledger.posted",
			ChangedBy:  &job.UserID,
		})
	}
}
