package handler

// Tests de integración HTTP para handlers de autenticación y streaming HLS.
// Se utilizan fakes en memoria para DB y Cache, sin dependencias externas (PostgreSQL/Redis).

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"test_zapping/internal/auth"
	"test_zapping/internal/hls"
)

// ─────────────────────────────────────────────
// Fakes en memoria: DB y Cache
// ─────────────────────────────────────────────

type usuarioFake struct {
	ID           string
	Name         string
	Email        string
	PasswordHash string
	CreatedAt    string
}

type storeFake struct {
	usuarios      map[string]*usuarioFake // clave: email
	refreshTokens map[string]bool         // clave: userID:tokenID
	playlists     map[string]string
}

func newStoreFake() *storeFake {
	return &storeFake{
		usuarios:      make(map[string]*usuarioFake),
		refreshTokens: make(map[string]bool),
		playlists:     make(map[string]string),
	}
}

// ─────────────────────────────────────────────
// handlersDeTest: replica los handlers sin depender de db.DB / cache.CacheService
// ─────────────────────────────────────────────

type handlersDeTest struct {
	store *storeFake
	sm    *hls.StreamManager
}

func nuevoHandlerDeTest(t *testing.T) *handlersDeTest {
	t.Helper()
	sm, _ := hls.NewStreamManager("../../media/segments")
	return &handlersDeTest{
		store: newStoreFake(),
		sm:    sm,
	}
}

// Registro de usuario
func (h *handlersDeTest) registro(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	var dto RegistroDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		responderJSONError(w, http.StatusBadRequest, "JSON inválido")
		return
	}
	dto.Email = strings.TrimSpace(strings.ToLower(dto.Email))

	if dto.Nombre == "" || dto.Email == "" || len(dto.Password) < 6 {
		responderJSONError(w, http.StatusBadRequest, "Datos incompletos o contraseña muy corta (mínimo 6 caracteres)")
		return
	}
	if _, existe := h.store.usuarios[dto.Email]; existe {
		responderJSONError(w, http.StatusConflict, "El correo electrónico ya se encuentra registrado")
		return
	}

	hash, err := auth.HashPassword(dto.Password)
	if err != nil {
		responderJSONError(w, http.StatusInternalServerError, "Error procesando credenciales")
		return
	}

	id := "uid-" + dto.Email
	h.store.usuarios[dto.Email] = &usuarioFake{
		ID: id, Name: dto.Nombre, Email: dto.Email,
		PasswordHash: hash, CreatedAt: time.Now().Format(time.RFC3339),
	}

	tokens, err := auth.GenerateTokenPair(id, dto.Email, dto.Nombre)
	if err != nil {
		responderJSONError(w, http.StatusInternalServerError, "Error generando tokens")
		return
	}
	h.store.refreshTokens[id+":"+tokens.RefreshTokenID] = true

	responderJSON(w, http.StatusCreated, map[string]interface{}{
		"access_token":  tokens.AccessToken,
		"refresh_token": tokens.RefreshToken,
	})
}

// Login de usuario
func (h *handlersDeTest) login(w http.ResponseWriter, r *http.Request) {
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

	u, ok := h.store.usuarios[dto.Email]
	if !ok || !auth.CheckPasswordHash(dto.Password, u.PasswordHash) {
		responderJSONError(w, http.StatusUnauthorized, "Credenciales incorrectas")
		return
	}

	tokens, err := auth.GenerateTokenPair(u.ID, u.Email, u.Name)
	if err != nil {
		responderJSONError(w, http.StatusInternalServerError, "Error al generar sesión")
		return
	}
	h.store.refreshTokens[u.ID+":"+tokens.RefreshTokenID] = true

	responderJSON(w, http.StatusOK, map[string]interface{}{
		"access_token":  tokens.AccessToken,
		"refresh_token": tokens.RefreshToken,
	})
}

// Playlist HLS en vivo
func (h *handlersDeTest) playlist(w http.ResponseWriter, r *http.Request) {
	canalID := r.URL.Query().Get("channel")
	if canalID == "" {
		canalID = "kuspid-sports"
	}
	canal, ok := h.sm.GetChannel(canalID)
	if !ok {
		http.Error(w, "Canal no encontrado", http.StatusNotFound)
		return
	}
	configurarEncabezadosCacheHLS(w)
	w.Write([]byte(canal.GenerateM3U8()))
}

// Segmento .ts
func (h *handlersDeTest) segmento(w http.ResponseWriter, r *http.Request) {
	partes := strings.Split(r.URL.Path, "/")
	nombreArchivo := partes[len(partes)-1]
	if !strings.HasSuffix(nombreArchivo, ".ts") {
		http.Error(w, "Segmento inválido", http.StatusBadRequest)
		return
	}
	datos, err := h.sm.GetSegmentContent(nombreArchivo)
	if err != nil {
		http.Error(w, "Segmento no encontrado", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "video/mp2t")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Write(datos)
}

// ─────────────────────────────────────────────
// Tests: /api/auth/register
// ─────────────────────────────────────────────

func TestRegistro_Exito(t *testing.T) {
	h := nuevoHandlerDeTest(t)

	body := `{"name":"Alex Kuspid","email":"alex@kuspid.tv","password":"secret123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.registro(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("Esperado 201 Created, obtenido %d — body: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("Respuesta JSON inválida: %v", err)
	}
	if resp["access_token"] == "" || resp["access_token"] == nil {
		t.Error("access_token ausente o vacío en respuesta de registro")
	}
	if resp["refresh_token"] == "" || resp["refresh_token"] == nil {
		t.Error("refresh_token ausente o vacío en respuesta de registro")
	}
}

func TestRegistro_CamposIncompletos(t *testing.T) {
	casos := []struct {
		nombre string
		body   string
	}{
		{"sin_email", `{"name":"Alex","password":"secret123"}`},
		{"password_corta", `{"name":"Alex","email":"a@b.com","password":"123"}`},
		{"sin_nombre", `{"email":"a@b.com","password":"secret123"}`},
		{"json_vacio", `{}`},
	}

	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			h := nuevoHandlerDeTest(t)
			req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString(c.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			h.registro(rec, req)
			if rec.Code != http.StatusBadRequest {
				t.Errorf("[%s] Esperado 400, obtenido %d", c.nombre, rec.Code)
			}
		})
	}
}

func TestRegistro_EmailDuplicado(t *testing.T) {
	h := nuevoHandlerDeTest(t)
	body := `{"name":"Alex","email":"dup@kuspid.tv","password":"secret123"}`

	// Primer registro exitoso
	req1 := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString(body))
	req1.Header.Set("Content-Type", "application/json")
	rec1 := httptest.NewRecorder()
	h.registro(rec1, req1)
	if rec1.Code != http.StatusCreated {
		t.Fatalf("Primer registro falló con %d", rec1.Code)
	}

	// Segundo registro con el mismo email → 409
	req2 := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString(body))
	req2.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	h.registro(rec2, req2)
	if rec2.Code != http.StatusConflict {
		t.Errorf("Esperado 409 Conflict, obtenido %d", rec2.Code)
	}
}

func TestRegistro_MetodoNoPermitido(t *testing.T) {
	h := nuevoHandlerDeTest(t)
	req := httptest.NewRequest(http.MethodGet, "/api/auth/register", nil)
	rec := httptest.NewRecorder()
	h.registro(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("Esperado 405, obtenido %d", rec.Code)
	}
}

// ─────────────────────────────────────────────
// Tests: /api/auth/login
// ─────────────────────────────────────────────

func TestLogin_Exito(t *testing.T) {
	h := nuevoHandlerDeTest(t)

	// Registrar usuario de prueba
	regBody := `{"name":"Alex","email":"login@kuspid.tv","password":"secret123"}`
	req0 := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString(regBody))
	req0.Header.Set("Content-Type", "application/json")
	h.registro(httptest.NewRecorder(), req0)

	// Login correcto
	loginBody := `{"email":"login@kuspid.tv","password":"secret123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(loginBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.login(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Esperado 200 OK, obtenido %d — body: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("Respuesta JSON inválida: %v", err)
	}
	if resp["access_token"] == nil {
		t.Error("access_token ausente en respuesta de login")
	}
}

func TestLogin_PasswordIncorrecta(t *testing.T) {
	h := nuevoHandlerDeTest(t)

	// Registrar usuario
	regBody := `{"name":"Alex","email":"wrong@kuspid.tv","password":"correcta123"}`
	req0 := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString(regBody))
	req0.Header.Set("Content-Type", "application/json")
	h.registro(httptest.NewRecorder(), req0)

	// Login con contraseña incorrecta → 401
	loginBody := `{"email":"wrong@kuspid.tv","password":"INCORRECTA"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(loginBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.login(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Esperado 401 Unauthorized, obtenido %d", rec.Code)
	}
}

func TestLogin_UsuarioInexistente(t *testing.T) {
	h := nuevoHandlerDeTest(t)

	loginBody := `{"email":"fantasma@kuspid.tv","password":"cualquiera"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(loginBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.login(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Esperado 401 Unauthorized, obtenido %d", rec.Code)
	}
}

// ─────────────────────────────────────────────
// Tests: Streaming HLS
// ─────────────────────────────────────────────

func TestPlaylist_RetornaM3U8Valido(t *testing.T) {
	h := nuevoHandlerDeTest(t)

	req := httptest.NewRequest(http.MethodGet, "/live.m3u8?channel=kuspid-sports", nil)
	rec := httptest.NewRecorder()
	h.playlist(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Esperado 200 OK, obtenido %d", rec.Code)
	}

	body := rec.Body.String()
	partes := []string{"#EXTM3U", "#EXT-X-VERSION", "#EXT-X-MEDIA-SEQUENCE", "#EXTINF"}
	for _, parte := range partes {
		if !strings.Contains(body, parte) {
			t.Errorf("Manifiesto m3u8 no contiene '%s'", parte)
		}
	}

	ct := rec.Header().Get("Content-Type")
	if ct != "application/vnd.apple.mpegurl" {
		t.Errorf("Content-Type incorrecto: %s", ct)
	}
}

// TestPlaylist_VentanaTiene3Segmentos verifica que el manifiesto contiene exactamente 3 segmentos (30s = 3 × 10s).
func TestPlaylist_VentanaTiene3Segmentos(t *testing.T) {
	h := nuevoHandlerDeTest(t)

	req := httptest.NewRequest(http.MethodGet, "/live.m3u8?channel=kuspid-sports", nil)
	rec := httptest.NewRecorder()
	h.playlist(rec, req)

	body := rec.Body.String()
	conteo := strings.Count(body, "#EXTINF")
	if conteo != 3 {
		t.Errorf("El manifiesto debe tener exactamente 3 segmentos #EXTINF (30s = 3 × 10s), tiene %d", conteo)
	}
}

func TestSegmento_ContentTypeCorrecto(t *testing.T) {
	h := nuevoHandlerDeTest(t)

	req := httptest.NewRequest(http.MethodGet, "/api/stream/segments/segment0.ts", nil)
	rec := httptest.NewRecorder()
	h.segmento(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Esperado 200 OK para segmento .ts, obtenido %d", rec.Code)
	}
	if rec.Header().Get("Content-Type") != "video/mp2t" {
		t.Errorf("Content-Type incorrecto: %s", rec.Header().Get("Content-Type"))
	}
	if rec.Body.Len() == 0 {
		t.Error("El cuerpo del segmento .ts no debe estar vacío")
	}
}

func TestSegmento_ExtensionInvalidaRetorna400(t *testing.T) {
	h := nuevoHandlerDeTest(t)

	casos := []string{"/api/stream/segments/virus.exe", "/api/stream/segments/foto.jpg", "/api/stream/segments/datos.json"}
	for _, url := range casos {
		t.Run(url, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, url, nil)
			rec := httptest.NewRecorder()
			h.segmento(rec, req)
			if rec.Code != http.StatusBadRequest {
				t.Errorf("[%s] Esperado 400 Bad Request, obtenido %d", url, rec.Code)
			}
		})
	}
}
