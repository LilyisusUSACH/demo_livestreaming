package auth

// Tests adicionales del paquete auth:
// - Token expirado rechazado
// - Token con firma alterada rechazado
// - Hash bcrypt correcto/incorrecto
// - Unicidad de JTI entre pares de tokens generados consecutivamente

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TestToken_Expirado verifica que un token firmado con expiración en el pasado es rechazado.
func TestToken_Expirado(t *testing.T) {
	claims := &Claims{
		UserID:    "uid-exp",
		Email:     "exp@kuspid.tv",
		Name:      "Expirado",
		TokenType: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-5 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-20 * time.Minute)),
		},
	}
	tokenStr, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(JwtSecret)
	if err != nil {
		t.Fatalf("Error generando token expirado de prueba: %v", err)
	}

	_, err = ValidateToken(tokenStr)
	if err == nil {
		t.Error("ValidateToken debería rechazar un token expirado, pero retornó nil error")
	}
}

// TestToken_FirmaAlterada verifica que un token con firma modificada es rechazado.
func TestToken_FirmaAlterada(t *testing.T) {
	par, err := GenerateTokenPair("uid-fake", "fake@kuspid.tv", "Fake")
	if err != nil {
		t.Fatalf("Error generando token pair: %v", err)
	}

	// Alterar el último carácter de la firma
	tokenStr := par.AccessToken
	alterado := tokenStr[:len(tokenStr)-3] + "XYZ"

	_, err = ValidateToken(alterado)
	if err == nil {
		t.Error("ValidateToken debería rechazar un token con firma alterada")
	}
}

// TestToken_FormatoInvalido verifica que strings arbitrarios son rechazados.
func TestToken_FormatoInvalido(t *testing.T) {
	casos := []string{"", "no.es.jwt", "Bearer token", "12345"}
	for _, c := range casos {
		_, err := ValidateToken(c)
		if err == nil {
			t.Errorf("ValidateToken debería rechazar '%s', pero pasó", c)
		}
	}
}

// TestHashPassword_VerificacionCorrecta verifica que CheckPasswordHash devuelve true con la contraseña correcta.
func TestHashPassword_VerificacionCorrecta(t *testing.T) {
	password := "SuperSecret2026!"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword falló: %v", err)
	}

	if !CheckPasswordHash(password, hash) {
		t.Error("CheckPasswordHash debería retornar true con contraseña correcta")
	}
}

// TestHashPassword_VerificacionIncorrecta verifica que CheckPasswordHash devuelve false con contraseña incorrecta.
func TestHashPassword_VerificacionIncorrecta(t *testing.T) {
	hash, _ := HashPassword("correcta123")
	if CheckPasswordHash("incorrecta999", hash) {
		t.Error("CheckPasswordHash debería retornar false con contraseña incorrecta")
	}
}

// TestToken_JTIUnico verifica que dos pares de tokens generados consecutivamente tienen JTIs diferentes.
func TestToken_JTIUnico(t *testing.T) {
	par1, err := GenerateTokenPair("uid-a", "a@kuspid.tv", "A")
	if err != nil {
		t.Fatalf("Error generando par1: %v", err)
	}
	par2, err := GenerateTokenPair("uid-a", "a@kuspid.tv", "A")
	if err != nil {
		t.Fatalf("Error generando par2: %v", err)
	}

	if par1.RefreshTokenID == par2.RefreshTokenID {
		t.Error("RefreshTokenID debe ser único entre pares de tokens distintos")
	}
}

// TestToken_ClaimsTipoAcceso verifica que el access token tiene tipo "access" y el refresh tiene tipo "refresh".
func TestToken_ClaimsTipoAcceso(t *testing.T) {
	par, err := GenerateTokenPair("uid-tipos", "tipos@kuspid.tv", "Tipos")
	if err != nil {
		t.Fatalf("Error generando token pair: %v", err)
	}

	accessClaims, err := ValidateToken(par.AccessToken)
	if err != nil {
		t.Fatalf("ValidateToken falló en access token: %v", err)
	}
	if accessClaims.TokenType != "access" {
		t.Errorf("Access token debería tener TokenType 'access', tiene '%s'", accessClaims.TokenType)
	}

	refreshClaims, err := ValidateToken(par.RefreshToken)
	if err != nil {
		t.Fatalf("ValidateToken falló en refresh token: %v", err)
	}
	if refreshClaims.TokenType != "refresh" {
		t.Errorf("Refresh token debería tener TokenType 'refresh', tiene '%s'", refreshClaims.TokenType)
	}
}

// TestToken_ExpiresIn verifica que ExpiresIn corresponde a la duración del access token (15 minutos = 900s).
func TestToken_ExpiresIn(t *testing.T) {
	par, err := GenerateTokenPair("uid-exp2", "exp2@kuspid.tv", "Exp2")
	if err != nil {
		t.Fatalf("Error generando token pair: %v", err)
	}
	if par.ExpiresIn != 900 {
		t.Errorf("ExpiresIn debería ser 900 segundos (15 min), obtenido %d", par.ExpiresIn)
	}
}

// TestToken_EmailEnClaims verifica que el email queda correctamente embebido en los claims del token.
func TestToken_EmailEnClaims(t *testing.T) {
	email := "claims@kuspid.tv"
	par, err := GenerateTokenPair("uid-claims", email, "Claims")
	if err != nil {
		t.Fatalf("Error generando token pair: %v", err)
	}

	claims, err := ValidateToken(par.AccessToken)
	if err != nil {
		t.Fatalf("ValidateToken falló: %v", err)
	}
	if claims.Email != email {
		t.Errorf("Email en claims incorrecto: esperado '%s', obtenido '%s'", email, claims.Email)
	}
	if !strings.Contains(claims.Email, "@") {
		t.Error("El email en claims no parece válido")
	}
}
