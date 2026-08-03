package company

import (
	"log"
	"net/http"
	"time"

	"github.com/Val-senseisama/payments/cmd/config"
	"github.com/Val-senseisama/payments/internal/common"
	"github.com/Val-senseisama/payments/internal/common/auth"
	"github.com/Val-senseisama/payments/internal/mailer"
	"github.com/Val-senseisama/payments/types"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	store      types.CompanyStore
	userStore  types.UserStore
	redisStore types.RedisStore
	config     config.Config
	auditStore types.AuditStore
	mailer     *mailer.Mailer
}

func NewHandler(store types.CompanyStore, userStore types.UserStore, redisStore types.RedisStore, config config.Config, auditStore types.AuditStore, m *mailer.Mailer) *Handler {
	return &Handler{store: store, userStore: userStore, redisStore: redisStore, config: config, auditStore: auditStore, mailer: m}
}

func (h *Handler) RegisterRoutes(router chi.Router) {
	router.Post("/", h.CreateCompany)
	router.Get("/", h.GetCompanies)
	router.Get("/{id}", h.GetCompanyByID)
	router.Put("/{id}", h.UpdateCompany)
	router.Delete("/{id}", h.DeleteCompany)
	router.Post("/invite", h.InviteUserToCompany)
	router.Post("/accept-invite", h.AcceptCompanyInvite)
}

func (h *Handler) CreateCompany(w http.ResponseWriter, r *http.Request) {
	//check for user id on context

	authUser, err := auth.GetUserFromContext(r.Context())
	if err != nil {
		common.WriteErrorJSON(w, http.StatusUnauthorized, "user not found on context")
		return
	}

	var payload types.CreateCompanyPayload

	if err := common.ReadJSON(r, &payload); err != nil {
		common.WriteErrorJSON(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if err := common.Validate.Struct(payload); err != nil {
		common.WriteErrorJSON(w, http.StatusBadRequest, "Missing or invalid fields: "+err.Error())
		return
	}

	company, err := h.store.CreateCompany(r.Context(), payload.Name, authUser.UserID)
	if err != nil {
		common.WriteErrorJSON(w, http.StatusInternalServerError, "Error creating company: "+err.Error())
		return
	}

	// audit trail

	err = h.auditStore.CreateAuditLog(r.Context(), &types.AuditLog{
		EntityType: "company",
		EntityID:   &company.ID,
		CompanyID:  &company.ID,
		Action:     types.AuditActionCompanyCreated,
		ChangedBy:  &authUser.UserID,
	})

	if err != nil {
		log.Printf("failed to write audit log: %v", err)
	}

	common.WriteJSON(w, http.StatusCreated, company)

}
func (h *Handler) GetCompanyByID(w http.ResponseWriter, r *http.Request) {

	authUser, err := auth.GetUserFromContext(r.Context())

	if err != nil {
		common.WriteErrorJSON(w, http.StatusUnauthorized, "user not found on context")
		return
	}

	companyIDStr := chi.URLParam(r, "id")
	companyID, err := uuid.Parse(companyIDStr)
	if err != nil {
		common.WriteErrorJSON(w, http.StatusBadRequest, "Invalid company ID")
		return
	}

	if !auth.CanAccessCompany(authUser, companyID) {
		common.WriteErrorJSON(w, http.StatusForbidden, "user is not authorized to access this company")
		return
	}

	company, err := h.store.GetCompanyByID(r.Context(), companyID)
	if err != nil {
		common.WriteErrorJSON(w, http.StatusNotFound, "Error getting company: "+err.Error())
		return
	}

	common.WriteJSON(w, http.StatusOK, company)
}
func (h *Handler) GetCompanies(w http.ResponseWriter, r *http.Request) {
	authUser, err := auth.GetUserFromContext(r.Context())

	if err != nil {
		common.WriteErrorJSON(w, http.StatusUnauthorized, "user not found on context")
		return
	}

	if !auth.IsSuperAdmin(authUser) {

		common.WriteErrorJSON(w, http.StatusForbidden, "user is not authorized to access this resource")
		return
	}

	companies, err := h.store.GetCompanies(r.Context())
	if err != nil {
		common.WriteErrorJSON(w, http.StatusInternalServerError, "Error getting companies: "+err.Error())
		return
	}

	common.WriteJSON(w, http.StatusOK, companies)
}

func (h *Handler) UpdateCompany(w http.ResponseWriter, r *http.Request) {
	authUser, err := auth.GetUserFromContext(r.Context())
	if err != nil {
		common.WriteErrorJSON(w, http.StatusUnauthorized, "user not found on context")
		return
	}

	companyIDStr := chi.URLParam(r, "id")
	companyID, err := uuid.Parse(companyIDStr)
	if err != nil {
		common.WriteErrorJSON(w, http.StatusBadRequest, "Invalid company ID")
		return
	}

	if !auth.RequestAccess(authUser, companyID, types.RoleAdmin) {
		common.WriteErrorJSON(w, http.StatusForbidden, "user is not authorized to update this company")
		return
	}

	var payload types.CreateCompanyPayload
	if err := common.ReadJSON(r, &payload); err != nil {
		common.WriteErrorJSON(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if err := common.Validate.Struct(payload); err != nil {
		common.WriteErrorJSON(w, http.StatusBadRequest, "Missing or invalid fields: "+err.Error())
		return
	}

	company, err := h.store.UpdateCompany(r.Context(), companyID, payload.Name)
	if err != nil {
		common.WriteErrorJSON(w, http.StatusInternalServerError, "Error updating company: "+err.Error())
		return
	}

	err = h.auditStore.CreateAuditLog(r.Context(), &types.AuditLog{
		EntityType: "company",
		EntityID:   &company.ID,
		CompanyID:  &company.ID,
		Action:     types.AuditActionCompanyUpdated,
		ChangedBy:  &authUser.UserID,
	})
	if err != nil {
		log.Printf("failed to write audit log: %v", err)
	}

	common.WriteJSON(w, http.StatusOK, company)
}

func (h *Handler) DeleteCompany(w http.ResponseWriter, r *http.Request) {
	authUser, err := auth.GetUserFromContext(r.Context())
	if err != nil {
		common.WriteErrorJSON(w, http.StatusUnauthorized, "user not found on context")
		return
	}

	companyIDStr := chi.URLParam(r, "id")
	companyID, err := uuid.Parse(companyIDStr)
	if err != nil {
		common.WriteErrorJSON(w, http.StatusBadRequest, "Invalid company ID")
		return
	}

	if !auth.RequestAccess(authUser, companyID, types.RoleAdmin) {
		common.WriteErrorJSON(w, http.StatusForbidden, "user is not authorized to delete this company")
		return
	}

	company, err := h.store.DeleteCompany(r.Context(), companyID)
	if err != nil {
		common.WriteErrorJSON(w, http.StatusInternalServerError, "Error deleting company: "+err.Error())
		return
	}

	err = h.auditStore.CreateAuditLog(r.Context(), &types.AuditLog{
		EntityType: "company",
		EntityID:   &company.ID,
		CompanyID:  &company.ID,
		Action:     types.AuditActionCompanyDeleted,
		ChangedBy:  &authUser.UserID,
	})
	if err != nil {
		log.Printf("failed to write audit log: %v", err)
	}

	common.WriteJSON(w, http.StatusOK, company)
}

func (h *Handler) InviteUserToCompany(w http.ResponseWriter, r *http.Request) {
	authUser, err := auth.GetUserFromContext(r.Context())
	if err != nil {
		common.WriteErrorJSON(w, http.StatusUnauthorized, "user not found on context")
		return
	}

	var payload types.InviteUserToCompanyPayload
	if err := common.ReadJSON(r, &payload); err != nil {
		common.WriteErrorJSON(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if err := common.Validate.Struct(payload); err != nil {
		common.WriteErrorJSON(w, http.StatusBadRequest, "Missing or invalid fields: "+err.Error())
		return
	}

	if !auth.RequestAccess(authUser, payload.CompanyID, types.RoleAdmin) {
		common.WriteErrorJSON(w, http.StatusForbidden, "user is not authorized to invite users to this company")
		return
	}

	company, err := h.store.GetCompanyByID(r.Context(), payload.CompanyID)
	if err != nil {
		common.WriteErrorJSON(w, http.StatusInternalServerError, "failed to fetch company: "+err.Error())
		return
	}

	inviteToken := uuid.New().String()
	role := payload.Role
	if role == "" {
		role = types.RoleViewer
	}
	expiresAt := time.Now().Add(48 * time.Hour)

	_, err = h.store.CreateInvitationToken(r.Context(), payload.CompanyID, payload.Email, role, inviteToken, expiresAt)
	if err != nil {
		common.WriteErrorJSON(w, http.StatusInternalServerError, "failed to save invitation token: "+err.Error())
		return
	}

	if err := h.mailer.SendCompanyInvite(payload.Email, company.Name, "an administrator"); err != nil {
		log.Printf("failed to send invite email to %s: %v", payload.Email, err)
		common.WriteErrorJSON(w, http.StatusInternalServerError, "failed to send invite email")
		return
	}

	common.WriteJSON(w, http.StatusOK, map[string]string{
		"message": "invite sent to " + payload.Email,
		"token":   inviteToken,
	})
}

func (h *Handler) AcceptCompanyInvite(w http.ResponseWriter, r *http.Request) {
	var payload types.AcceptCompanyInvitePayload
	if err := common.ReadJSON(r, &payload); err != nil {
		common.WriteErrorJSON(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if err := common.Validate.Struct(payload); err != nil {
		common.WriteErrorJSON(w, http.StatusBadRequest, "Missing or invalid fields: "+err.Error())
		return
	}

	inviteToken, err := h.store.GetInvitationToken(r.Context(), payload.Token)
	if err != nil {
		common.WriteErrorJSON(w, http.StatusBadRequest, "Invalid invitation token")
		return
	}

	if inviteToken.AcceptedAt != nil {
		common.WriteErrorJSON(w, http.StatusBadRequest, "Invitation token has already been used")
		return
	}

	if time.Now().After(inviteToken.ExpiresAt) {
		common.WriteErrorJSON(w, http.StatusBadRequest, "Invitation token has expired")
		return
	}

	user, err := h.userStore.GetUserByEmail(r.Context(), inviteToken.Email)
	if err != nil {
		common.WriteErrorJSON(w, http.StatusNotFound, "User not found for this invitation. Please register first.")
		return
	}

	if err := h.userStore.AddUserToCompany(r.Context(), user.ID, inviteToken.CompanyID, inviteToken.Role); err != nil {
		common.WriteErrorJSON(w, http.StatusInternalServerError, "Failed to join company: "+err.Error())
		return
	}

	if err := h.store.MarkInvitationTokenAccepted(r.Context(), inviteToken.ID); err != nil {
		log.Printf("failed to mark invitation token accepted: %v", err)
	}

	_ = h.auditStore.CreateAuditLog(r.Context(), &types.AuditLog{
		EntityType: "company",
		EntityID:   &inviteToken.CompanyID,
		CompanyID:  &inviteToken.CompanyID,
		Action:     types.AuditActionUserUpdated,
		ChangedBy:  &user.ID,
	})

	common.WriteJSON(w, http.StatusOK, map[string]string{"message": "successfully joined company"})
}