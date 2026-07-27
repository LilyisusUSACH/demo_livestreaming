package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"test_zapping/internal/cache"
	"test_zapping/internal/db"
	"test_zapping/internal/handler"
	"test_zapping/internal/hls"
	"test_zapping/internal/middleware"
	"test_zapping/internal/ws"
)

type Config struct {
	Puerto      string
	DBHost      string
	DBPort      string
	DBUser      string
	DBPass      string
	DBName      string
	DBSSLMode   string
	RedisAddr   string
	DirectorIOS string
}

func cargarConfiguracion() Config {
	return Config{
		Puerto:      obtenerEnv("PORT", "8080"),
		DBHost:      obtenerEnv("DB_HOST", "localhost"),
		DBPort:      obtenerEnv("DB_PORT", "5432"),
		DBUser:      obtenerEnv("DB_USER", "kuspid"),
		DBPass:      obtenerEnv("DB_PASSWORD", "kuspid_pass"),
		DBName:      obtenerEnv("DB_NAME", "kuspid_db"),
		DBSSLMode:   obtenerEnv("DB_SSLMODE", "disable"),
		RedisAddr:   obtenerEnv("REDIS_ADDR", "localhost:6379"),
		DirectorIOS: obtenerEnv("SEGMENTS_DIR", filepath.Join("media", "segments")),
	}
}

func main() {
	cfg := cargarConfiguracion()

	log.Println("Iniciando Servidor Kuspid HLS Live Streaming 3D")

	baseDatos := conectarBaseDatos(cfg)
	cacheSrv := cache.NewCacheService(cfg.RedisAddr)

	streamManager, err := hls.NewStreamManager(cfg.DirectorIOS)
	if err != nil {
		log.Fatalf("Error inicializando gestor HLS: %v", err)
	}

	wsHub := ws.NewHub(cacheSrv)
	go wsHub.Run()

	h := handler.NewHandler(baseDatos, cacheSrv, streamManager, wsHub, "web")
	limitadorIP := middleware.NewIPRateLimiter()

	router := construirRouter(h, cacheSrv, limitadorIP)

	servidor := &http.Server{
		Addr:         ":" + cfg.Puerto,
		Handler:      middleware.SecurityHeadersMiddleware(router),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	apagarConGracefulShutdown(servidor, cfg.Puerto)
}

func conectarBaseDatos(cfg Config) *db.DB {
	dbCfg := db.Config{
		Host:     cfg.DBHost,
		Port:     cfg.DBPort,
		User:     cfg.DBUser,
		Password: cfg.DBPass,
		DBName:   cfg.DBName,
		SSLMode:  cfg.DBSSLMode,
	}

	baseDatos, err := db.InitDB(dbCfg)
	if err != nil {
		log.Printf("PostgreSQL no disponible (%v). Reintentando conexion...", err)
		time.Sleep(3 * time.Second)
		baseDatos, err = db.InitDB(dbCfg)
		if err != nil {
			log.Fatalf("No se pudo conectar a PostgreSQL: %v", err)
		}
	}
	log.Printf("PostgreSQL conectado en %s:%s", cfg.DBHost, cfg.DBPort)
	return baseDatos
}

func construirRouter(h *handler.Handler, cacheSrv *cache.CacheService, limitador *middleware.IPRateLimiter) http.Handler {
	mux := http.NewServeMux()

	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/svg+xml")
		http.ServeFile(w, r, "web/static/favicon.svg")
	})

	mux.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "web/register.html")
	})
	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "web/login.html")
	})
	mux.HandleFunc("/swagger", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "web/swagger.html")
	})
	mux.HandleFunc("/swagger.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		http.ServeFile(w, r, "web/swagger.json")
	})

	filtroAuthRateLimit := limitador.RateLimit(10, 1)
	mux.Handle("/api/auth/register", filtroAuthRateLimit(http.HandlerFunc(h.HandleRegister)))
	mux.Handle("/api/auth/login", filtroAuthRateLimit(http.HandlerFunc(h.HandleLogin)))

	middlewareAuth := middleware.AuthMiddleware(cacheSrv)
	protected := http.NewServeMux()

	protected.HandleFunc("/api/auth/logout", h.HandleLogout)
	protected.HandleFunc("/api/auth/me", h.HandleMe)
	protected.HandleFunc("/api/admin/sessions", h.HandleGetActiveSessions)
	protected.HandleFunc("/api/admin/sessions/revoke", h.HandleRevokeUserSession)
	protected.HandleFunc("/api/stream/channels", h.HandleListChannels)
	protected.HandleFunc("/api/stream/segments/", h.HandleSegment)
	protected.HandleFunc("/api/stream/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "master.m3u8") {
			h.HandleMasterPlaylist(w, r)
			return
		}
		h.HandleLivePlaylist(w, r)
	})
	protected.HandleFunc("/master.m3u8", h.HandleMasterPlaylist)
	protected.HandleFunc("/live.m3u8", h.HandleLivePlaylist)
	protected.HandleFunc("/stream/live.m3u8", h.HandleLivePlaylist)
	protected.HandleFunc("/api/metrics", h.HandleMetrics)
	protected.HandleFunc("/api/ws", func(w http.ResponseWriter, r *http.Request) {
		ws.ServeWS(h.WSHub(), w, r)
	})
	protected.HandleFunc("/player", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "web/player.html")
	})

	rutasProtegidas := []string{
		"/api/auth/logout", "/api/auth/me", "/api/admin/sessions",
		"/api/admin/sessions/revoke", "/api/stream/channels",
		"/api/stream/segments/", "/api/stream/", "/master.m3u8",
		"/live.m3u8", "/stream/", "/api/metrics", "/api/ws", "/player",
	}

	for _, ruta := range rutasProtegidas {
		mux.Handle(ruta, middlewareAuth(protected))
	}

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".ts") {
			middlewareAuth(http.HandlerFunc(h.HandleSegment)).ServeHTTP(w, r)
			return
		}
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, "/player", http.StatusSeeOther)
	})

	return mux
}

func apagarConGracefulShutdown(servidor *http.Server, puerto string) {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("Servidor escuchando en http://localhost:%s", puerto)
		if err := servidor.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Error HTTP: %v", err)
		}
	}()

	<-stop
	log.Println("Apagando servidor Kuspid de forma segura...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := servidor.Shutdown(ctx); err != nil {
		log.Fatalf("Error en apagado forzado: %v", err)
	}

	log.Println("Servidor apagado exitosamente.")
}

func obtenerEnv(clave, valorPorDefecto string) string {
	if val := os.Getenv(clave); val != "" {
		return val
	}
	return valorPorDefecto
}
