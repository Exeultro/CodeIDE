package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"

	"collab-ide-backend/internal/api/rest"
	myws "collab-ide-backend/internal/api/websocket"
	"collab-ide-backend/internal/auth"
	"collab-ide-backend/internal/config"
	"collab-ide-backend/internal/core/ai"
	"collab-ide-backend/internal/middleware"
	"collab-ide-backend/internal/repository"
	"collab-ide-backend/internal/telegram"
)

func main() {
	// Загружаем конфиг
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	// Инициализация JWT
	auth.InitJWT(cfg.JWTSecret)

	// Подключение к БД
	db, err := repository.NewPostgres(cfg)
	if err != nil {
		log.Fatal("Postgres connection failed:", err)
	}
	defer db.Pool.Close()

	// Миграции
	if err := repository.Migrate(db); err != nil {
		log.Fatal("Migration failed:", err)
	}
	log.Println("✅ Database migrated successfully")

	// Инициализация репозиториев
	snapRepo := repository.NewSnapshotRepo(db)
	userRepo := repository.NewUserRepo(db)
	fileRepo := repository.NewFileRepo(db)
	scoreRepo := repository.NewScoreRepo(db)

	// AI клиент
	aiClient := ai.NewOllamaClient(cfg.OllamaURL, cfg.OllamaModel)
	log.Printf("✅ AI Client initialized (model: %s)", cfg.OllamaModel)

	// Получаем Redis клиент
	redisRepo, err := repository.NewRedis(cfg)
	var redisClient *redis.Client
	if err == nil && redisRepo != nil {
		redisClient = redisRepo.Client
		log.Println("✅ Redis connected")
	} else {
		log.Println("⚠️ Redis warning:", err)
	}

	// WebSocket hub
	hub := myws.NewHub(snapRepo, scoreRepo, aiClient, db)
	go hub.Run()

	// Создаём sessionHandler с Redis
	sessionHandler := rest.NewSessionHandler(db, fileRepo, snapRepo, scoreRepo, hub, aiClient, redisClient)
	authHandler := rest.NewAuthHandler(userRepo)

	// Telegram бот
	var tgBot *telegram.Bot
	if cfg.TelegramToken != "" {
		// Передаём Redis клиент в handlers
		handlers := telegram.NewHandlers(db, redisClient, snapRepo, scoreRepo, aiClient)
		tgBot = telegram.NewBot(cfg.TelegramToken, cfg.TelegramChatID, handlers)

		go func() {
			ctx := context.Background()
			if err := tgBot.Start(ctx); err != nil {
				log.Printf("⚠️ Telegram bot error: %v", err)
			}
		}()
		log.Println("✅ Telegram bot initialized")
	} else {
		log.Println("⚠️ Telegram bot disabled (no token)")
	}

	// Rate Limiter (100 запросов в минуту)
	rateLimiter := middleware.NewRateLimiter(100, time.Minute)

	// WebSocket upgrader с CORS
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	// Настройка роутера
	mux := http.NewServeMux()

	// Публичные эндпоинты
	mux.HandleFunc("/healthz", middleware.EnableCORS(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(rest.Ok(map[string]any{"status": "ok", "timestamp": time.Now()}))
	}))
	mux.HandleFunc("/api/health", middleware.EnableCORS(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"ok": true, "services": checkServices(db)})
	}))

	// WebSocket (без rate limiting для WebSocket)
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		handleWebSocket(w, r, upgrader, hub)
	})

	// Auth endpoints
	mux.HandleFunc("/api/auth/register", middleware.RateLimit(rateLimiter, middleware.EnableCORS(authHandler.Register)))
	mux.HandleFunc("/api/auth/login", middleware.RateLimit(rateLimiter, middleware.EnableCORS(authHandler.Login)))

	// Session endpoints
	mux.HandleFunc("/api/sessions", handleWithCORS(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			middleware.RateLimit(rateLimiter, middleware.JWTAuth(sessionHandler.CreateSession)).ServeHTTP(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}))

	mux.HandleFunc("/api/user/sessions", handleWithCORS(func(w http.ResponseWriter, r *http.Request) {
		middleware.RateLimit(rateLimiter, middleware.JWTAuth(sessionHandler.GetUserSessions)).ServeHTTP(w, r)
	}))

	mux.HandleFunc("/api/sessions/", handleWithCORS(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		if strings.Contains(path, "/files") {
			middleware.RateLimit(rateLimiter, middleware.JWTAuth(sessionHandler.FileRouter)).ServeHTTP(w, r)
		} else if strings.HasSuffix(path, "/content") {
			if r.Method == http.MethodGet {
				middleware.RateLimit(rateLimiter, sessionHandler.GetContent).ServeHTTP(w, r)
			} else if r.Method == http.MethodPut {
				middleware.RateLimit(rateLimiter, middleware.JWTAuth(sessionHandler.UpdateContent)).ServeHTTP(w, r)
			}
		} else if strings.HasSuffix(path, "/events") {
			middleware.RateLimit(rateLimiter, sessionHandler.GetEvents).ServeHTTP(w, r)
		} else if strings.HasSuffix(path, "/ai-reviews") && !strings.Contains(path, "/apply") {
			middleware.RateLimit(rateLimiter, sessionHandler.GetAIReviews).ServeHTTP(w, r)
		} else if strings.HasSuffix(path, "/participants") && !strings.Contains(path, "/participants/") {
			middleware.RateLimit(rateLimiter, sessionHandler.GetParticipants).ServeHTTP(w, r)
		} else if strings.HasSuffix(path, "/invite") {
			middleware.RateLimit(rateLimiter, middleware.JWTAuth(sessionHandler.InviteUser)).ServeHTTP(w, r)
		} else if strings.Contains(path, "/participants/") && r.Method == http.MethodDelete {
			middleware.RateLimit(rateLimiter, middleware.JWTAuth(sessionHandler.RemoveParticipant)).ServeHTTP(w, r)
		} else if strings.HasSuffix(path, "/apply") && strings.Contains(path, "/ai-reviews/") {
			middleware.RateLimit(rateLimiter, middleware.JWTAuth(sessionHandler.ApplyAIReview)).ServeHTTP(w, r)
		} else if strings.HasSuffix(path, "/history") {
			middleware.RateLimit(rateLimiter, sessionHandler.GetHistory).ServeHTTP(w, r)
		} else if strings.HasSuffix(path, "/invite-link") {
			middleware.RateLimit(rateLimiter, middleware.JWTAuth(sessionHandler.GetInviteLink)).ServeHTTP(w, r)
		} else if strings.HasSuffix(path, "/join-by-invite") {
			middleware.RateLimit(rateLimiter, middleware.JWTAuth(sessionHandler.JoinByInvite)).ServeHTTP(w, r)
		} else if strings.HasSuffix(path, "/leaderboard") {
			middleware.RateLimit(rateLimiter, sessionHandler.GetLeaderboard).ServeHTTP(w, r)
		} else if strings.HasSuffix(path, "/hint") {
			middleware.RateLimit(rateLimiter, middleware.JWTAuth(sessionHandler.GetHint)).ServeHTTP(w, r)
		} else if strings.HasSuffix(path, "/profile") {
			if r.Method == http.MethodGet {
				middleware.RateLimit(rateLimiter, middleware.JWTAuth(sessionHandler.GetProfile)).ServeHTTP(w, r)
			} else if r.Method == http.MethodPut {
				middleware.RateLimit(rateLimiter, middleware.JWTAuth(sessionHandler.UpdateProfile)).ServeHTTP(w, r)
			}
		} else if strings.HasSuffix(path, "/restore") {
			middleware.RateLimit(rateLimiter, middleware.JWTAuth(sessionHandler.RestoreVersion)).ServeHTTP(w, r)
		} else {
			if r.Method == http.MethodGet {
				middleware.RateLimit(rateLimiter, sessionHandler.GetSession).ServeHTTP(w, r)
			} else {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		}
	}))

	// Настройка сервера
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		log.Printf("🚀 Server started on :%s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Server failed:", err)
		}
	}()

	// Ожидание сигнала завершения
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("🛑 Shutting down server...")

	// Останавливаем Telegram бота
	if tgBot != nil {
		tgBot.Stop()
		log.Println("🤖 Telegram bot stopped")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("✅ Server exited gracefully")
}

// Вспомогательная функция для обработки CORS
func handleWithCORS(handler http.HandlerFunc) http.HandlerFunc {
	return middleware.EnableCORS(func(w http.ResponseWriter, r *http.Request) {
		handler(w, r)
	})
}

// Обработка WebSocket соединения
func handleWebSocket(w http.ResponseWriter, r *http.Request, upgrader websocket.Upgrader, hub *myws.Hub) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}

	roomID := r.URL.Query().Get("room")
	userID := r.URL.Query().Get("user")
	username := r.URL.Query().Get("username")

	if roomID == "" || userID == "" {
		log.Println("Missing room or user ID")
		conn.Close()
		return
	}

	client := &myws.Client{
		Hub:      hub,
		Conn:     conn,
		Send:     make(chan []byte, 256),
		SendBin:  make(chan []byte, 256),
		RoomID:   roomID,
		UserID:   userID,
		Username: username,
	}

	room := hub.GetOrCreateRoom(roomID)
	if room == nil {
		conn.Close()
		return
	}

	room.Register <- client
	go client.WritePump()
	go client.ReadPump()
}

// Проверка состояния сервисов
func checkServices(db *repository.PostgresRepo) map[string]any {
	status := make(map[string]any)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := db.Pool.Ping(ctx); err != nil {
		status["postgres"] = map[string]any{"status": "down", "error": err.Error()}
	} else {
		status["postgres"] = map[string]any{"status": "up"}
	}

	return status
}
