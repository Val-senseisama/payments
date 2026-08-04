package account

import (
	"net/http"

	"github.com/Val-senseisama/payments/internal/common"
	"github.com/Val-senseisama/payments/internal/common/auth"
	"github.com/Val-senseisama/payments/types"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	store      types.AccountStore
	auditStore types.AuditStore
}

func NewHandler(store types.AccountStore, auditStore types.AuditStore) *Handler {
	return &Handler{store: store, auditStore: auditStore}
}

func (h *Handler) RegisterRoutes(router chi.Router) {
	router.Post("/", h.CreateAccount)
	router.Get("/", h.GetAccounts)
	router.Get("/{id}", h.GetAccountByID)
}

func (h *Handler) CreateAccount(w http.ResponseWriter, r *http.Request) {

	authUser, err := auth.GetUserFromContext(r.Context())

	if err != nil {
		common.WriteErrorJSON(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	if !auth.RequestAccess(authUser, *authUser.CompanyID, types.RoleAdmin) {
		common.WriteErrorJSON(w, http.StatusUnauthorized, "You don't have permission to create an account")
		return
	}

	if r.Body == nil {
		common.WriteErrorJSON(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	var payload types.CreateAccountPayload
	if err := common.ReadJSON(r, &payload); err != nil {
		common.WriteErrorJSON(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if err := common.Validate.Struct(payload); err != nil {
		common.WriteErrorJSON(w, http.StatusBadRequest, "Missing or invalid fields: "+err.Error())
		return
	}

	account, err := h.store.CreateAccount(r.Context(), *authUser.CompanyID, payload.ProfileID, payload.Type, payload.Name)
	if err != nil {
		common.WriteErrorJSON(w, http.StatusInternalServerError, "Error creating account")
		return
	}

	_ = h.auditStore.CreateAuditLog(r.Context(), &types.AuditLog{
		CompanyID:  authUser.CompanyID,
		EntityType: "account",
		EntityID:   &account.ID,
		Action:     "account.created",
		ChangedBy:  &authUser.UserID,
	})

	common.WriteJSON(w, http.StatusCreated, account)

}

func (h *Handler) GetAccounts(w http.ResponseWriter, r *http.Request) {
	authUser, err := auth.GetUserFromContext(r.Context())
	if err != nil {
		common.WriteErrorJSON(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	if authUser.CompanyID == nil {
		common.WriteErrorJSON(w, http.StatusForbidden, "user does not belong to a company")
		return
	}

	accounts, err := h.store.GetAccountsByCompany(r.Context(), *authUser.CompanyID)
	if err != nil {
		common.WriteErrorJSON(w, http.StatusInternalServerError, "Error retrieving accounts")
		return
	}

	common.WriteJSON(w, http.StatusOK, accounts)
}

func (h *Handler) GetAccountByID(w http.ResponseWriter, r *http.Request) {
	authUser, err := auth.GetUserFromContext(r.Context())
	if err != nil {
		common.WriteErrorJSON(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		common.WriteErrorJSON(w, http.StatusBadRequest, "Invalid account ID")
		return
	}

	account, err := h.store.GetAccountByID(r.Context(), id)
	if err != nil {
		common.WriteErrorJSON(w, http.StatusNotFound, "Account not found")
		return
	}

	if !auth.CanAccessCompany(authUser, account.CompanyID) {
		common.WriteErrorJSON(w, http.StatusForbidden, "user is not authorized to access this account")
		return
	}

	common.WriteJSON(w, http.StatusOK, account)
}
