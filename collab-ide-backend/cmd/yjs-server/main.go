package main

import (
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	addr = flag.String("addr", ":1234", "WebSocket server address")
)

func main() {
	flag.Parse()

	// Настройка логгера
	logger := newLogger()
	defer logger.Sync()

	logger.Info("Starting Yjs WebSocket server",
		zap.String("addr", *addr),
	)

	// WebSocket upgrader
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Разрешаем все origins для разработки
		},
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	// Создаем хаб для управления комнатами
	hub := NewHub(logger)
	go hub.Run()

	// Обработчик WebSocket
	http.HandleFunc("/yjs", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			logger.Error("WebSocket upgrade failed", zap.Error(err))
			return
		}

		roomID := r.URL.Query().Get("room")
		if roomID == "" {
			logger.Warn("No room ID provided")
			conn.Close()
			return
		}

		client := &Client{
			hub:    hub,
			conn:   conn,
			send:   make(chan []byte, 256),
			roomID: roomID,
			userID: r.URL.Query().Get("user"),
		}

		client.hub.register <- client

		go client.writePump()
		go client.readPump()
	})

	// Health check
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Запуск сервера
	server := &http.Server{
		Addr: *addr,
	}

	go func() {
		logger.Info("Server listening", zap.String("addr", server.Addr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server failed", zap.Error(err))
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")
	hub.Stop()
	server.Close()
}

func newLogger() *zap.Logger {
	config := zap.NewDevelopmentConfig()
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	logger, _ := config.Build()
	return logger
}
