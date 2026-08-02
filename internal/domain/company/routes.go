package company

import (
	"log"
	"net/http"

	"github.com/Val-senseisama/payments/cmd/config"
	"github.com/Val-senseisama/payments/internal/common"
	"github.com/Val-senseisama/payments/internal/common/auth"
	"github.com/Val-senseisama/payments/types"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handle struct {
	store      types.CompanyStore
	redisStore types.RedisStore
	config     config.Config
	auditStore types.AuditStore
}

func NewHandle(store types.CompanyStore, redisStore types.RedisStore, config config.Config, auditStore types.AuditStore) *Handle {
	return &Handle{store: store, redisStore: redisStore, config: config, auditStore: auditStore}
}

func (h *Handle) RegisterRoutes(router chi.Router) {
	router.Post("/", h.CreateCompany)
	router.Get("/", h.GetCompanies)
	router.Get("/{id}", h.GetCompanyByID)
	router.Put("/{id}", h.UpdateCompany)
	router.Delete("/{id}", h.DeleteCompany)
}

func (h *Handle) CreateCompany(w http.ResponseWriter, r *http.Request) {
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
func (h *Handle) GetCompanyByID(w http.ResponseWriter, r *http.Request) {

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
func (h *Handle) GetCompanies(w http.ResponseWriter, r *http.Request)  {
	authUser, err := auth.GetUserFromContext(r.Context())

	if err != nil {
		common.WriteErrorJSON(w, http.StatusUnauthorized, "user not found on context")
		return
	}

	if !auth.IsSuperAdmin(authUser){
		
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

func (h *Handle) UpdateCompany(w http.ResponseWriter, r *http.Request) {
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

func (h *Handle) DeleteCompany(w http.ResponseWriter, r *http.Request) {
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
