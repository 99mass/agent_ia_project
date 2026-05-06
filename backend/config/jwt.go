package config

import (
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var JWTSecret []byte

func InitJWT() {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "unchk-agent-ia-secret-key-2025"
	}
	JWTSecret = []byte(secret)
}

// Claims represents the JWT claims for UNCHK users
type Claims struct {
	UserID  uint   `json:"user_id"`
	Email   string `json:"email"`
	Role    string `json:"role"`
	MustChangePassword bool `json:"must_change_password"`
	jwt.RegisteredClaims
}

// GenerateToken creates a JWT token for an authenticated user
func GenerateToken(userID uint, email, role string, mustChangePassword bool) (string, error) {
	claims := Claims{
		UserID:  userID,
		Email:   email,
		Role:    role,
		MustChangePassword: mustChangePassword,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "unchk-agent-ia",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(JWTSecret)
}

// ValidateToken verifies and parses a JWT token
func ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return JWTSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, jwt.ErrSignatureInvalid
}
