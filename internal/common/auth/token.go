package auth

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

func GenerateSecureOTP() (string, error) {
	maxVal := big.NewInt(10000)
	n, err := rand.Int(rand.Reader, maxVal)
	if err != nil {
		return "", fmt.Errorf("failed to generate secure OTP: %w", err)
	}

	return fmt.Sprintf("%04d", n.Int64()), nil
}
