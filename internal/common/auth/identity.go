package auth

import (
	"context"
	"fmt"

	"github.com/Val-senseisama/payments/types"
	"github.com/google/uuid"
)

func GetUserFromContext(ctx context.Context) (*types.AuthenthicatedUser, error) {
	userID, ok := ctx.Value(UserKey).(uuid.UUID)
	if !ok {
		return nil, fmt.Errorf("unauthorized")
	}

	var companyID *uuid.UUID
	if companyIDVal, ok := ctx.Value(CompanyKey).(uuid.UUID); ok {
		companyID = &companyIDVal
	}

	role, ok := ctx.Value(RoleKey).(types.UserRole)
	if !ok {
		return nil, fmt.Errorf("unauthorized")
	}

	return &types.AuthenthicatedUser{
		UserID:    userID,
		CompanyID: companyID,
		Role:      role,
	}, nil

}

func CanAccessCompany(user *types.AuthenthicatedUser, targetCompanyID uuid.UUID) bool {

	if user == nil {
		return false
	}

	if targetCompanyID == uuid.Nil {
		return false
	}

	// check if super admin

	if user.Role == types.RoleSuperAdmin {
		return true
	}

	// check if company member

	return user.CompanyID != nil && *user.CompanyID == targetCompanyID
}

func RequestAccess(user *types.AuthenthicatedUser, targetCompanyID uuid.UUID, roles ...types.UserRole) bool {

	if !CanAccessCompany(user, targetCompanyID) {
		return false
	}

	for _, allowedRole := range roles {
		if user.Role == allowedRole {
			return true
		}
	}

	return false

}

func IsSuperAdmin(user *types.AuthenthicatedUser) bool {
	if user == nil {
		return false
	}
	return user.Role == types.RoleSuperAdmin
}
