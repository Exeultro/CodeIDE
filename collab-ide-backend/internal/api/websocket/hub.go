package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"collab-ide-backend/internal/core/ai"
	"collab-ide-backend/internal/core/sandbox"
	"collab-ide-backend/internal/repository"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Hub struct {
	rooms     map[string]*Room
	mu        sync.RWMutex
	snapRepo  *repository.SnapshotRepo
	scoreRepo *repository.ScoreRepo
	aiClient  *ai.OllamaClient
	db        *repository.PostgresRepo
}

type Room struct {
	ID            string
	Clients       map[*Client]bool
	Register      chan *Client
	Unregister    chan *Client
	BroadcastText chan []byte
	BroadcastBin  chan []byte

	saveText    chan saveRequest
	sessionID   uuid.UUID
	snapRepo    *repository.SnapshotRepo
	scoreRepo   *repository.ScoreRepo
	lastText    string
	lastVersion int64
	mu          sync.RWMutex
	containerID string
	sandbox     *sandbox.Sandbox
	sandboxMu   sync.Mutex
	aiClient    *ai.OllamaClient
	language    string
}

type saveRequest struct {
	Content  string
	UserID   string
	Username string
}

func NewHub(snapRepo *repository.SnapshotRepo, scoreRepo *repository.ScoreRepo, aiClient *ai.OllamaClient, db *repository.PostgresRepo) *Hub {
	return &Hub{
		rooms:     make(map[string]*Room),
		snapRepo:  snapRepo,
		scoreRepo: scoreRepo,
		aiClient:  aiClient,
		db:        db,
	}
}

func (h *Hub) GetOrCreateRoom(roomID string) *Room {
	h.mu.Lock()
	defer h.mu.Unlock()
	if room, ok := h.rooms[roomID]; ok {
		return room
	}
	sessionUUID, err := uuid.Parse(roomID)
	if err != nil {
		log.Printf("invalid room id: %s", roomID)
		return nil
	}
	lastText, version, err := h.snapRepo.LoadLatest(sessionUUID)
	if err != nil {
		log.Printf("load snapshot: %v", err)
	}

	// 👇 ДОБАВЛЯЕМ ПОЛУЧЕНИЕ ЯЗЫКА ИЗ СЕССИИ
	var language string
	err = h.db.Pool.QueryRow(context.Background(),
		`SELECT language FROM sessions WHERE id = $1`, sessionUUID).Scan(&language)
	if err != nil {
		language = "python"
		log.Printf("failed to get language for session %s: %v, using default", sessionUUID, err)
	}

	room := &Room{
		ID: roomID, Clients: map[*Client]bool{}, Register: make(chan *Client),
		Unregister: make(chan *Client), BroadcastText: make(chan []byte, 256),
		BroadcastBin: make(chan []byte, 256), saveText: make(chan saveRequest, 64),
		sessionID: sessionUUID, snapRepo: h.snapRepo, scoreRepo: h.scoreRepo,
		lastText: lastText, lastVersion: version, aiClient: h.aiClient,
		language: language, // 👈 ДОБАВЛЯЕМ
	}
	go room.run()
	h.rooms[roomID] = room
	return room
}

func (r *Room) hasConflict(oldText, newText string) bool {
	// Простая проверка на конфликт (изменение одной строки)
	oldLines := strings.Split(oldText, "\n")
	newLines := strings.Split(newText, "\n")
	if len(oldLines) != len(newLines) {
		return false
	}
	for i := range oldLines {
		if oldLines[i] != newLines[i] && oldLines[i] != "" && newLines[i] != "" {
			return true
		}
	}
	return false
}

func (r *Room) broadcastMerged(content string, version int64) {
	data, _ := json.Marshal(map[string]any{
		"type": "merged_state",
		"payload": map[string]any{
			"content": content,
			"version": version,
		},
	})
	for client := range r.Clients {
		select {
		case client.Send <- data:
		default:
		}
	}
}

func (r *Room) run() {
	for {
		select {
		case client := <-r.Register:
			r.Clients[client] = true
			r.addEvent(client.UserID, "join", map[string]any{"username": client.Username})
			r.sendFullState(client)
			client.broadcastParticipants()
		case client := <-r.Unregister:
			if _, ok := r.Clients[client]; ok {
				delete(r.Clients, client)
				r.addEvent(client.UserID, "leave", map[string]any{"username": client.Username})
				close(client.Send)
				close(client.SendBin)
				if len(r.Clients) > 0 {
					for cl := range r.Clients {
						cl.broadcastParticipants()
						break
					}
				}
			}
		// В комнате, при получении двух конфликтующих сохранений
		case req := <-r.saveText:
			r.mu.Lock()
			oldText := r.lastText
			r.lastText = req.Content
			newVersion := r.lastVersion + 1
			r.lastVersion = newVersion
			r.mu.Unlock()

			// Проверка на конфликт (если изменения в одной строке)
			if r.hasConflict(oldText, req.Content) && r.aiClient != nil {
				merged, err := r.aiClient.MergeConflicts(oldText, req.Content)
				if err == nil && merged != "" {
					r.lastText = merged
					go r.persistText(req, oldText, newVersion)
					// Отправляем объединённую версию всем
					r.broadcastMerged(merged, newVersion)
					return
				}
			}
			go r.persistText(req, oldText, newVersion)
		case msg := <-r.BroadcastText:
			for client := range r.Clients {
				select {
				case client.Send <- msg:
				default:
					delete(r.Clients, client)
					close(client.Send)
					close(client.SendBin)
				}
			}
		case msg := <-r.BroadcastBin:
			for client := range r.Clients {
				select {
				case client.SendBin <- msg:
				default:
					delete(r.Clients, client)
					close(client.Send)
					close(client.SendBin)
				}
			}
		}
	}
}

func (r *Room) sendFullState(client *Client) {
	r.mu.RLock()
	text, version := r.lastText, r.lastVersion
	r.mu.RUnlock()
	msg := map[string]any{"type": "full_state", "payload": map[string]any{"content": text, "version": version}}
	data, _ := json.Marshal(msg)
	select {
	case client.Send <- data:
	default:
	}
}

func (r *Room) persistText(req saveRequest, oldText string, version int64) {
	if err := r.snapRepo.Save(r.sessionID, req.Content, version); err != nil {
		log.Printf("save snapshot: %v", err)
		return
	}
	r.addEvent(req.UserID, "save", map[string]any{"version": version, "length": len(req.Content), "username": req.Username})
	if uid, err := uuid.Parse(req.UserID); err == nil && r.scoreRepo != nil {
		_ = r.scoreRepo.AddPoints(context.Background(), r.sessionID, uid, req.Username, 5)
	}
	if req.Content != oldText && len(strings.TrimSpace(req.Content)) > 10 {
		go r.runAIAnalysis(req.Content)
	}
}

func (r *Room) addEvent(userIDStr, eventType string, details map[string]any) {
	userUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		userUUID = uuid.Nil
	}
	if err := r.snapRepo.AddEvent(r.sessionID, userUUID, eventType, details); err != nil {
		log.Printf("add event: %v", err)
	}
}

func (r *Room) SetState(content string, version int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lastText = content
	r.lastVersion = version
}

func (r *Room) SaveText(content, userID, username string) {
	select {
	case r.saveText <- saveRequest{Content: content, UserID: userID, Username: username}:
	default:
	}
}

func (r *Room) ensureContainer() error {
	r.sandboxMu.Lock()
	defer r.sandboxMu.Unlock()
	if r.sandbox != nil && r.containerID != "" {
		return nil
	}
	sb, err := sandbox.NewSandbox()
	if err != nil {
		return err
	}
	r.sandbox = sb
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	id, err := sb.CreateContainer(ctx, "session-"+r.ID)
	if err != nil {
		return err
	}
	r.containerID = id
	return nil
}

func (r *Room) RunCode() (string, error) {
	if err := r.ensureContainer(); err != nil {
		return "", fmt.Errorf("failed to create container: %v", err)
	}

	r.mu.RLock()
	code := r.lastText
	lang := r.language
	r.mu.RUnlock()

	if strings.TrimSpace(code) == "" {
		return "", nil
	}

	if lang == "" {
		lang = "python"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return r.sandbox.RunCode(ctx, r.containerID, code, lang)
}

// Добавляем функцию определения языка
func (r *Room) detectLanguage() string {
	// Можно получать из session или определять по расширению
	return "python" // По умолчанию Python
}

func (r *Room) Exec(command string) (string, error) {
	if err := r.ensureContainer(); err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return r.sandbox.Exec(ctx, r.containerID, command)
}

func (r *Room) runAIAnalysis(newText string) {
	if r.aiClient == nil {
		return
	}
	review, err := r.aiClient.ReviewCode(newText)
	if err != nil || strings.TrimSpace(review) == "" {
		if err != nil {
			log.Printf("ai review: %v", err)
		}
		return
	}
	msg := map[string]any{"type": "ai_suggestion", "payload": map[string]any{"message": review, "created_at": time.Now()}}
	data, _ := json.Marshal(msg)
	select {
	case r.BroadcastText <- data:
	default:
	}
	_ = r.snapRepo.SaveAIReview(r.sessionID, review, newText)
}

// Run запускает hub в горутине
func (h *Hub) Run() {
	// Можно добавить очистку неактивных комнат
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			h.cleanupInactiveRooms()
		}
	}
}

// cleanupInactiveRooms удаляет пустые комнаты
func (h *Hub) cleanupInactiveRooms() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for id, room := range h.rooms {
		if len(room.Clients) == 0 {
			delete(h.rooms, id)
			log.Printf("🧹 Removed inactive room: %s", id)
		}
	}
}
func (r *Room) SetLanguage(lang string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.language = lang
}

func (r *Room) GetLanguage() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.language
}

func (h *Hub) Stop() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, room := range h.rooms {
		for client := range room.Clients {
			client.Conn.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, "server shutdown"))
			client.Conn.Close()
		}
	}
}
