package payments

import (
	"net/http"

	"github.com/Val-senseisama/payments/internal/common"
	"github.com/Val-senseisama/payments/internal/common/auth"
	"github.com/Val-senseisama/payments/types"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	service      *Service
	paymentStore types.PaymentStore
}

func NewHandler(service *Service, paymentStore types.PaymentStore) *Handler {
	return &Handler{
		service:      service,
		paymentStore: paymentStore,
	}
}

func (h *Handler) RegisterRoutes(router chi.Router) {
	router.Post("/process", h.HandleProcessPayment)
	router.Get("/attempts/{txn_id}", h.HandleGetPaymentAttempts)
}

func (h *Handler) HandleProcessPayment(w http.ResponseWriter, r *http.Request) {
	authUser, err := auth.GetUserFromContext(r.Context())
	if err != nil {
		common.WriteErrorJSON(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	if authUser.CompanyID == nil {
		common.WriteErrorJSON(w, http.StatusForbidden, "User does not belong to a company")
		return
	}

	var payload types.ProcessPaymentPayload
	if err := common.ReadJSON(r, &payload); err != nil {
		common.WriteErrorJSON(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := common.Validate.Struct(payload); err != nil {
		common.WriteErrorJSON(w, http.StatusBadRequest, "Validation error: "+err.Error())
		return
	}

	attempt, err := h.service.ProcessPayment(r.Context(), payload)
	if err != nil {
		if attempt != nil {
			// Payment charge failed but attempt record was saved
			common.WriteJSON(w, http.StatusPaymentRequired, map[string]any{
				"error":   err.Error(),
				"attempt": attempt,
			})
			return
		}
		common.WriteErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}

	common.WriteJSON(w, http.StatusOK, attempt)
}

func (h *Handler) HandleGetPaymentAttempts(w http.ResponseWriter, r *http.Request) {
	_, err := auth.GetUserFromContext(r.Context())
	if err != nil {
		common.WriteErrorJSON(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	txnIDParam := chi.URLParam(r, "txn_id")
	txnID, err := uuid.Parse(txnIDParam)
	if err != nil {
		common.WriteErrorJSON(w, http.StatusBadRequest, "Invalid transaction_id parameter")
		return
	}

	attempts, err := h.paymentStore.GetPaymentAttemptsByTransactionID(r.Context(), txnID)
	if err != nil {
		common.WriteErrorJSON(w, http.StatusInternalServerError, "Failed to retrieve payment attempts")
		return
	}

	common.WriteJSON(w, http.StatusOK, attempts)
}
