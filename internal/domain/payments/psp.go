package payments

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Val-senseisama/payments/types"
	"github.com/google/uuid"
)

type MockPSPAdapter struct {
	name string
}

func NewMockPSPAdapter(name string) *MockPSPAdapter {
	if name == "" {
		name = "mock"
	}
	return &MockPSPAdapter{name: name}
}

func (m *MockPSPAdapter) Name() string {
	return m.name
}

func (m *MockPSPAdapter) Charge(ctx context.Context, req types.ChargeRequest) (*types.ChargeResponse, error) {
	// Simulate gateway failure if reference contains "fail"
	if strings.Contains(strings.ToLower(req.Reference), "fail") {
		rawErr, _ := json.Marshal(map[string]any{
			"status":  "failed",
			"code":    "insufficient_funds",
			"message": "Transaction declined by issuer",
		})
		errMsg := "Payment declined by issuer"
		return &types.ChargeResponse{
			Success:           false,
			ExternalReference: fmt.Sprintf("MOCK-FAIL-%s", uuid.New().String()[:8]),
			ResponseRaw:       rawErr,
			ErrorMessage:      &errMsg,
		}, nil
	}

	// Default Success Response
	rawSuccess, _ := json.Marshal(map[string]any{
		"status":     "success",
		"gateway_id": fmt.Sprintf("MOCK-TXN-%s", uuid.New().String()),
		"amount":     req.Amount,
		"currency":   req.Currency,
	})

	return &types.ChargeResponse{
		Success:           true,
		ExternalReference: fmt.Sprintf("MOCK-REF-%s", uuid.New().String()[:8]),
		ResponseRaw:       rawSuccess,
	}, nil
}
