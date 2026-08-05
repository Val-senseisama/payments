package transactions

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Val-senseisama/payments/cmd/config"
	"github.com/Val-senseisama/payments/internal/common"
	"github.com/Val-senseisama/payments/internal/common/auth"
	"github.com/Val-senseisama/payments/internal/domain/audit"
	"github.com/Val-senseisama/payments/types"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	store       types.TransactionStore
	redisStore  types.RedisStore
	config      config.Config
	auditWorker *audit.Worker
}

func NewHandler(store types.TransactionStore, redisStore types.RedisStore, cfg config.Config, auditWorker *audit.Worker) *Handler {
	return &Handler{
		store:       store,
		redisStore:  redisStore,
		config:      cfg,
		auditWorker: auditWorker,
	}
}

func (h *Handler) RegisterRoutes(router chi.Router) {
	router.Post("/", h.CreateTransaction)
	router.Get("/", h.GetTransactions)
	router.Get("/stream", h.StreamTransactionUpdates)
	router.Get("/{id}", h.GetTransactionByID)
}

func (h *Handler) CreateTransaction(w http.ResponseWriter, r *http.Request) {
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
		common.WriteErrorJSON(w, http.StatusForbidden, "user is not authorized to create transactions")
		return
	}

	var payload types.CreateTransactionPayload
	if err := common.ReadJSON(r, &payload); err != nil {
		common.WriteErrorJSON(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if err := common.Validate.Struct(payload); err != nil {
		common.WriteErrorJSON(w, http.StatusBadRequest, "Missing or invalid fields: "+err.Error())
		return
	}

	// Redis Idempotency Key check
	idempotencyKey := fmt.Sprintf("%s:%s", authUser.CompanyID.String(), payload.Reference)
	if cachedTxn, err := h.redisStore.GetIdempotencyKey(r.Context(), idempotencyKey); err == nil {
		var txn types.Transaction
		if jsonErr := json.Unmarshal(cachedTxn, &txn); jsonErr == nil {
			common.WriteJSON(w, http.StatusOK, txn)
			return
		}
	}

	txn, err := h.store.CreateTransaction(
		r.Context(),
		*authUser.CompanyID,
		authUser.UserID,
		payload.Reference,
		payload.Type,
		payload.Amount,
	)
	if err != nil {
		common.WriteErrorJSON(w, http.StatusInternalServerError, "Error creating transaction: "+err.Error())
		return
	}

	// Cache idempotency response for 24 hours
	if txnBytes, err := json.Marshal(txn); err == nil {
		_ = h.redisStore.SetIdempotencyKey(r.Context(), idempotencyKey, txnBytes, 24*time.Hour)
	}

	// Async audit log
	h.auditWorker.Send(&types.AuditLog{
		CompanyID:  authUser.CompanyID,
		EntityType: "transaction",
		EntityID:   &txn.ID,
		Action:     "transaction.created",
		ChangedBy:  &authUser.UserID,
	})

	common.WriteJSON(w, http.StatusCreated, txn)
}

func (h *Handler) GetTransactions(w http.ResponseWriter, r *http.Request) {
	authUser, err := auth.GetUserFromContext(r.Context())
	if err != nil {
		common.WriteErrorJSON(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	if authUser.CompanyID == nil {
		common.WriteErrorJSON(w, http.StatusForbidden, "user does not belong to a company")
		return
	}

	txns, err := h.store.GetTransactionsByCompany(r.Context(), *authUser.CompanyID)
	if err != nil {
		common.WriteErrorJSON(w, http.StatusInternalServerError, "Error retrieving transactions: "+err.Error())
		return
	}

	common.WriteJSON(w, http.StatusOK, txns)
}

func (h *Handler) GetTransactionByID(w http.ResponseWriter, r *http.Request) {
	authUser, err := auth.GetUserFromContext(r.Context())
	if err != nil {
		common.WriteErrorJSON(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		common.WriteErrorJSON(w, http.StatusBadRequest, "Invalid transaction ID")
		return
	}

	txn, err := h.store.GetTransactionByID(r.Context(), id)
	if err != nil {
		common.WriteErrorJSON(w, http.StatusNotFound, "Transaction not found")
		return
	}

	if !auth.CanAccessCompany(authUser, txn.CompanyID) {
		common.WriteErrorJSON(w, http.StatusForbidden, "user is not authorized to access this transaction")
		return
	}

	common.WriteJSON(w, http.StatusOK, txn)
}

func (h *Handler) StreamTransactionUpdates(w http.ResponseWriter, r *http.Request) {
	authUser, err := auth.GetUserFromContext(r.Context())
	if err != nil {
		common.WriteErrorJSON(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	if authUser.CompanyID == nil {
		common.WriteErrorJSON(w, http.StatusForbidden, "user does not belong to a company")
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		common.WriteErrorJSON(w, http.StatusInternalServerError, "Streaming unsupported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	updatesChan, cleanup := h.redisStore.SubscribeTransactionUpdates(r.Context(), authUser.CompanyID.String())
	defer cleanup()

	notify := r.Context().Done()
	for {
		select {
		case <-notify:
			return
		case msg, open := <-updatesChan:
			if !open {
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", msg)
			flusher.Flush()
		}
	}
}
