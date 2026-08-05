package auth_test

import (
	"testing"

	"github.com/Val-senseisama/payments/internal/common/auth"
	"github.com/Val-senseisama/payments/types"
	"github.com/google/uuid"
)

func TestRequestAccess(t *testing.T) {
	companyID := uuid.New()
	otherCompanyID := uuid.New()

	tests := []struct {
		name         string
		user         *types.AuthenthicatedUser
		targetCompID uuid.UUID
		requiredRole types.UserRole
		wantAllowed  bool
	}{
		{
			name: "SuperAdmin allowed with RoleSuperAdmin requirement",
			user: &types.AuthenthicatedUser{
				UserID:    uuid.New(),
				CompanyID: nil,
				Role:      types.RoleSuperAdmin,
			},
			targetCompID: companyID,
			requiredRole: types.RoleSuperAdmin,
			wantAllowed:  true,
		},
		{
			name: "Admin allowed for matching company",
			user: &types.AuthenthicatedUser{
				UserID:    uuid.New(),
				CompanyID: &companyID,
				Role:      types.RoleAdmin,
			},
			targetCompID: companyID,
			requiredRole: types.RoleAdmin,
			wantAllowed:  true,
		},
		{
			name: "Admin denied for different company",
			user: &types.AuthenthicatedUser{
				UserID:    uuid.New(),
				CompanyID: &companyID,
				Role:      types.RoleAdmin,
			},
			targetCompID: otherCompanyID,
			requiredRole: types.RoleAdmin,
			wantAllowed:  false,
		},
		{
			name: "Viewer denied for Admin required role",
			user: &types.AuthenthicatedUser{
				UserID:    uuid.New(),
				CompanyID: &companyID,
				Role:      types.RoleViewer,
			},
			targetCompID: companyID,
			requiredRole: types.RoleAdmin,
			wantAllowed:  false,
		},
		{
			name: "Finance allowed for Finance required role",
			user: &types.AuthenthicatedUser{
				UserID:    uuid.New(),
				CompanyID: &companyID,
				Role:      types.RoleFinance,
			},
			targetCompID: companyID,
			requiredRole: types.RoleFinance,
			wantAllowed:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := auth.RequestAccess(tt.user, tt.targetCompID, tt.requiredRole)
			if got != tt.wantAllowed {
				t.Errorf("RequestAccess() = %v, want %v", got, tt.wantAllowed)
			}
		})
	}
}

func TestCanAccessCompany(t *testing.T) {
	companyID := uuid.New()
	otherCompanyID := uuid.New()

	superAdmin := &types.AuthenthicatedUser{
		UserID: uuid.New(),
		Role:   types.RoleSuperAdmin,
	}

	member := &types.AuthenthicatedUser{
		UserID:    uuid.New(),
		CompanyID: &companyID,
		Role:      types.RoleViewer,
	}

	if !auth.CanAccessCompany(superAdmin, companyID) {
		t.Errorf("expected super admin to access company")
	}

	if !auth.CanAccessCompany(member, companyID) {
		t.Errorf("expected member to access their company")
	}

	if auth.CanAccessCompany(member, otherCompanyID) {
		t.Errorf("expected member to be denied access to other company")
	}
}
