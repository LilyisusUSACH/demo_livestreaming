package middleware

// Tests de integración del AuthMiddleware JWT.
// Verifica que tokens válidos pasan, expirados retornan 401, sesiones revocadas son rechazadas
// y el token tipo "refresh" no puede usarse como access token directamente.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"test_zapping/internal/auth"
)

// ─────────────────────────────────────────────
// cacheFake: implementa la interfaz CacheAuth sin Redis
// ─────────────────────────────────────────────

type cacheFake struct {
	sesiones     map[string]int  // userID → sesiones activas
	tokenValidos map[string]bool // "userID:tokenID" → válido
}

func newCacheFake() *cacheFake {
	return &cacheFake{
		sesiones:     make(map[string]int),
		tokenValidos: make(map[string]bool),
	}
}

func (c *cacheFake) CountActiveUserSessions(_ context.Context, userID string) int {
	return c.sesiones[userID]
}

func (c *cacheFake) IsRefreshTokenValid(_ context.Context, userID, tokenID string) bool {
	return c.tokenValidos[userID+":"+tokenID]
}

func (c *cacheFake) StoreRefreshToken(_ context.Context, userID, tokenID string, _ time.Duration) error {
	c.tokenValidos[userID+":"+tokenID] = true
	c.sesiones[userID]++
	return nil
}

// ─────────────────────────────────────────────
// Helper: genera token con duración personalizada para tests
// ─────────────────────────────────────────────

func generarTokenTest(userID, email, nombre, tipo string, duracion time.Duration) (string, error) {
	claims := &auth.Claims{
		UserID:    userID,
		Email:     email,
		Name:      nombre,
		TokenType: tipo,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(duracion)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   userID,
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(auth.JwtSecret)
}

// Handler centinela: confirma que los claims llegaron al contexto
var handlerOK = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(UserContextKey).(*auth.Claims)
	if !ok || claims == nil {
		http.Error(w, "claims ausentes", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(claims.UserID))
})

// ─────────────────────────────────────────────
// Tests
// ─────────────────────────────────────────────

// TestAuthMiddleware_TokenValido verifica que un access token válido con sesión activa pasa el middleware.
func TestAuthMiddleware_TokenValido(t *testing.T) {
	cache := newCacheFake()
	userID := "uid-001"
	token, _ := generarTokenTest(userID, "alex@kuspid.tv", "Alex", "access", 15*time.Minute)
	cache.sesiones[userID] = 1

	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	AuthMiddleware(cache)(handlerOK).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Esperado 200 OK con token válido, obtenido %d", rec.Code)
	}
	if rec.Body.String() != userID {
		t.Errorf("UserID en contexto incorrecto: %s", rec.Body.String())
	}
}

// TestAuthMiddleware_TokenViaQuery verifica que el token se acepta como ?token= en query string (WebSocket).
func TestAuthMiddleware_TokenViaQuery(t *testing.T) {
	cache := newCacheFake()
	userID := "uid-002"
	cache.sesiones[userID] = 1
	token, _ := generarTokenTest(userID, "ws@kuspid.tv", "WS", "access", 15*time.Minute)

	req := httptest.NewRequest(http.MethodGet, "/api/ws?token="+token, nil)
	rec := httptest.NewRecorder()

	AuthMiddleware(cache)(handlerOK).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Esperado 200 OK con token via ?token=, obtenido %d", rec.Code)
	}
}

// TestAuthMiddleware_TokenViaCookie verifica que el token se acepta desde la cookie auth_token.
func TestAuthMiddleware_TokenViaCookie(t *testing.T) {
	cache := newCacheFake()
	userID := "uid-003"
	cache.sesiones[userID] = 1
	token, _ := generarTokenTest(userID, "cookie@kuspid.tv", "Cookie", "access", 15*time.Minute)

	req := httptest.NewRequest(http.MethodGet, "/player", nil)
	req.AddCookie(&http.Cookie{Name: "auth_token", Value: token})
	rec := httptest.NewRecorder()

	AuthMiddleware(cache)(handlerOK).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Esperado 200 OK con token en cookie, obtenido %d", rec.Code)
	}
}

// TestAuthMiddleware_TokenExpirado verifica que un token expirado retorna 401.
func TestAuthMiddleware_TokenExpirado(t *testing.T) {
	cache := newCacheFake()
	userID := "uid-004"
	cache.sesiones[userID] = 1
	token, _ := generarTokenTest(userID, "expired@kuspid.tv", "Expired", "access", -1*time.Minute)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	AuthMiddleware(cache)(handlerOK).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Esperado 401 Unauthorized con token expirado, obtenido %d", rec.Code)
	}
}

// TestAuthMiddleware_SinTokenEnAPI verifica que /api/ sin token retorna 401.
func TestAuthMiddleware_SinTokenEnAPI(t *testing.T) {
	cache := newCacheFake()
	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	rec := httptest.NewRecorder()

	AuthMiddleware(cache)(handlerOK).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Esperado 401 Unauthorized sin token en /api/, obtenido %d", rec.Code)
	}
}

// TestAuthMiddleware_SinTokenEnPlayer verifica que /player sin token redirige a /login.
func TestAuthMiddleware_SinTokenEnPlayer(t *testing.T) {
	cache := newCacheFake()
	req := httptest.NewRequest(http.MethodGet, "/player", nil)
	rec := httptest.NewRecorder()

	AuthMiddleware(cache)(handlerOK).ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Errorf("Esperado 303 SeeOther (redirect a /login), obtenido %d", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/login" {
		t.Errorf("Location incorrecto: esperado '/login', obtenido '%s'", loc)
	}
}

// TestAuthMiddleware_SesionRevocada verifica que un token válido pero sin sesión en Redis retorna 401.
func TestAuthMiddleware_SesionRevocada(t *testing.T) {
	cache := newCacheFake()
	userID := "uid-005"
	// Sin sesiones activas → cache.sesiones[userID] = 0 (default)
	token, _ := generarTokenTest(userID, "revoked@kuspid.tv", "Revoked", "access", 15*time.Minute)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	AuthMiddleware(cache)(handlerOK).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Esperado 401 con sesión revocada (0 sesiones en Redis), obtenido %d", rec.Code)
	}
}

// TestAuthMiddleware_RefreshTokenNoAutentica verifica que un refresh token NO pasa el middleware como access.
func TestAuthMiddleware_RefreshTokenNoAutentica(t *testing.T) {
	cache := newCacheFake()
	userID := "uid-006"
	cache.sesiones[userID] = 1
	// Generar token con tipo "refresh"
	refreshToken, _ := generarTokenTest(userID, "tipo@kuspid.tv", "Tipo", "refresh", 30*24*time.Hour)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+refreshToken)
	rec := httptest.NewRecorder()

	AuthMiddleware(cache)(handlerOK).ServeHTTP(rec, req)

	if rec.Code == http.StatusOK {
		t.Error("Un refresh token NO debe autenticar directamente como access token en el middleware")
	}
}
