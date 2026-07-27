package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/google/uuid"
	"test_zapping/internal/auth"
	"test_zapping/internal/cache"
	"test_zapping/internal/db"
	"test_zapping/internal/hls"
	"test_zapping/internal/middleware"
	"test_zapping/internal/ws"
)

type Handler struct {
	db            *db.DB
	cache         *cache.CacheService
	streamManager *hls.StreamManager
	wsHub         *ws.Hub
	staticDir     string
}

func NewHandler(database *db.DB, cacheSrv *cache.CacheService, sm *hls.StreamManager, hub *ws.Hub, staticDir string) *Handler {
	return &Handler{
		db:            database,
		cache:         cacheSrv,
		streamManager: sm,
		wsHub:         hub,
		staticDir:     staticDir,
	}
}

func (h *Handler) WSHub() *ws.Hub {
	return h.wsHub
}

type RegistroDTO struct {
	Nombre   string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginDTO struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RespuestaAuth struct {
	Usuario      *db.User `json:"user"`
	AccessToken  string   `json:"access_token"`
	RefreshToken string   `json:"refresh_token"`
}

type RevocarSesionDTO struct {
	UserID string `json:"user_id"`
}

func (h *Handler) HandleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	var dto RegistroDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		responderJSONError(w, http.StatusBadRequest, "Estructura JSON inválida")
		return
	}

	dto.Email = strings.TrimSpace(strings.ToLower(dto.Email))
	if dto.Nombre == "" || dto.Email == "" || len(dto.Password) < 6 {
		responderJSONError(w, http.StatusBadRequest, "Datos incompletos o contraseña muy corta (mínimo 6 caracteres)")
		return
	}

	existente, _ := h.db.GetUserByEmail(dto.Email)
	if existente != nil {
		responderJSONError(w, http.StatusConflict, "El correo electrónico ya se encuentra registrado")
		return
	}

	hash, err := auth.HashPassword(dto.Password)
	if err != nil {
		responderJSONError(w, http.StatusInternalServerError, "Error procesando credenciales")
		return
	}

	userID := uuid.New().String()
	usuario, err := h.db.CreateUser(userID, dto.Nombre, dto.Email, hash)
	if err != nil {
		responderJSONError(w, http.StatusInternalServerError, "No se pudo registrar el usuario")
		return
	}

	tokens, err := auth.GenerateTokenPair(usuario.ID, usuario.Email, usuario.Name)
	if err != nil {
		responderJSONError(w, http.StatusInternalServerError, "Error generando tokens de acceso")
		return
	}

	_ = h.cache.StoreRefreshToken(r.Context(), usuario.ID, tokens.RefreshTokenID, auth.RefreshTokenDuration)
	establecerCookiesAuth(w, tokens)

	responderJSON(w, http.StatusCreated, RespuestaAuth{
		Usuario:      usuario,
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	})
}

func (h *Handler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	var dto LoginDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		responderJSONError(w, http.StatusBadRequest, "JSON inválido")
		return
	}

	dto.Email = strings.TrimSpace(strings.ToLower(dto.Email))
	usuario, err := h.db.GetUserByEmail(dto.Email)
	if err != nil || usuario == nil {
		responderJSONError(w, http.StatusUnauthorized, "Credenciales incorrectas")
		return
	}

	if !auth.CheckPasswordHash(dto.Password, usuario.PasswordHash) {
		responderJSONError(w, http.StatusUnauthorized, "Credenciales incorrectas")
		return
	}

	tokens, err := auth.GenerateTokenPair(usuario.ID, usuario.Email, usuario.Name)
	if err != nil {
		responderJSONError(w, http.StatusInternalServerError, "Error al generar sesión")
		return
	}

	_ = h.cache.StoreRefreshToken(r.Context(), usuario.ID, tokens.RefreshTokenID, auth.RefreshTokenDuration)
	establecerCookiesAuth(w, tokens)

	responderJSON(w, http.StatusOK, RespuestaAuth{
		Usuario:      usuario,
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	})
}

func (h *Handler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(middleware.UserContextKey).(*auth.Claims)
	if ok && claims != nil {
		_ = h.cache.RevokeAllUserTokens(r.Context(), claims.UserID)
	}

	limpiarCookiesAuth(w)
	responderJSON(w, http.StatusOK, map[string]string{"message": "Sesión cerrada correctamente"})
}

func (h *Handler) HandleMe(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(middleware.UserContextKey).(*auth.Claims)
	if !ok || claims == nil {
		responderJSONError(w, http.StatusUnauthorized, "Sesión no encontrada")
		return
	}

	usuario, err := h.db.GetUserByID(claims.UserID)
	if err != nil || usuario == nil {
		responderJSONError(w, http.StatusNotFound, "Usuario no encontrado")
		return
	}

	conteoSesiones := h.cache.CountActiveUserSessions(r.Context(), claims.UserID)

	responderJSON(w, http.StatusOK, map[string]interface{}{
		"user":                   usuario,
		"active_sessions_count": conteoSesiones,
		"has_other_devices":     conteoSesiones > 1,
	})
}

func (h *Handler) HandleGetActiveSessions(w http.ResponseWriter, r *http.Request) {
	sesiones, err := h.cache.GetAllActiveSessions(r.Context())
	if err != nil {
		responderJSONError(w, http.StatusInternalServerError, "Error consultando sesiones en Redis")
		return
	}

	responderJSON(w, http.StatusOK, map[string]interface{}{
		"active_users_sessions": sesiones,
		"total_active_sessions": len(sesiones),
	})
}

func (h *Handler) HandleRevokeUserSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	var dto RevocarSesionDTO
	_ = json.NewDecoder(r.Body).Decode(&dto)

	if dto.UserID == "" {
		dto.UserID = r.URL.Query().Get("user_id")
	}

	if dto.UserID == "" {
		claims, ok := r.Context().Value(middleware.UserContextKey).(*auth.Claims)
		if ok && claims != nil {
			dto.UserID = claims.UserID
		}
	}

	if dto.UserID == "" {
		responderJSONError(w, http.StatusBadRequest, "ID de usuario requerido")
		return
	}

	err := h.cache.RevokeAllUserTokens(r.Context(), dto.UserID)
	if err != nil {
		responderJSONError(w, http.StatusInternalServerError, "Error revocando sesiones")
		return
	}

	responderJSON(w, http.StatusOK, map[string]string{
		"message": fmt.Sprintf("Todas las sesiones del usuario %s han sido revocadas exitosamente.", dto.UserID),
	})
}

func (h *Handler) HandleListChannels(w http.ResponseWriter, r *http.Request) {
	canales := h.streamManager.ListChannels()
	responderJSON(w, http.StatusOK, canales)
}

func (h *Handler) HandleMasterPlaylist(w http.ResponseWriter, r *http.Request) {
	canalID := extraerCanalID(r)
	canal, ok := h.streamManager.GetChannel(canalID)
	if !ok {
		http.Error(w, "Canal no disponible", http.StatusNotFound)
		return
	}

	configurarEncabezadosCacheHLS(w)
	w.Write([]byte(canal.GenerateM3U8()))
}

func (h *Handler) HandleLivePlaylist(w http.ResponseWriter, r *http.Request) {
	canalID := r.URL.Query().Get("channel")
	if canalID == "" {
		canalID = extraerCanalID(r)
	}

	cacheKey := fmt.Sprintf("playlist:%s", canalID)
	if cachedPlaylist, ok := h.cache.GetCachedPlaylist(r.Context(), cacheKey); ok {
		configurarEncabezadosCacheHLS(w)
		w.Header().Set("X-Cache-Status", "HIT-REDIS")
		w.Write([]byte(cachedPlaylist))
		return
	}

	canal, ok := h.streamManager.GetChannel(canalID)
	if !ok {
		http.Error(w, "Canal no encontrado", http.StatusNotFound)
		return
	}

	playlistStr := canal.GenerateM3U8()

	_ = h.cache.CachePlaylist(r.Context(), cacheKey, playlistStr, 2*time.Second)

	configurarEncabezadosCacheHLS(w)
	w.Header().Set("X-Cache-Status", "MISS-GENERATE")
	w.Write([]byte(playlistStr))
}

func (h *Handler) HandleSegment(w http.ResponseWriter, r *http.Request) {
	nombreArchivo := filepath.Base(r.URL.Path)
	if !strings.HasSuffix(nombreArchivo, ".ts") {
		http.Error(w, "Segmento inválido", http.StatusBadRequest)
		return
	}

	if datosCache, ok := h.cache.GetCachedSegment(r.Context(), nombreArchivo); ok {
		w.Header().Set("Content-Type", "video/mp2t")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		w.Header().Set("X-Cache-Status", "HIT-REDIS")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(datosCache)))
		w.Write(datosCache)
		return
	}

	datos, err := h.streamManager.GetSegmentContent(nombreArchivo)
	if err != nil {
		http.Error(w, "Segmento no encontrado", http.StatusNotFound)
		return
	}

	_ = h.cache.CacheSegment(r.Context(), nombreArchivo, datos, 10*time.Minute)

	w.Header().Set("Content-Type", "video/mp2t")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Header().Set("X-Cache-Status", "MISS-DISK")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(datos)))
	w.Write(datos)
}

func (h *Handler) HandleMetrics(w http.ResponseWriter, r *http.Request) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	allocMB := float64(m.Alloc) / 1024 / 1024

	responderJSON(w, http.StatusOK, map[string]interface{}{
		"memory_alloc_mb":       fmt.Sprintf("%.2f", allocMB),
		"goroutines":            runtime.NumGoroutine(),
		"active_ws_connections": h.wsHub.ActiveClientsCount(),
		"timestamp":             time.Now().Unix(),
	})
}

func responderJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func responderJSONError(w http.ResponseWriter, status int, mensaje string) {
	responderJSON(w, status, map[string]string{"error": mensaje})
}

func establecerCookiesAuth(w http.ResponseWriter, tokens *auth.TokenPair) {
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    tokens.AccessToken,
		Path:     "/",
		Expires:  time.Now().Add(auth.AccessTokenDuration),
		HttpOnly: false,
		SameSite: http.SameSiteLaxMode,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    tokens.RefreshToken,
		Path:     "/",
		Expires:  time.Now().Add(auth.RefreshTokenDuration),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func limpiarCookiesAuth(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		HttpOnly: false,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
	})
}

func extraerCanalID(r *http.Request) string {
	partes := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	for _, parte := range partes {
		if parte != "" && parte != "api" && parte != "stream" && parte != "live.m3u8" && parte != "master.m3u8" {
			return parte
		}
	}
	return "kuspid-sports"
}

func configurarEncabezadosCacheHLS(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
}
