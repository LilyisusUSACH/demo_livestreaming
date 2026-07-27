package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"test_zapping/internal/auth"
)

type contextKey string

const UserContextKey contextKey = "user_claims"

// CacheAuth define los métodos de caché requeridos por el middleware de autenticación.
// Permite la inyección de fakes para tests sin dependencia a Redis.
type CacheAuth interface {
	CountActiveUserSessions(ctx context.Context, userID string) int
	IsRefreshTokenValid(ctx context.Context, userID, tokenID string) bool
	StoreRefreshToken(ctx context.Context, userID, tokenID string, ttl time.Duration) error
}

func AuthMiddleware(cacheSvc CacheAuth) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var accessTokenStr string
			var refreshTokenStr string

			// 1. Read Access Token from Cookie / Header / Query
			cookie, err := r.Cookie("auth_token")
			if err == nil && cookie.Value != "" {
				accessTokenStr = cookie.Value
			}
			if accessTokenStr == "" {
				authHeader := r.Header.Get("Authorization")
				if strings.HasPrefix(authHeader, "Bearer ") {
					accessTokenStr = strings.TrimPrefix(authHeader, "Bearer ")
				}
			}
			if accessTokenStr == "" {
				accessTokenStr = r.URL.Query().Get("token")
			}

			// 2. Read Refresh Token from Cookie / Header / Query
			rCookie, rErr := r.Cookie("refresh_token")
			if rErr == nil && rCookie.Value != "" {
				refreshTokenStr = rCookie.Value
			}
			if refreshTokenStr == "" {
				refreshTokenStr = r.Header.Get("X-Refresh-Token")
			}
			if refreshTokenStr == "" {
				refreshTokenStr = r.URL.Query().Get("refresh_token")
			}

			// 3. Try validating Access Token & verify active sessions in Redis
			claims, err := auth.ValidateToken(accessTokenStr)
			if err == nil && claims.TokenType == "access" {
				// Verify Redis has active session tokens for this user (if revoked by "Forzar salida", return 401!)
				if cacheSvc.CountActiveUserSessions(r.Context(), claims.UserID) > 0 {
					ctx := context.WithValue(r.Context(), UserContextKey, claims)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}

			// 4. Access Token is expired/missing -> Try auto-renewal using Refresh Token & Redis validation
			if refreshTokenStr != "" {
				refreshClaims, rErr := auth.ValidateToken(refreshTokenStr)
				if rErr == nil && refreshClaims.TokenType == "refresh" {
					// Check Redis to verify Refresh Token is NOT revoked
					if cacheSvc.IsRefreshTokenValid(r.Context(), refreshClaims.UserID, refreshClaims.TokenID) {
						// Auto-renew Access Token
						newTokenPair, genErr := auth.GenerateTokenPair(refreshClaims.UserID, refreshClaims.Email, refreshClaims.Name)
						if genErr == nil {
							// Store newly generated refresh token session in Redis
							_ = cacheSvc.StoreRefreshToken(r.Context(), refreshClaims.UserID, newTokenPair.RefreshTokenID, auth.RefreshTokenDuration)

							// Return new token in X-New-Token header
							w.Header().Set("X-New-Token", newTokenPair.AccessToken)
							w.Header().Set("Access-Control-Expose-Headers", "X-New-Token")

							// Update auth_token cookie
							http.SetCookie(w, &http.Cookie{
								Name:     "auth_token",
								Value:    newTokenPair.AccessToken,
								Path:     "/",
								Expires:  time.Now().Add(auth.AccessTokenDuration),
								HttpOnly: true,
								SameSite: http.SameSiteLaxMode,
							})

							// Update active claims and proceed!
							newClaims, _ := auth.ValidateToken(newTokenPair.AccessToken)
							ctx := context.WithValue(r.Context(), UserContextKey, newClaims)
							next.ServeHTTP(w, r.WithContext(ctx))
							return
						}
					}
				}
			}

			// 5. Unauthenticated / Revoked Token
			if strings.HasPrefix(r.URL.Path, "/api/") {
				http.Error(w, `{"error":"Acceso no autorizado. Sesión revocada o expirada."}`, http.StatusUnauthorized)
				return
			}
			http.Redirect(w, r, "/login", http.StatusSeeOther)
		})
	}
}
