package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// Clave secreta para la firma de tokens JWT
var JwtSecret = []byte("super-secret-zapping-hls-key-2026")

const (
	AccessTokenDuration  = 15 * time.Minute
	RefreshTokenDuration = 30 * 24 * time.Hour
)

// Reivindicaciones (Claims) estructuradas de autenticación en token JWT
type Claims struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	TokenType string `json:"token_type"` // "access" o "refresh"
	TokenID   string `json:"token_id,omitempty"`
	jwt.RegisteredClaims
}

// TokenPair representa el par de Access Token (corto plazo) y Refresh Token (largo plazo)
type TokenPair struct {
	AccessToken    string `json:"access_token"`
	RefreshToken   string `json:"refresh_token"`
	RefreshTokenID string `json:"refresh_token_id"`
	ExpiresIn      int64  `json:"expires_in"`
}

// HashPassword genera un hash seguro bcrypt para la contraseña proporcionada
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPasswordHash compara una contraseña en texto plano contra su hash bcrypt
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateTokenPair construye el par de Access Token y Refresh Token con UUIDs únicos
func GenerateTokenPair(userID, email, name string) (*TokenPair, error) {
	now := time.Now()

	// 1. Generar Access Token (15 minutos)
	accessClaims := &Claims{
		UserID:    userID,
		Email:     email,
		Name:      name,
		TokenType: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(AccessTokenDuration)),
			IssuedAt:  jwt.NewNumericDate(now),
			Subject:   userID,
			ID:        uuid.New().String(),
		},
	}
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString(JwtSecret)
	if err != nil {
		return nil, err
	}

	// 2. Generar Refresh Token de larga duración (30 días)
	refreshTokenID := uuid.New().String()
	refreshClaims := &Claims{
		UserID:    userID,
		Email:     email,
		Name:      name,
		TokenType: "refresh",
		TokenID:   refreshTokenID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(RefreshTokenDuration)),
			IssuedAt:  jwt.NewNumericDate(now),
			Subject:   userID,
			ID:        refreshTokenID,
		},
	}
	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString(JwtSecret)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:    accessToken,
		RefreshToken:   refreshToken,
		RefreshTokenID: refreshTokenID,
		ExpiresIn:      int64(AccessTokenDuration.Seconds()),
	}, nil
}

// ValidateToken decodifica y verifica la validez y firma de un token JWT
func ValidateToken(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("método de firma no esperado")
		}
		return JwtSecret, nil
	})

	if err != nil || !token.Valid {
		return nil, errors.New("token inválido o expirado")
	}

	return claims, nil
}
