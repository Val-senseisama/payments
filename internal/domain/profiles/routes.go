package profiles

import (
	"net/http"

	"github.com/Val-senseisama/payments/cmd/config"
	"github.com/Val-senseisama/payments/internal/common"
	"github.com/Val-senseisama/payments/internal/common/auth"
	"github.com/Val-senseisama/payments/types"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	store      types.ProfileStore
	config     config.Config
	auditStore types.AuditStore
}

func NewHandler(store types.ProfileStore, cfg config.Config, auditStore types.AuditStore) *Handler {
	return &Handler{
		store:      store,
		config:     cfg,
		auditStore: auditStore,
	}
}

func (h *Handler) RegisterRoutes(router chi.Router) {
	router.Post("/", h.CreateProfile)
	router.Get("/", h.GetProfiles)
	router.Get("/{id}", h.GetProfileByID)
	router.Put("/{id}", h.UpdateProfile)
	router.Delete("/{id}", h.DeleteProfile)
}

func (h *Handler) CreateProfile(w http.ResponseWriter, r *http.Request) {
	authUser, err := auth.GetUserFromContext(r.Context())
	if err != nil {
		common.WriteErrorJSON(w, http.StatusUnauthorized, "user not found on context")
		return
	}

	var payload types.CreateProfilePayload
	if err := common.ReadJSON(r, &payload); err != nil {
		common.WriteErrorJSON(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if err := common.Validate.Struct(payload); err != nil {
		common.WriteErrorJSON(w, http.StatusBadRequest, "Missing or invalid fields: "+err.Error())
		return
	}

	if !auth.RequestAccess(authUser, payload.CompanyID, types.RoleAdmin, types.RoleFinance) {
		common.WriteErrorJSON(w, http.StatusForbidden, "user is not authorized to create profiles for this company")
		return
	}

	profile, err := h.store.CreateProfile(r.Context(), payload.CompanyID, payload.Type, payload.Name, payload.Email, payload.Phone, payload.AvatarURL, nil)
	if err != nil {
		common.WriteErrorJSON(w, http.StatusInternalServerError, "Error creating profile: "+err.Error())
		return
	}

	_ = h.auditStore.CreateAuditLog(r.Context(), &types.AuditLog{
		EntityType: "profile",
		EntityID:   &profile.ID,
		CompanyID:  &profile.CompanyID,
		Action:     "profile.created",
		ChangedBy:  &authUser.UserID,
	})

	common.WriteJSON(w, http.StatusCreated, profile)
}

func (h *Handler) GetProfiles(w http.ResponseWriter, r *http.Request) {
	authUser, err := auth.GetUserFromContext(r.Context())
	if err != nil {
		common.WriteErrorJSON(w, http.StatusUnauthorized, "user not found on context")
		return
	}

	if authUser.CompanyID == nil && !auth.IsSuperAdmin(authUser) {
		common.WriteErrorJSON(w, http.StatusForbidden, "user does not belong to a company")
		return
	}

	var pType *types.ProfileType
	if typeParam := r.URL.Query().Get("type"); typeParam != "" {
		t := types.ProfileType(typeParam)
		pType = &t
	}

	companyID := *authUser.CompanyID
	profiles, err := h.store.GetProfilesByCompany(r.Context(), companyID, pType)
	if err != nil {
		common.WriteErrorJSON(w, http.StatusInternalServerError, "Error retrieving profiles: "+err.Error())
		return
	}

	common.WriteJSON(w, http.StatusOK, profiles)
}

func (h *Handler) GetProfileByID(w http.ResponseWriter, r *http.Request) {
	authUser, err := auth.GetUserFromContext(r.Context())
	if err != nil {
		common.WriteErrorJSON(w, http.StatusUnauthorized, "user not found on context")
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		common.WriteErrorJSON(w, http.StatusBadRequest, "Invalid profile ID")
		return
	}

	profile, err := h.store.GetProfileByID(r.Context(), id)
	if err != nil {
		common.WriteErrorJSON(w, http.StatusNotFound, "Error getting profile: "+err.Error())
		return
	}

	if !auth.CanAccessCompany(authUser, profile.CompanyID) {
		common.WriteErrorJSON(w, http.StatusForbidden, "user is not authorized to access this profile")
		return
	}

	common.WriteJSON(w, http.StatusOK, profile)
}

func (h *Handler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	authUser, err := auth.GetUserFromContext(r.Context())
	if err != nil {
		common.WriteErrorJSON(w, http.StatusUnauthorized, "user not found on context")
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		common.WriteErrorJSON(w, http.StatusBadRequest, "Invalid profile ID")
		return
	}

	existing, err := h.store.GetProfileByID(r.Context(), id)
	if err != nil {
		common.WriteErrorJSON(w, http.StatusNotFound, "Profile not found")
		return
	}

	if !auth.RequestAccess(authUser, existing.CompanyID, types.RoleAdmin, types.RoleFinance) {
		common.WriteErrorJSON(w, http.StatusForbidden, "user is not authorized to update profiles for this company")
		return
	}

	var payload types.UpdateProfilePayload
	if err := common.ReadJSON(r, &payload); err != nil {
		common.WriteErrorJSON(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	updated, err := h.store.UpdateProfile(r.Context(), id, payload.Name, payload.Email, payload.Phone, payload.AvatarURL)
	if err != nil {
		common.WriteErrorJSON(w, http.StatusInternalServerError, "Error updating profile: "+err.Error())
		return
	}

	_ = h.auditStore.CreateAuditLog(r.Context(), &types.AuditLog{
		EntityType: "profile",
		EntityID:   &updated.ID,
		CompanyID:  &updated.CompanyID,
		Action:     "profile.updated",
		ChangedBy:  &authUser.UserID,
	})

	common.WriteJSON(w, http.StatusOK, updated)
}

func (h *Handler) DeleteProfile(w http.ResponseWriter, r *http.Request) {
	authUser, err := auth.GetUserFromContext(r.Context())
	if err != nil {
		common.WriteErrorJSON(w, http.StatusUnauthorized, "user not found on context")
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		common.WriteErrorJSON(w, http.StatusBadRequest, "Invalid profile ID")
		return
	}

	existing, err := h.store.GetProfileByID(r.Context(), id)
	if err != nil {
		common.WriteErrorJSON(w, http.StatusNotFound, "Profile not found")
		return
	}

	if !auth.RequestAccess(authUser, existing.CompanyID, types.RoleAdmin) {
		common.WriteErrorJSON(w, http.StatusForbidden, "user is not authorized to delete profiles for this company")
		return
	}

	deleted, err := h.store.DeleteProfile(r.Context(), id)
	if err != nil {
		common.WriteErrorJSON(w, http.StatusInternalServerError, "Error deleting profile: "+err.Error())
		return
	}

	_ = h.auditStore.CreateAuditLog(r.Context(), &types.AuditLog{
		EntityType: "profile",
		EntityID:   &deleted.ID,
		CompanyID:  &deleted.CompanyID,
		Action:     "profile.deleted",
		ChangedBy:  &authUser.UserID,
	})

	common.WriteJSON(w, http.StatusOK, deleted)
}
