package auth

import (
	"fmt"
	"time"

	"github.com/Val-senseisama/payments/types"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func CreateJWTs(accessSecret []byte, refreshSecret []byte, userID uuid.UUID, companyID uuid.UUID, role types.UserRole, expAcc time.Duration, expRef time.Duration) (*types.TokenPair, string, error) {
	refreshTokenID := uuid.New().String()
	accessClaims := jwt.MapClaims{
		"user_id":    userID.String(),
		"company_id": companyID.String(),
		"role":       string(role),
		"exp":        time.Now().Add(expAcc).Unix(),
	}
	refreshTokenClaims := jwt.MapClaims{
		"user_id":  userID.String(),
		"token_id": refreshTokenID,
		"exp":      time.Now().Add(expRef).Unix(),
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshTokenClaims)

	accessTokenString, err := accessToken.SignedString(accessSecret)
	if err != nil {
		return nil, "", fmt.Errorf("failed to sign token: %w", err)
	}
	refreshTokenString, err := refreshToken.SignedString(refreshSecret)
	if err != nil {
		return nil, "", fmt.Errorf("failed to sign token: %w", err)
	}

	return &types.TokenPair{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
	}, refreshTokenID, nil
}

func ValidateAccessToken(tokenString string, secret []byte) (*jwt.MapClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwt.MapClaims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

func ValidateRefreshToken(tokenString string, secret []byte) (*jwt.MapClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwt.MapClaims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}
