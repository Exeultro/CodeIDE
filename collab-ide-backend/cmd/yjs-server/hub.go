package main

import (
	"sync"

	"go.uber.org/zap"
)

type Hub struct {
	rooms      map[string]*Room
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte
	mu         sync.RWMutex
	logger     *zap.Logger
	stop       chan struct{}
}

func NewHub(logger *zap.Logger) *Hub {
	return &Hub{
		rooms:      make(map[string]*Room),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte, 256),
		logger:     logger,
		stop:       make(chan struct{}),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.handleRegister(client)

		case client := <-h.unregister:
			h.handleUnregister(client)

		case message := <-h.broadcast:
			h.handleBroadcast(message)

		case <-h.stop:
			h.logger.Info("Hub stopped")
			return
		}
	}
}

func (h *Hub) Stop() {
	close(h.stop)
}

func (h *Hub) handleRegister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	room, exists := h.rooms[client.roomID]
	if !exists {
		room = NewRoom(client.roomID, h.logger)
		h.rooms[client.roomID] = room
		go room.Run()
		h.logger.Info("Room created", zap.String("room", client.roomID))
	}

	room.register <- client
}

func (h *Hub) handleUnregister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	room, exists := h.rooms[client.roomID]
	if exists {
		room.unregister <- client

		if len(room.clients) == 0 {
			delete(h.rooms, client.roomID)
			h.logger.Info("Room removed", zap.String("room", client.roomID))
		}
	}
}

func (h *Hub) handleBroadcast(message []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, room := range h.rooms {
		select {
		case room.broadcast <- message:
		default:
		}
	}
}

func (h *Hub) GetRoom(roomID string) *Room {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.rooms[roomID]
}
