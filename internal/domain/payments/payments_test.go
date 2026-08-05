package payments_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Val-senseisama/payments/internal/domain/payments"
	"github.com/Val-senseisama/payments/types"
	"github.com/google/uuid"
)

func TestMockPSPAdapter_Charge_Success(t *testing.T) {
	adapter := payments.NewMockPSPAdapter("mock")

	req := types.ChargeRequest{
		Amount:    500000,
		Currency:  "NGN",
		Reference: "REF-SUCCESS-001",
	}

	res, err := adapter.Charge(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !res.Success {
		t.Errorf("expected charge to succeed, got failure")
	}

	if !strings.HasPrefix(res.ExternalReference, "MOCK-REF-") {
		t.Errorf("expected external ref prefix MOCK-REF-, got %s", res.ExternalReference)
	}
}

func TestMockPSPAdapter_Charge_Failure(t *testing.T) {
	adapter := payments.NewMockPSPAdapter("mock")

	req := types.ChargeRequest{
		Amount:    500000,
		Currency:  "NGN",
		Reference: "REF-FAIL-CARD-DECLINED",
	}

	res, err := adapter.Charge(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Success {
		t.Errorf("expected charge to fail when ref contains fail, got success")
	}

	if res.ErrorMessage == nil || *res.ErrorMessage == "" {
		t.Errorf("expected error message on failed charge")
	}
}

// Mock Stores for Unit Testing Payment Service

type mockPaymentStore struct {
	attempts map[uuid.UUID]*types.PaymentAttempt
}

func newMockPaymentStore() *mockPaymentStore {
	return &mockPaymentStore{attempts: make(map[uuid.UUID]*types.PaymentAttempt)}
}

func (m *mockPaymentStore) CreatePaymentAttempt(ctx context.Context, attempt *types.PaymentAttempt) (*types.PaymentAttempt, error) {
	if attempt.ID == uuid.Nil {
		attempt.ID = uuid.New()
	}
	m.attempts[attempt.ID] = attempt
	return attempt, nil
}

func (m *mockPaymentStore) UpdatePaymentAttemptStatus(ctx context.Context, attempt *types.PaymentAttempt) (*types.PaymentAttempt, error) {
	m.attempts[attempt.ID] = attempt
	return attempt, nil
}

func (m *mockPaymentStore) GetPaymentAttemptsByTransactionID(ctx context.Context, txnID uuid.UUID) ([]*types.PaymentAttempt, error) {
	var result []*types.PaymentAttempt
	for _, att := range m.attempts {
		if att.TransactionID == txnID {
			result = append(result, att)
		}
	}
	return result, nil
}

type mockTxnStore struct {
	txns map[uuid.UUID]*types.Transaction
}

func newMockTxnStore() *mockTxnStore {
	return &mockTxnStore{txns: make(map[uuid.UUID]*types.Transaction)}
}

func (m *mockTxnStore) CreateTransaction(ctx context.Context, companyID, createdBy uuid.UUID, ref string, tType types.TransactionType, amount int64) (*types.Transaction, error) {
	txn := &types.Transaction{
		ID:        uuid.New(),
		CompanyID: companyID,
		Reference: ref,
		Type:      tType,
		Amount:    amount,
		Status:    types.TxnStatusPending,
		CreatedBy: createdBy,
	}
	m.txns[txn.ID] = txn
	return txn, nil
}

func (m *mockTxnStore) GetTransactionByID(ctx context.Context, id uuid.UUID) (*types.Transaction, error) {
	txn, ok := m.txns[id]
	if !ok {
		return nil, types.ErrNotFound
	}
	return txn, nil
}

func (m *mockTxnStore) GetTransactionsByCompany(ctx context.Context, companyID uuid.UUID) ([]*types.Transaction, error) {
	var list []*types.Transaction
	for _, t := range m.txns {
		if t.CompanyID == companyID {
			list = append(list, t)
		}
	}
	return list, nil
}

func (m *mockTxnStore) UpdateTransactionStatus(ctx context.Context, id uuid.UUID, status types.TransactionStatus) error {
	if txn, ok := m.txns[id]; ok {
		txn.Status = status
	}
	return nil
}

type mockRedisStore struct{}

func (m *mockRedisStore) SaveRefreshToken(ctx context.Context, userID string, tokenID string, ttl time.Duration) error {
	return nil
}
func (m *mockRedisStore) GetRefreshToken(ctx context.Context, userID string) (string, error) {
	return "", nil
}
func (m *mockRedisStore) DeleteRefreshToken(ctx context.Context, userID string) error { return nil }
func (m *mockRedisStore) SetIdempotencyKey(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return nil
}
func (m *mockRedisStore) GetIdempotencyKey(ctx context.Context, key string) ([]byte, error) {
	return nil, nil
}
func (m *mockRedisStore) AcquirePostingLock(ctx context.Context, txnID string, ttl time.Duration) (bool, error) {
	return true, nil
}
func (m *mockRedisStore) ReleasePostingLock(ctx context.Context, txnID string) error { return nil }
func (m *mockRedisStore) PublishTransactionUpdate(ctx context.Context, companyID string, payload []byte) error {
	return nil
}
func (m *mockRedisStore) EnqueueLedgerJob(ctx context.Context, payload []byte) error { return nil }
func (m *mockRedisStore) DequeueLedgerJob(ctx context.Context, timeout time.Duration) ([]byte, error) {
	return nil, nil
}
func (m *mockRedisStore) EnqueueDLQJob(ctx context.Context, payload []byte) error { return nil }
func (m *mockRedisStore) SubscribeTransactionUpdates(ctx context.Context, companyID string) (<-chan string, func()) {
	ch := make(chan string)
	close(ch)
	return ch, func() {}
}

func TestPaymentService_ProcessPayment_Success(t *testing.T) {
	payStore := newMockPaymentStore()
	txnStore := newMockTxnStore()
	redisStore := &mockRedisStore{}
	mockPSP := payments.NewMockPSPAdapter("mock")

	companyID := uuid.New()
	userID := uuid.New()
	txn, _ := txnStore.CreateTransaction(context.Background(), companyID, userID, "TXN-TEST-1", types.TxnPaymentIn, 100000)

	service := payments.NewService(payStore, txnStore, redisStore, []types.PSPAdapter{mockPSP}, nil)

	payload := types.ProcessPaymentPayload{
		TransactionID: txn.ID,
		PSPName:       "mock",
	}

	attempt, err := service.ProcessPayment(context.Background(), payload)
	if err != nil {
		t.Fatalf("unexpected error processing payment: %v", err)
	}

	if attempt.Status != types.PasSuccess {
		t.Errorf("expected attempt status PasSuccess, got %s", attempt.Status)
	}

	updatedTxn, _ := txnStore.GetTransactionByID(context.Background(), txn.ID)
	if updatedTxn.Status != types.TxnStatusCompleted {
		t.Errorf("expected transaction status TxnStatusCompleted, got %s", updatedTxn.Status)
	}
}
