package users

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

const otpTTL = 10 * time.Minute

type Handler struct {
	store      types.UserStore
	redisStore types.RedisStore
	config     config.Config
	auditStore types.AuditStore
	mailer     *mailer.Mailer
}

func NewHandler(store types.UserStore, redisStore types.RedisStore, cfg config.Config, auditStore types.AuditStore, m *mailer.Mailer) *Handler {
	return &Handler{store: store, redisStore: redisStore, config: cfg, auditStore: auditStore, mailer: m}
}

func (h *Handler) RegisterRoutes(router chi.Router) {
	router.Post("/register", h.handleRegister)
	router.Post("/request-otp", h.RequestLoginOTP)
	router.Post("/verify-otp", h.VerifyLoginOTP)
	router.Post("/refresh", h.HandleRefresh)
}

func (h *Handler) handleRegister(w http.ResponseWriter, r *http.Request) {

	if r.Body == nil {
		common.WriteErrorJSON(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	var payload types.CreateUserPayload
	if err := common.ReadJSON(r, &payload); err != nil {
		common.WriteErrorJSON(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if err := common.Validate.Struct(payload); err != nil {
		common.WriteErrorJSON(w, http.StatusBadRequest, "Missing or invalid fields: "+err.Error())
		return
	}

	_, err := h.store.GetUserByEmail(r.Context(), payload.Email)

	if err == nil {
		common.WriteErrorJSON(w, http.StatusBadRequest, "User with this email already exists")
		return
	}

	role := payload.Role
	if role == "" {
		role = types.RoleViewer
	}

	user, err := h.store.CreateUser(r.Context(), payload.Name, payload.Email, role, payload.AvatarURL)

	if err != nil {
		common.WriteErrorJSON(w, http.StatusInternalServerError, "Error creating user: "+err.Error())
		return
	}

	_ = h.auditStore.CreateAuditLog(r.Context(), &types.AuditLog{
		EntityType: "user",
		EntityID:   &user.ID,
		Action:     types.AuditActionUserCreated,
		CompanyID:  payload.CompanyID,
		ChangedBy: &user.ID,
	})

	common.WriteJSON(w, http.StatusOK, user)

}

func (h *Handler) RequestLoginOTP(w http.ResponseWriter, r *http.Request) {

	if r.Body == nil {
		common.WriteErrorJSON(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	var payload types.RequestLoginOTPPayload

	if err := common.ReadJSON(r, &payload); err != nil {
		common.WriteErrorJSON(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if err := common.Validate.Struct(payload); err != nil {
		common.WriteErrorJSON(w, http.StatusBadRequest, "Missing or invalid fields: "+err.Error())
		return
	}

	// Verify the user exists before issuing an OTP
	user, err := h.store.GetUserByEmail(r.Context(), payload.Email)
	if err != nil {
		// Return a generic message to avoid email enumeration
		common.WriteJSON(w, http.StatusOK, map[string]string{"message": "if that email is registered, a code has been sent"})
		return
	}

	loginOtp, err := auth.GenerateSecureOTP()
	if err != nil {
		common.WriteErrorJSON(w, http.StatusInternalServerError, "Error generating OTP")
		return
	}

	// Store OTP token in PostgreSQL tokens table
	_, err = h.store.CreateToken(r.Context(), user.ID, loginOtp, "otp", time.Now().Add(otpTTL))
	if err != nil {
		log.Printf("failed to save OTP token for %s: %v", payload.Email, err)
		common.WriteErrorJSON(w, http.StatusInternalServerError, "Error saving OTP")
		return
	}

	// Email the OTP — never return it in the response
	if err := h.mailer.SendLoginOTP(payload.Email, loginOtp); err != nil {
		log.Printf("failed to send OTP email to %s: %v", payload.Email, err)
		common.WriteErrorJSON(w, http.StatusInternalServerError, "Error sending OTP email")
		return
	}

	common.WriteJSON(w, http.StatusOK, map[string]string{"message": "if that email is registered, a code has been sent"})
}

func (h *Handler) VerifyLoginOTP(w http.ResponseWriter, r *http.Request) {

	if r.Body == nil {
		common.WriteErrorJSON(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	var payload types.LoginWithOTPPayload

	if err := common.ReadJSON(r, &payload); err != nil {
		common.WriteErrorJSON(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if err := common.Validate.Struct(payload); err != nil {
		common.WriteErrorJSON(w, http.StatusBadRequest, "Missing or invalid fields: "+err.Error())
		return
	}

	user, err := h.store.GetUserByEmail(r.Context(), payload.Email)
	if err != nil {
		common.WriteErrorJSON(w, http.StatusUnauthorized, "invalid or expired OTP")
		return
	}

	// Verify OTP token in database (must be un-used and un-expired)
	tokenRecord, err := h.store.GetValidToken(r.Context(), user.ID, payload.OTP, "otp")
	if err != nil {
		common.WriteErrorJSON(w, http.StatusUnauthorized, "invalid or expired OTP")
		return
	}

	// Mark OTP token used to prevent reuse
	if err := h.store.MarkTokenUsed(r.Context(), tokenRecord.ID); err != nil {
		log.Printf("failed to mark token as used: %v", err)
	}


	tokens, refreshTokenID, err := auth.CreateJWTs(
		[]byte(h.config.JWTSecret),
		[]byte(h.config.JWTRefreshSecret),
		user.ID,
		user.CompanyID,
		user.Role,
		h.config.JWTAccessExpiration,
		h.config.JWTRefreshExpiration,
	)
	if err != nil {
		common.WriteErrorJSON(w, http.StatusInternalServerError, "Error generating tokens")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    tokens.RefreshToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(h.config.JWTRefreshExpiration.Seconds()),
	})

	err = h.redisStore.SaveRefreshToken(r.Context(), user.ID.String(), refreshTokenID, h.config.JWTRefreshExpiration)
	if err != nil {
		common.WriteErrorJSON(w, http.StatusInternalServerError, "Error saving refresh token")
		return
	}

	common.WriteJSON(w, http.StatusOK, tokens.AccessToken)
}

func (h *Handler) HandleRefresh(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("x-refresh-token")

	if authHeader == "" {
		common.WriteErrorJSON(w, http.StatusUnauthorized, "missing x-refresh-token header")
		return
	}

	claims, err := auth.ValidateRefreshToken(authHeader, []byte(h.config.JWTRefreshSecret))

	if err != nil {
		common.WriteErrorJSON(w, http.StatusUnauthorized, "invalid refresh token")
		return
	}

	claimsMap := *claims

	userIDStr, ok := claimsMap["user_id"].(string)
	if !ok {
		common.WriteErrorJSON(w, http.StatusUnauthorized, "invalid refresh token claims")
		return
	}

	incomingTokenID, ok := claimsMap["token_id"].(string)
	if !ok {
		common.WriteErrorJSON(w, http.StatusUnauthorized, "invalid refresh token claims")
		return
	}

	activeTokenID, err := h.redisStore.GetRefreshToken(r.Context(), userIDStr)

	if err != nil {
		common.WriteErrorJSON(w, http.StatusUnauthorized, "invalid refresh token")
		return
	}

	if incomingTokenID != activeTokenID {
		common.WriteErrorJSON(w, http.StatusUnauthorized, "invalid refresh token")
		return
	}

	userID, err := uuid.Parse(userIDStr)

	if err != nil {
		common.WriteErrorJSON(w, http.StatusInternalServerError, "invalid refresh token claims")
		return
	}

	user, err := h.store.GetUserByID(r.Context(), userID)

	if err != nil {
		common.WriteErrorJSON(w, http.StatusInternalServerError, "Error getting user")
		return
	}

	tokens, newTokenID, err := auth.CreateJWTs(
		[]byte(h.config.JWTSecret),
		[]byte(h.config.JWTRefreshSecret),
		user.ID,
		user.CompanyID,
		user.Role,
		h.config.JWTAccessExpiration,
		h.config.JWTRefreshExpiration,
	)
	if err != nil {
		common.WriteErrorJSON(w, http.StatusInternalServerError, "Error generating tokens")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    tokens.RefreshToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(h.config.JWTRefreshExpiration.Seconds()),
	})

	err = h.redisStore.SaveRefreshToken(r.Context(), user.ID.String(), newTokenID, h.config.JWTRefreshExpiration)
	if err != nil {
		common.WriteErrorJSON(w, http.StatusInternalServerError, "Error saving refresh token")
		return
	}

	common.WriteJSON(w, http.StatusOK, tokens.AccessToken)
}
