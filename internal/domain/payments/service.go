package payments

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Val-senseisama/payments/internal/domain/audit"
	"github.com/Val-senseisama/payments/types"
)

type Service struct {
	paymentStore types.PaymentStore
	txnStore     types.TransactionStore
	redisStore   types.RedisStore
	pspAdapter   map[string]types.PSPAdapter
	auditWorker  *audit.Worker
}

func NewService(
	paymentStore types.PaymentStore,
	txnStore types.TransactionStore,
	redisStore types.RedisStore,
	pspAdapters []types.PSPAdapter,
	auditWorker *audit.Worker,
) *Service {
	adapterMap := make(map[string]types.PSPAdapter)

	for _, adapter := range pspAdapters {
		adapterMap[adapter.Name()] = adapter
	}

	return &Service{
		paymentStore: paymentStore,
		txnStore:     txnStore,
		redisStore:   redisStore,
		pspAdapter:   adapterMap,
		auditWorker:  auditWorker,
	}
}

func (s *Service) ProcessPayment(ctx context.Context, payload types.ProcessPaymentPayload) (*types.PaymentAttempt, error) {
	// lock with redis

	lockKey := payload.TransactionID.String()
	acquired, err := s.redisStore.AcquirePostingLock(ctx, lockKey, 15*time.Second)

	if err != nil {
		return nil, fmt.Errorf("Failed to check payment lock: %w", err)
	}

	if !acquired {
		return nil, fmt.Errorf("payment processing already in progress for transaction %s", payload.TransactionID)
	}

	defer s.redisStore.ReleasePostingLock(ctx, lockKey)

	// fetch and validate transaction
	txn, err := s.txnStore.GetTransactionByID(ctx, payload.TransactionID)
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch transaction: %w", err)
	}

	if txn.Status == types.TxnStatusCompleted {
		return nil, fmt.Errorf("Transaction %s is already completed", payload.TransactionID)
	}

	// get the adpater being used

	psp, ok := s.pspAdapter[payload.PSPName]

	if !ok {
		return nil, fmt.Errorf("unsupported psp provider: %s", payload.PSPName)
	}

	// create pending payment attempt

	initialAttempt := &types.PaymentAttempt{
		TransactionID: txn.ID,
		PSP:           payload.PSPName,
		Status:        types.PasPending,
		Response:      json.RawMessage([]byte("{}")),
		CreatedAt:     time.Now(),
		RetryCount:    0,
	}

	createdAttempt, err := s.paymentStore.CreatePaymentAttempt(ctx, initialAttempt)
	if err != nil {
		return nil, fmt.Errorf("Failed to create payment attempt: %w", err)
	}

	// create the charge request

	chargeRequest := types.ChargeRequest{
		Amount:    txn.Amount,
		Currency:  "NGN",
		Reference: txn.Reference,
		Metadata: map[string]any{
			"transaction_id": txn.ID.String(),
			"company_id":     txn.CompanyID.String(),
		},
	}

	// charge the card and update the status

	chargeResponse, err := psp.Charge(ctx, chargeRequest)
	if err != nil || !chargeResponse.Success {
		// handle the failure
		createdAttempt.Status = types.PasFailed
		createdAttempt.Response = chargeResponse.ResponseRaw

		if chargeResponse.ExternalReference != "" {
			createdAttempt.ExternalReference = chargeResponse.ExternalReference
		}

		updatedAttempt, err := s.paymentStore.UpdatePaymentAttemptStatus(ctx, createdAttempt)

		if err != nil {
			return nil, fmt.Errorf("failed to update payment attempt status: %w", err)
		}

		_ = s.txnStore.UpdateTransactionStatus(ctx, txn.ID, types.TxnStatusFailed)

		return updatedAttempt, fmt.Errorf("payment charge failed: %v", chargeResponse.ErrorMessage)
	}

	// handle success

	createdAttempt.Status = types.PasSuccess
	createdAttempt.ExternalReference = chargeResponse.ExternalReference
	createdAttempt.Response = chargeResponse.ResponseRaw

	updatedAttempt, err := s.paymentStore.UpdatePaymentAttemptStatus(ctx, createdAttempt)
	if err != nil {
		return nil, fmt.Errorf("failed to update attempt status to success: %w", err)
	}
	// Update Transaction Status to Completed
	if err := s.txnStore.UpdateTransactionStatus(ctx, txn.ID, types.TxnStatusCompleted); err != nil {
		return nil, fmt.Errorf("failed to complete transaction status: %w", err)
	}

	// put job in queue

	jobPayload, _ := json.Marshal(map[string]any{
		"transaction_id": txn.ID,
		"company_id":     txn.CompanyID,
		"amount":         txn.Amount,
	})

	_ = s.redisStore.EnqueueLedgerJob(ctx, jobPayload)

	// publish on sse via redis

	eventPayload, _ := json.Marshal(map[string]any{
		"event":          "payment.succeeded",
		"transaction_id": txn.ID,
		"company_id":     txn.CompanyID,
		"amount":         txn.Amount,
	})
	_ = s.redisStore.PublishTransactionUpdate(ctx, txn.CompanyID.String(), eventPayload)
	return updatedAttempt, nil

}
