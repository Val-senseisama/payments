package ledger

import (
	"net/http"

	"github.com/Val-senseisama/payments/internal/common"
	"github.com/Val-senseisama/payments/internal/common/auth"
	"github.com/Val-senseisama/payments/types"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	store      types.LedgerStore
	txnStore   types.TransactionStore
	postWorker *PostWorker
}

func NewHandler(store types.LedgerStore, txnStore types.TransactionStore, postWorker *PostWorker) *Handler {
	return &Handler{
		store:      store,
		txnStore:   txnStore,
		postWorker: postWorker,
	}
}

func (h *Handler) RegisterRoutes(router chi.Router) {
	router.Post("/post", h.PostLedger)
	router.Get("/{transaction_id}/entries", h.GetEntries)
}

func (h *Handler) PostLedger(w http.ResponseWriter, r *http.Request) {
	authUser, err := auth.GetUserFromContext(r.Context())
	if err != nil {
		common.WriteErrorJSON(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	if authUser.CompanyID == nil {
		common.WriteErrorJSON(w, http.StatusForbidden, "user does not belong to a company")
		return
	}

	if !auth.RequestAccess(authUser, *authUser.CompanyID, types.RoleAdmin, types.RoleFinance) {
		common.WriteErrorJSON(w, http.StatusForbidden, "user is not authorized to post ledger entries")
		return
	}

	var payload types.PostLedgerPayload
	if err := common.ReadJSON(r, &payload); err != nil {
		common.WriteErrorJSON(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if err := common.Validate.Struct(payload); err != nil {
		common.WriteErrorJSON(w, http.StatusBadRequest, "Missing or invalid fields: "+err.Error())
		return
	}

	// Verify transaction exists and belongs to this company
	txn, err := h.txnStore.GetTransactionByID(r.Context(), payload.TransactionID)
	if err != nil {
		common.WriteErrorJSON(w, http.StatusNotFound, "Transaction not found")
		return
	}

	if txn.CompanyID != *authUser.CompanyID {
		common.WriteErrorJSON(w, http.StatusForbidden, "Transaction does not belong to your company")
		return
	}

	if txn.Status == types.TxnStatusCompleted {
		common.WriteErrorJSON(w, http.StatusBadRequest, "Transaction has already been posted")
		return
	}

	err = h.postWorker.Enqueue(r.Context(), PostJob{
		TransactionID:   txn.ID,
		CompanyID:       txn.CompanyID,
		UserID:          authUser.UserID,
		DebitAccountID:  payload.DebitAccount,
		CreditAccountID: payload.CreditAccount,
		Amount:          txn.Amount,
	})

	if err != nil {
		common.WriteErrorJSON(w, http.StatusInternalServerError, "Failed to queue ledger posting: "+err.Error())
		return
	}

	common.WriteJSON(w, http.StatusAccepted, map[string]string{
		"message":        "posting queued",
		"transaction_id": txn.ID.String(),
	})
}

func (h *Handler) GetEntries(w http.ResponseWriter, r *http.Request) {
	authUser, err := auth.GetUserFromContext(r.Context())
	if err != nil {
		common.WriteErrorJSON(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	txnIDStr := chi.URLParam(r, "transaction_id")
	txnID, err := uuid.Parse(txnIDStr)
	if err != nil {
		common.WriteErrorJSON(w, http.StatusBadRequest, "Invalid transaction ID")
		return
	}

	txn, err := h.txnStore.GetTransactionByID(r.Context(), txnID)
	if err != nil {
		common.WriteErrorJSON(w, http.StatusNotFound, "Transaction not found")
		return
	}

	if !auth.CanAccessCompany(authUser, txn.CompanyID) {
		common.WriteErrorJSON(w, http.StatusForbidden, "user is not authorized to access entries for this transaction")
		return
	}

	entries, err := h.store.GetEntriesByTransaction(r.Context(), txnID)
	if err != nil {
		common.WriteErrorJSON(w, http.StatusInternalServerError, "Error retrieving entries: "+err.Error())
		return
	}

	common.WriteJSON(w, http.StatusOK, entries)
}
