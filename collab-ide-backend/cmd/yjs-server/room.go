package main

import (
	"sync"

	"go.uber.org/zap"
)

type Room struct {
	ID         string
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte
	mu         sync.RWMutex
	logger     *zap.Logger
	stop       chan struct{}
}

func NewRoom(id string, logger *zap.Logger) *Room {
	return &Room{
		ID:         id,
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte, 256),
		logger:     logger,
		stop:       make(chan struct{}),
	}
}

func (r *Room) Run() {
	for {
		select {
		case client := <-r.register:
			r.mu.Lock()
			r.clients[client] = true
			r.mu.Unlock()
			r.logger.Debug("Client joined",
				zap.String("room", r.ID),
				zap.String("user", client.userID),
				zap.Int("total", len(r.clients)),
			)

		case client := <-r.unregister:
			r.mu.Lock()
			if _, ok := r.clients[client]; ok {
				delete(r.clients, client)
				close(client.send)
				r.logger.Debug("Client left",
					zap.String("room", r.ID),
					zap.String("user", client.userID),
					zap.Int("total", len(r.clients)),
				)
			}
			r.mu.Unlock()

		case message := <-r.broadcast:
			r.mu.RLock()
			for client := range r.clients {
				select {
				case client.send <- message:
				default:
				}
			}
			r.mu.RUnlock()

		case <-r.stop:
			return
		}
	}
}

func (r *Room) Stop() {
	close(r.stop)
}

func (r *Room) Broadcast(data []byte) {
	select {
	case r.broadcast <- data:
	default:
	}
}

func (r *Room) GetClientCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.clients)
}
