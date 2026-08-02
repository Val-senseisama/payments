package auth

import (
	"context"
	"net/http"

	"github.com/Val-senseisama/payments/internal/common"
	"github.com/Val-senseisama/payments/types"
	"github.com/google/uuid"
)

const (
	UserKey    types.ContextKey = "userID"
	CompanyKey types.ContextKey = "companyID"
	RoleKey    types.ContextKey = "userRole"
)

func AuthMiddleware(jwtsecret []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("x-access-token")

			if authHeader == "" {
				common.WriteErrorJSON(w, http.StatusUnauthorized, "missing x-access-token header")
				return
			}

			claims, err := ValidateAccessToken(authHeader, jwtsecret)
			if err != nil {
				common.WriteErrorJSON(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}

			claimsMap := *claims

			// 1. Type Assertions (returns string, bool NOT string, error)
			userIDStr, ok := claimsMap["user_id"].(string)
			if !ok {
				common.WriteErrorJSON(w, http.StatusUnauthorized, "invalid user_id claim")
				return
			}

			companyIDStr, ok := claimsMap["company_id"].(string)
			if !ok {
				common.WriteErrorJSON(w, http.StatusUnauthorized, "invalid company_id claim")
				return
			}

			roleStr, ok := claimsMap["role"].(string)
			if !ok {
				common.WriteErrorJSON(w, http.StatusUnauthorized, "invalid role claim")
				return
			}

			// 2. Parse UUID strings into uuid.UUID types
			userID, err := uuid.Parse(userIDStr)
			if err != nil {
				common.WriteErrorJSON(w, http.StatusUnauthorized, "malformed user_id UUID")
				return
			}

			companyID, err := uuid.Parse(companyIDStr)
			if err != nil {
				common.WriteErrorJSON(w, http.StatusUnauthorized, "malformed company_id UUID")
				return
			}

			// 3. Build context chain correctly
			ctx := r.Context()
			ctx = context.WithValue(ctx, UserKey, userID)
			ctx = context.WithValue(ctx, CompanyKey, companyID)
			ctx = context.WithValue(ctx, RoleKey, types.UserRole(roleStr))

			// 4. Pass request with updated context to next handler
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
