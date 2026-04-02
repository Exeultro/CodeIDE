package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	myws "collab-ide-backend/internal/api/websocket"
	"collab-ide-backend/internal/core/ai"
	"collab-ide-backend/internal/models"
	"collab-ide-backend/internal/repository"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type SessionHandler struct {
	DB        *repository.PostgresRepo
	FileRepo  *repository.FileRepo
	SnapRepo  *repository.SnapshotRepo
	ScoreRepo *repository.ScoreRepo
	Hub       *myws.Hub
	AI        *ai.OllamaClient
	Redis     *redis.Client
}

func NewSessionHandler(db *repository.PostgresRepo, fileRepo *repository.FileRepo, snapRepo *repository.SnapshotRepo, scoreRepo *repository.ScoreRepo, hub *myws.Hub, aiClient *ai.OllamaClient, redisClient *redis.Client) *SessionHandler {
	return &SessionHandler{
		DB:        db,
		FileRepo:  fileRepo,
		SnapRepo:  snapRepo,
		ScoreRepo: scoreRepo,
		Hub:       hub,
		AI:        aiClient,
		Redis:     redisClient,
	}
}

func (h *SessionHandler) CreateSession(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(Fail(401, "unauthorized"))
		return
	}
	var req struct {
		Name     string `json:"name"`
		FileName string `json:"file_name"`
		Language string `json:"language"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Fail(400, "invalid request"))
		return
	}
	if req.Name == "" {
		req.Name = "Новая сессия"
	}
	if req.FileName == "" {
		req.FileName = "main.py"
	}
	if req.Language == "" {
		req.Language = "python"
	}
	session := models.Session{
		ID:        uuid.New(),
		Name:      req.Name,
		FileName:  req.FileName,
		Language:  req.Language,
		OwnerID:   uuid.MustParse(user.UserID),
		CreatedAt: time.Now(),
		Active:    true,
		Content:   "print('Hello, team!')\n",
		Version:   1,
	}

	_, err := h.DB.Pool.Exec(r.Context(),
		`INSERT INTO sessions (id,name,file_name,language,owner_id,created_at,active,content,version) 
         VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		session.ID, session.Name, session.FileName, session.Language, session.OwnerID,
		session.CreatedAt, session.Active, session.Content, session.Version)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Fail(500, "database error"))
		return
	}

	// ДОБАВЛЯЕМ ВЛАДЕЛЬЦА В УЧАСТНИКИ
	_, err = h.DB.Pool.Exec(r.Context(),
		`INSERT INTO session_participants (session_id, user_id, joined_at)
         VALUES ($1, $2, NOW())`,
		session.ID, session.OwnerID)
	if err != nil {
		// Логируем, но не прерываем создание сессии
		log.Printf("Failed to add owner to participants: %v", err)
	}

	_ = h.SnapRepo.Save(session.ID, session.Content, session.Version)
	_, _ = h.FileRepo.CreateFile(r.Context(), session.ID, req.FileName, false, &session.Content)
	_ = h.ScoreRepo.SetProfile(r.Context(), session.ID, session.OwnerID, user.Username, false, "")

	json.NewEncoder(w).Encode(Ok(session))
}

func (h *SessionHandler) GetSession(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/sessions/")

	// Проверяем другие эндпоинты
	switch {
	case strings.HasSuffix(path, "/content"):
		h.GetContent(w, r)
		return
	case strings.HasSuffix(path, "/events"):
		h.GetEvents(w, r)
		return
	case strings.HasSuffix(path, "/ai-reviews"):
		h.GetAIReviews(w, r)
		return
	case strings.HasSuffix(path, "/history"):
		h.GetHistory(w, r)
		return
	case strings.HasSuffix(path, "/leaderboard"):
		h.GetLeaderboard(w, r)
		return
	case strings.HasSuffix(path, "/hint"):
		h.GetHint(w, r)
		return
	case strings.HasSuffix(path, "/restore"):
		h.RestoreVersion(w, r)
		return
	}

	idStr := strings.Split(path, "/")[0]
	sessionID, err := uuid.Parse(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Fail(400, "invalid session id"))
		return
	}

	// Пробуем получить из кэша
	if cachedSession, err := h.getSessionFromCache(sessionID); err == nil && cachedSession != nil {
		json.NewEncoder(w).Encode(Ok(cachedSession))
		return
	}

	// Если нет в кэше, грузим из БД
	var sess models.Session
	err = h.DB.Pool.QueryRow(r.Context(),
		`SELECT id, name, file_name, language, owner_id, created_at, active, content, version 
         FROM sessions WHERE id=$1`, sessionID).Scan(
		&sess.ID, &sess.Name, &sess.FileName, &sess.Language,
		&sess.OwnerID, &sess.CreatedAt, &sess.Active, &sess.Content, &sess.Version)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(Fail(404, "session not found"))
		return
	}

	// Сохраняем в кэш
	h.saveSessionToCache(&sess)

	json.NewEncoder(w).Encode(Ok(sess))
}

// getSessionFromCache получает сессию из Redis кэша
func (h *SessionHandler) getSessionFromCache(sessionID uuid.UUID) (*models.Session, error) {
	if h.Redis == nil {
		return nil, fmt.Errorf("redis not available")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	key := "session:" + sessionID.String()
	data, err := h.Redis.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("cache miss")
		}
		return nil, err
	}

	var session models.Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, err
	}

	log.Printf("[Cache] Session %s retrieved from Redis", sessionID.String()[:8])
	return &session, nil
}

// saveSessionToCache сохраняет сессию в Redis кэш на 5 минут
func (h *SessionHandler) saveSessionToCache(session *models.Session) {
	if h.Redis == nil {
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		key := "session:" + session.ID.String()
		data, err := json.Marshal(session)
		if err != nil {
			log.Printf("[Cache] Failed to marshal session: %v", err)
			return
		}

		if err := h.Redis.Set(ctx, key, data, 5*time.Minute).Err(); err != nil {
			log.Printf("[Cache] Failed to save session to Redis: %v", err)
			return
		}

		log.Printf("[Cache] Session %s saved to Redis (TTL: 5min)", session.ID.String()[:8])
	}()
}

// deleteSessionFromCache удаляет сессию из кэша (при обновлении)
func (h *SessionHandler) deleteSessionFromCache(sessionID uuid.UUID) {
	if h.Redis == nil {
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		key := "session:" + sessionID.String()
		if err := h.Redis.Del(ctx, key).Err(); err != nil {
			log.Printf("[Cache] Failed to delete session from Redis: %v", err)
			return
		}

		log.Printf("[Cache] Session %s deleted from Redis", sessionID.String()[:8])
	}()
}

func (h *SessionHandler) GetContent(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/sessions/"), "/content")
	sessionID, err := uuid.Parse(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Fail(400, "invalid session id"))
		return
	}
	content, version, err := h.SnapRepo.LoadLatest(sessionID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Fail(500, "database error"))
		return
	}
	json.NewEncoder(w).Encode(Ok(map[string]any{"content": content, "version": version}))
}

func (h *SessionHandler) UpdateContent(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	idStr := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/sessions/"), "/content")
	sessionID, err := uuid.Parse(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Fail(400, "invalid session id"))
		return
	}
	var req struct {
		Content     string `json:"content"`
		BaseVersion int64  `json:"base_version"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Fail(400, "invalid request"))
		return
	}
	oldContent, latestVersion, err := h.SnapRepo.LoadLatest(sessionID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Fail(500, "database error"))
		return
	}
	if req.BaseVersion != latestVersion {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(Fail(409, "conflict: newer version exists"))
		return
	}
	newVersion := latestVersion + 1
	if err := h.SnapRepo.Save(sessionID, req.Content, newVersion); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Fail(500, "database error"))
		return
	}
	if _, err := h.DB.Pool.Exec(r.Context(), `UPDATE sessions SET content=$1, version=$2 WHERE id=$3`, req.Content, newVersion, sessionID); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Fail(500, "database error"))
		return
	}

	userID := uuid.Nil
	username := "system"
	if user != nil {
		if parsed, err := uuid.Parse(user.UserID); err == nil {
			userID = parsed
		}
		if strings.TrimSpace(user.Username) != "" {
			username = user.Username
		}
	}

	_ = h.SnapRepo.AddEvent(sessionID, userID, "save", map[string]any{
		"version":  newVersion,
		"length":   len(req.Content),
		"username": username,
		"source":   "rest",
	})

	if room := h.Hub.GetOrCreateRoom(sessionID.String()); room != nil {
		room.SetState(req.Content, newVersion)
		data, _ := json.Marshal(map[string]any{
			"type": "full_state",
			"payload": map[string]any{
				"content": req.Content,
				"version": newVersion,
			},
		})
		select {
		case room.BroadcastText <- data:
		default:
		}
	}

	if h.AI != nil && req.Content != oldContent && len(strings.TrimSpace(req.Content)) > 10 {
		go func(sessionID uuid.UUID, content string) {
			review, err := h.AI.ReviewCode(content)
			if err != nil || strings.TrimSpace(review) == "" {
				return
			}
			_ = h.SnapRepo.SaveAIReview(sessionID, review, content)
		}(sessionID, req.Content)
	}

	json.NewEncoder(w).Encode(Ok(map[string]any{"version": newVersion}))
}

func (h *SessionHandler) GetUserSessions(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(Fail(401, "unauthorized"))
		return
	}

	userUUID := uuid.MustParse(user.UserID)

	// Получаем сессии, где пользователь является ВЛАДЕЛЬЦЕМ или УЧАСТНИКОМ
	rows, err := h.DB.Pool.Query(r.Context(),
		`SELECT DISTINCT s.id, s.name, s.file_name, s.language, s.created_at, s.active, s.owner_id, s.content, s.version
         FROM sessions s
         LEFT JOIN session_participants sp ON sp.session_id = s.id
         WHERE s.owner_id = $1 OR sp.user_id = $1
         ORDER BY s.created_at DESC`,
		userUUID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Fail(500, "database error"))
		return
	}
	defer rows.Close()

	list := []models.Session{}
	for rows.Next() {
		var s models.Session
		if err := rows.Scan(&s.ID, &s.Name, &s.FileName, &s.Language, &s.CreatedAt, &s.Active, &s.OwnerID, &s.Content, &s.Version); err == nil {
			list = append(list, s)
		}
	}
	json.NewEncoder(w).Encode(Ok(list))
}

func (h *SessionHandler) GetEvents(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/sessions/"), "/events")
	sessionID, err := uuid.Parse(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Fail(400, "invalid session id"))
		return
	}

	// Пагинация
	limit := 50
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	rows, err := h.DB.Pool.Query(r.Context(),
		`SELECT id, COALESCE(user_id,'00000000-0000-0000-0000-000000000000'::uuid), 
                event_type, details, created_at 
         FROM session_events 
         WHERE session_id=$1 
         ORDER BY created_at DESC 
         LIMIT $2 OFFSET $3`,
		sessionID, limit, offset)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Fail(500, "database error"))
		return
	}
	defer rows.Close()

	events := make([]map[string]any, 0)
	for rows.Next() {
		var id, userID uuid.UUID
		var eventType string
		var details []byte
		var createdAt time.Time
		if err := rows.Scan(&id, &userID, &eventType, &details, &createdAt); err != nil {
			continue
		}
		var m map[string]any
		_ = json.Unmarshal(details, &m)
		events = append(events, map[string]any{
			"id": id, "user_id": userID,
			"event_type": eventType,
			"details":    m, "created_at": createdAt,
		})
	}

	// Добавляем информацию о пагинации
	json.NewEncoder(w).Encode(Ok(map[string]any{
		"events": events,
		"pagination": map[string]int{
			"limit":  limit,
			"offset": offset,
		},
	}))
}

func (h *SessionHandler) GetAIReviews(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/sessions/"), "/ai-reviews")
	sessionID, err := uuid.Parse(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Fail(400, "invalid session id"))
		return
	}
	rows, err := h.DB.Pool.Query(r.Context(), `SELECT id,type,start_line,end_line,original_snippet,suggested_snippet,message,resolved,created_at FROM ai_reviews WHERE session_id=$1 ORDER BY created_at DESC LIMIT 50`, sessionID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Fail(500, "database error"))
		return
	}
	defer rows.Close()
	reviews := make([]map[string]any, 0)
	for rows.Next() {
		var id uuid.UUID
		var typ, original, suggested, message string
		var startLine, endLine int
		var resolved bool
		var createdAt time.Time
		if err := rows.Scan(&id, &typ, &startLine, &endLine, &original, &suggested, &message, &resolved, &createdAt); err != nil {
			continue
		}
		reviews = append(reviews, map[string]any{"id": id, "type": typ, "location": map[string]int{"start_line": startLine, "end_line": endLine}, "original_snippet": original, "suggested_snippet": suggested, "message": message, "resolved": resolved, "created_at": createdAt})
	}
	json.NewEncoder(w).Encode(Ok(reviews))
}

func (h *SessionHandler) ApplyAIReview(w http.ResponseWriter, r *http.Request) {
	log.Printf("[ApplyAIReview] === CALLED ===")
	log.Printf("[ApplyAIReview] URL: %s", r.URL.Path)
	log.Printf("[ApplyAIReview] Method: %s", r.Method)

	user := GetUserFromContext(r.Context())
	if user == nil {
		log.Printf("[ApplyAIReview] User not authenticated")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(Fail(401, "unauthorized"))
		return
	}
	log.Printf("[ApplyAIReview] User: %s", user.UserID)

	// Парсим путь
	path := strings.TrimPrefix(r.URL.Path, "/api/sessions/")
	parts := strings.Split(path, "/")

	if len(parts) < 4 || parts[1] != "ai-reviews" || parts[3] != "apply" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Fail(400, "invalid path format"))
		return
	}

	sessionID, err := uuid.Parse(parts[0])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Fail(400, "invalid session id"))
		return
	}

	reviewID, err := uuid.Parse(parts[2])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Fail(400, "invalid review id"))
		return
	}

	// Получаем ревью из БД
	var originalSnippet, suggestedSnippet, message string
	var reviewType string
	var startLine, endLine int

	err = h.DB.Pool.QueryRow(r.Context(),
		`SELECT type, original_snippet, suggested_snippet, message, start_line, end_line 
		 FROM ai_reviews 
		 WHERE id = $1 AND session_id = $2 AND resolved = false`,
		reviewID, sessionID).Scan(&reviewType, &originalSnippet, &suggestedSnippet, &message, &startLine, &endLine)

	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(Fail(404, "review not found or already applied"))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Fail(500, "database error: "+err.Error()))
		return
	}

	log.Printf("[ApplyAIReview] Original snippet: %s", originalSnippet)
	log.Printf("[ApplyAIReview] Suggested snippet: %s", suggestedSnippet)

	// Получаем имя основного файла из сессии
	var mainFileName string
	err = h.DB.Pool.QueryRow(r.Context(),
		`SELECT file_name FROM sessions WHERE id = $1`, sessionID).Scan(&mainFileName)
	if err != nil {
		mainFileName = "main.py"
	}
	log.Printf("[ApplyAIReview] Main file name: %s", mainFileName)

	// Получаем текущий контент файла
	currentFileContent, currentFileVersion, err := h.FileRepo.GetFile(r.Context(), sessionID, mainFileName)
	if err != nil {
		log.Printf("[ApplyAIReview] Failed to get file: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Fail(500, "failed to get file content"))
		return
	}
	log.Printf("[ApplyAIReview] Current file version: %d", currentFileVersion)
	log.Printf("[ApplyAIReview] Current file content: %s", currentFileContent)

	// Применяем исправление
	var newContent string
	if suggestedSnippet != "" {
		// Полная замена содержимого файла на предложенный код
		newContent = suggestedSnippet
		log.Printf("[ApplyAIReview] Replacing entire content with suggested snippet")
	} else {
		newContent = currentFileContent
		log.Printf("[ApplyAIReview] No suggested snippet, keeping current content")
	}

	log.Printf("[ApplyAIReview] Old content: %s", currentFileContent)
	log.Printf("[ApplyAIReview] New content: %s", newContent)

	log.Printf("[ApplyAIReview] New content: %s", newContent)

	// Обновляем файл
	newVersion, err := h.FileRepo.UpdateFile(r.Context(), sessionID, mainFileName, newContent, currentFileVersion)
	if err != nil {
		log.Printf("[ApplyAIReview] Failed to update file: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Fail(500, "failed to update file: "+err.Error()))
		return
	}
	log.Printf("[ApplyAIReview] File updated to version: %d", newVersion)

	// Сохраняем снепшот
	if err := h.SnapRepo.Save(sessionID, newContent, newVersion); err != nil {
		log.Printf("[ApplyAIReview] Failed to save snapshot: %v", err)
	}

	// Обновляем сессию
	if _, err := h.DB.Pool.Exec(r.Context(),
		`UPDATE sessions SET content = $1, version = $2 WHERE id = $3`,
		newContent, newVersion, sessionID); err != nil {
		log.Printf("[ApplyAIReview] Failed to update session: %v", err)
	}

	// Отмечаем ревью как примененное
	_, err = h.DB.Pool.Exec(r.Context(),
		`UPDATE ai_reviews SET resolved = true WHERE id = $1`, reviewID)
	if err != nil {
		log.Printf("[ApplyAIReview] Failed to update review: %v", err)
	}

	// Добавляем событие
	userID := uuid.Nil
	if parsed, err := uuid.Parse(user.UserID); err == nil {
		userID = parsed
	}
	_ = h.SnapRepo.AddEvent(sessionID, userID, "ai_review_applied", map[string]any{
		"review_id": reviewID,
		"message":   message,
		"version":   newVersion,
		"username":  user.Username,
	})

	// Рассылаем обновление через WebSocket
	room := h.Hub.GetOrCreateRoom(sessionID.String())
	if room == nil {
		log.Printf("[ApplyAIReview] Room is nil for session %s", sessionID.String())
	} else {
		log.Printf("[ApplyAIReview] Room found, clients count: %d", len(room.Clients))
		room.SetState(newContent, newVersion)
		data, _ := json.Marshal(map[string]any{
			"type": "full_state",
			"payload": map[string]any{
				"content": newContent,
				"version": newVersion,
			},
		})
		select {
		case room.BroadcastText <- data:
			log.Printf("[ApplyAIReview] WebSocket update sent")
		default:
			log.Printf("[ApplyAIReview] WebSocket channel full")
		}
	}

	json.NewEncoder(w).Encode(Ok(map[string]any{
		"status":  "applied",
		"version": newVersion,
	}))
}

func (h *SessionHandler) GetHistory(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/sessions/"), "/history")
	sessionID, err := uuid.Parse(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Fail(400, "invalid session id"))
		return
	}
	rows, err := h.DB.Pool.Query(r.Context(), `SELECT version, created_at, LEFT(content, 180) FROM session_snapshots WHERE session_id=$1 ORDER BY version DESC LIMIT 30`, sessionID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Fail(500, "database error"))
		return
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var version int64
		var createdAt time.Time
		var preview string
		if err := rows.Scan(&version, &createdAt, &preview); err == nil {
			out = append(out, map[string]any{"version": version, "created_at": createdAt, "preview": preview})
		}
	}
	json.NewEncoder(w).Encode(Ok(out))
}

func (h *SessionHandler) RestoreVersion(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/sessions/"), "/restore")
	sessionID, err := uuid.Parse(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Fail(400, "invalid session id"))
		return
	}
	var req struct {
		Version int64 `json:"version"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Fail(400, "invalid request"))
		return
	}
	var content string
	if err := h.DB.Pool.QueryRow(r.Context(), `SELECT content FROM session_snapshots WHERE session_id=$1 AND version=$2`, sessionID, req.Version).Scan(&content); err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(Fail(404, "version not found"))
		return
	}
	_, latestVersion, _ := h.SnapRepo.LoadLatest(sessionID)
	newVersion := latestVersion + 1
	if err := h.SnapRepo.Save(sessionID, content, newVersion); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Fail(500, "restore failed"))
		return
	}
	room := h.Hub.GetOrCreateRoom(sessionID.String())
	if room != nil {
		room.SetState(content, newVersion)
		data, _ := json.Marshal(map[string]any{"type": "full_state", "payload": map[string]any{"content": content, "version": newVersion}})
		room.BroadcastText <- data
	}
	json.NewEncoder(w).Encode(Ok(map[string]any{"version": newVersion, "content": content}))
}

func (h *SessionHandler) GetLeaderboard(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/sessions/"), "/leaderboard")
	sessionID, err := uuid.Parse(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Fail(400, "invalid session id"))
		return
	}
	items, err := h.ScoreRepo.Leaderboard(r.Context(), sessionID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Fail(500, "database error"))
		return
	}
	json.NewEncoder(w).Encode(Ok(items))
}

func (h *SessionHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(Fail(401, "unauthorized"))
		return
	}

	idStr := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/sessions/"), "/profile")
	sessionID, err := uuid.Parse(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Fail(400, "invalid session id"))
		return
	}

	userUUID := uuid.MustParse(user.UserID)

	// Проверяем, является ли пользователь участником сессии
	var isParticipant bool
	err = h.DB.Pool.QueryRow(r.Context(),
		`SELECT EXISTS(SELECT 1 FROM session_participants WHERE session_id = $1 AND user_id = $2)`,
		sessionID, userUUID).Scan(&isParticipant)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Fail(500, "database error: "+err.Error()))
		return
	}

	if !isParticipant {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(Fail(403, "you are not a participant of this session"))
		return
	}

	var req struct {
		Incognito bool   `json:"incognito"`
		Nickname  string `json:"nickname"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Fail(400, "invalid request"))
		return
	}

	if err := h.ScoreRepo.SetProfile(r.Context(), sessionID, userUUID, user.Username, req.Incognito, req.Nickname); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Fail(500, "database error"))
		return
	}

	json.NewEncoder(w).Encode(Ok(map[string]any{
		"incognito": req.Incognito,
		"nickname":  req.Nickname,
	}))
}
func (h *SessionHandler) GetHint(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/sessions/"), "/hint")
	sessionID, err := uuid.Parse(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Fail(400, "invalid session id"))
		return
	}
	content, _, err := h.SnapRepo.LoadLatest(sessionID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Fail(500, "database error"))
		return
	}
	if h.AI == nil {
		json.NewEncoder(w).Encode(Ok(map[string]string{"hint": "AI отключен"}))
		return
	}
	hint, err := h.AI.NavigatorHint(content)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(Fail(502, err.Error()))
		return
	}
	json.NewEncoder(w).Encode(Ok(map[string]string{"hint": hint}))
}

func (h *SessionHandler) FileRouter(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/sessions/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	sessionID, err := uuid.Parse(parts[0])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Fail(400, "invalid session id"))
		return
	}
	filePath := ""
	if len(parts) > 2 {
		filePath, _ = url.QueryUnescape(strings.Join(parts[2:], "/"))
	}
	if !h.checkSessionAccess(r.Context(), sessionID) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(Fail(403, "access denied"))
		return
	}
	switch r.Method {
	case http.MethodGet:
		if filePath == "" {
			h.ListFiles(w, r, sessionID)
		} else {
			h.GetFileContent(w, r, sessionID, filePath)
		}
	case http.MethodPost:
		h.CreateFile(w, r, sessionID)
	case http.MethodPut:
		h.UpdateFileContent(w, r, sessionID, filePath)
	case http.MethodDelete:
		h.DeleteFile(w, r, sessionID, filePath)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *SessionHandler) ListFiles(w http.ResponseWriter, r *http.Request, sessionID uuid.UUID) {
	files, err := h.FileRepo.ListFiles(r.Context(), sessionID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Fail(500, "database error"))
		return
	}
	json.NewEncoder(w).Encode(Ok(files))
}

func (h *SessionHandler) CreateFile(w http.ResponseWriter, r *http.Request, sessionID uuid.UUID) {
	var req struct {
		Path    string `json:"path"`
		IsDir   bool   `json:"is_dir"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Fail(400, "invalid request"))
		return
	}
	req.Path = strings.Trim(req.Path, "/")
	var contentPtr *string
	if !req.IsDir {
		contentPtr = &req.Content
	}
	id, err := h.FileRepo.CreateFile(r.Context(), sessionID, req.Path, req.IsDir, contentPtr)
	if err != nil {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(Fail(409, err.Error()))
		return
	}
	h.broadcastFileEvent(sessionID.String(), "file_created", map[string]any{"id": id, "path": req.Path, "is_dir": req.IsDir})
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(Ok(map[string]any{"id": id, "path": req.Path, "is_dir": req.IsDir}))
}

func (h *SessionHandler) GetFileContent(w http.ResponseWriter, r *http.Request, sessionID uuid.UUID, filePath string) {
	content, version, err := h.FileRepo.GetFile(r.Context(), sessionID, filePath)
	if err == repository.ErrFileNotFound {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(Fail(404, "file not found"))
		return
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Fail(500, "database error"))
		return
	}
	json.NewEncoder(w).Encode(Ok(map[string]any{"content": content, "version": version}))
}

func (h *SessionHandler) UpdateFileContent(w http.ResponseWriter, r *http.Request, sessionID uuid.UUID, filePath string) {
	var req struct {
		Content     string `json:"content"`
		BaseVersion int64  `json:"base_version"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Fail(400, "invalid request"))
		return
	}
	newVersion, err := h.FileRepo.UpdateFile(r.Context(), sessionID, filePath, req.Content, req.BaseVersion)
	if err != nil {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(Fail(409, err.Error()))
		return
	}
	h.broadcastFileEvent(sessionID.String(), "file_updated", map[string]any{"path": filePath, "version": newVersion})
	json.NewEncoder(w).Encode(Ok(map[string]any{"new_version": newVersion}))
}

func (h *SessionHandler) DeleteFile(w http.ResponseWriter, r *http.Request, sessionID uuid.UUID, filePath string) {
	if err := h.FileRepo.DeleteFile(r.Context(), sessionID, filePath); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Fail(500, "failed to delete"))
		return
	}
	h.broadcastFileEvent(sessionID.String(), "file_deleted", map[string]any{"path": filePath})
	json.NewEncoder(w).Encode(Ok(map[string]string{"status": "deleted"}))
}

func (h *SessionHandler) checkSessionAccess(ctx context.Context, sessionID uuid.UUID) bool {
	user := GetUserFromContext(ctx)
	if user == nil {
		return false
	}
	var exists bool
	err := h.DB.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM sessions WHERE id=$1)`, sessionID).Scan(&exists)
	return err == nil && exists
}

func (h *SessionHandler) broadcastFileEvent(roomID, eventType string, payload map[string]any) {
	room := h.Hub.GetOrCreateRoom(roomID)
	if room == nil {
		return
	}
	data, _ := json.Marshal(map[string]any{"type": eventType, "payload": payload})
	select {
	case room.BroadcastText <- data:
	default:
	}
}

// InviteUser - приглашение пользователя в сессию
func (h *SessionHandler) InviteUser(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(Fail(401, "unauthorized"))
		return
	}

	idStr := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/sessions/"), "/invite")
	sessionID, err := uuid.Parse(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Fail(400, "invalid session id"))
		return
	}

	var req struct {
		Username string `json:"username"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Fail(400, "invalid request"))
		return
	}

	// Проверяем, существует ли пользователь
	var targetUserID uuid.UUID
	err = h.DB.Pool.QueryRow(r.Context(),
		`SELECT id FROM users WHERE username = $1`, req.Username).Scan(&targetUserID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(Fail(404, "user not found"))
		return
	}

	// Проверяем, не является ли пользователь уже участником
	var exists bool
	err = h.DB.Pool.QueryRow(r.Context(),
		`SELECT EXISTS(SELECT 1 FROM session_participants WHERE session_id = $1 AND user_id = $2)`,
		sessionID, targetUserID).Scan(&exists)
	if err == nil && exists {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(Fail(409, "user already in session"))
		return
	}

	// Добавляем пользователя в сессию
	_, err = h.DB.Pool.Exec(r.Context(),
		`INSERT INTO session_participants (session_id, user_id, joined_at)
		 VALUES ($1, $2, NOW())`,
		sessionID, targetUserID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Fail(500, "failed to invite user"))
		return
	}

	// Добавляем событие
	_ = h.SnapRepo.AddEvent(sessionID, targetUserID, "join", map[string]any{
		"username":   req.Username,
		"invited_by": user.Username,
	})

	// Уведомляем через WebSocket
	if room := h.Hub.GetOrCreateRoom(sessionID.String()); room != nil {
		data, _ := json.Marshal(map[string]any{
			"type": "user_invited",
			"payload": map[string]any{
				"user_id":  targetUserID,
				"username": req.Username,
			},
		})
		select {
		case room.BroadcastText <- data:
		default:
		}
	}

	json.NewEncoder(w).Encode(Ok(map[string]any{
		"status": "invited",
		"user":   req.Username,
	}))
}

// GetParticipants - список участников сессии
func (h *SessionHandler) GetParticipants(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/sessions/"), "/participants")
	sessionID, err := uuid.Parse(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Fail(400, "invalid session id"))
		return
	}

	rows, err := h.DB.Pool.Query(r.Context(),
		`SELECT u.id, u.username, sp.joined_at
		 FROM session_participants sp
		 JOIN users u ON u.id = sp.user_id
		 WHERE sp.session_id = $1
		 ORDER BY sp.joined_at ASC`,
		sessionID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Fail(500, "database error"))
		return
	}
	defer rows.Close()

	var participants []map[string]any
	for rows.Next() {
		var id uuid.UUID
		var username string
		var joinedAt time.Time
		if err := rows.Scan(&id, &username, &joinedAt); err != nil {
			continue
		}
		participants = append(participants, map[string]any{
			"user_id":   id,
			"username":  username,
			"joined_at": joinedAt,
		})
	}

	json.NewEncoder(w).Encode(Ok(participants))
}

// RemoveParticipant - удаление участника из сессии (только владелец)
func (h *SessionHandler) RemoveParticipant(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(Fail(401, "unauthorized"))
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/sessions/")
	parts := strings.Split(path, "/")
	if len(parts) < 3 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Fail(400, "invalid path"))
		return
	}

	sessionID, err := uuid.Parse(parts[0])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Fail(400, "invalid session id"))
		return
	}

	targetUserID, err := uuid.Parse(parts[2])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Fail(400, "invalid user id"))
		return
	}

	// Проверяем, что текущий пользователь - владелец сессии
	var ownerID uuid.UUID
	err = h.DB.Pool.QueryRow(r.Context(),
		`SELECT owner_id FROM sessions WHERE id = $1`, sessionID).Scan(&ownerID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(Fail(404, "session not found"))
		return
	}

	if ownerID.String() != user.UserID {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(Fail(403, "only session owner can remove participants"))
		return
	}

	// Удаляем участника
	_, err = h.DB.Pool.Exec(r.Context(),
		`DELETE FROM session_participants WHERE session_id = $1 AND user_id = $2`,
		sessionID, targetUserID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Fail(500, "failed to remove participant"))
		return
	}

	// Добавляем событие
	var username string
	h.DB.Pool.QueryRow(r.Context(), `SELECT username FROM users WHERE id = $1`, targetUserID).Scan(&username)
	_ = h.SnapRepo.AddEvent(sessionID, targetUserID, "leave", map[string]any{
		"username":   username,
		"removed_by": user.Username,
	})

	// Уведомляем через WebSocket
	if room := h.Hub.GetOrCreateRoom(sessionID.String()); room != nil {
		data, _ := json.Marshal(map[string]any{
			"type": "user_removed",
			"payload": map[string]any{
				"user_id":  targetUserID,
				"username": username,
			},
		})
		select {
		case room.BroadcastText <- data:
		default:
		}
	}

	json.NewEncoder(w).Encode(Ok(map[string]any{
		"status":  "removed",
		"user_id": targetUserID,
	}))
}

// GetProfile - получить настройки профиля пользователя в сессии
func (h *SessionHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(Fail(401, "unauthorized"))
		return
	}

	idStr := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/sessions/"), "/profile")
	sessionID, err := uuid.Parse(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Fail(400, "invalid session id"))
		return
	}

	userUUID := uuid.MustParse(user.UserID)

	// Проверяем, является ли пользователь участником сессии
	var isParticipant bool
	err = h.DB.Pool.QueryRow(r.Context(),
		`SELECT EXISTS(SELECT 1 FROM session_participants WHERE session_id = $1 AND user_id = $2)`,
		sessionID, userUUID).Scan(&isParticipant)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Fail(500, "database error: "+err.Error()))
		return
	}

	if !isParticipant {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(Fail(403, "you are not a participant of this session"))
		return
	}

	var incognito bool
	var nickname string
	err = h.DB.Pool.QueryRow(r.Context(),
		`SELECT incognito, nickname FROM session_scores 
         WHERE session_id = $1 AND user_id = $2`,
		sessionID, userUUID).Scan(&incognito, &nickname)

	if err != nil {
		// Если нет записи, возвращаем значения по умолчанию
		incognito = false
		nickname = ""
	}

	json.NewEncoder(w).Encode(Ok(map[string]any{
		"incognito": incognito,
		"nickname":  nickname,
	}))
}

// GetInviteLink - получить пригласительную ссылку для сессии
func (h *SessionHandler) GetInviteLink(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(Fail(401, "unauthorized"))
		return
	}

	idStr := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/sessions/"), "/invite-link")
	sessionID, err := uuid.Parse(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Fail(400, "invalid session id"))
		return
	}

	// Генерируем токен
	inviteToken := uuid.New().String()

	// Сохраняем токен в БД
	_, err = h.DB.Pool.Exec(r.Context(),
		`INSERT INTO session_invites (session_id, token, created_at, expires_at)
         VALUES ($1, $2, NOW(), NOW() + INTERVAL '7 days')
         ON CONFLICT (session_id) DO UPDATE 
         SET token = EXCLUDED.token, expires_at = EXCLUDED.expires_at`,
		sessionID, inviteToken)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Fail(500, "failed to generate invite link"))
		return
	}

	// Формируем ссылку
	inviteLink := fmt.Sprintf("http://localhost:5173/web/session/%s?invite=%s", sessionID, inviteToken)

	json.NewEncoder(w).Encode(Ok(map[string]any{
		"invite_link": inviteLink,
		"session_id":  sessionID,
		"token":       inviteToken,
	}))
}

// JoinByInvite - присоединиться по приглашению
func (h *SessionHandler) JoinByInvite(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(Fail(401, "unauthorized"))
		return
	}

	var req struct {
		InviteToken string `json:"invite_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Fail(400, "invalid request"))
		return
	}

	// Проверяем токен
	var sessionID uuid.UUID
	var expiresAt time.Time
	err := h.DB.Pool.QueryRow(r.Context(),
		`SELECT session_id, expires_at FROM session_invites WHERE token = $1`,
		req.InviteToken).Scan(&sessionID, &expiresAt)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(Fail(404, "invalid invite token"))
		return
	}

	if time.Now().After(expiresAt) {
		w.WriteHeader(http.StatusGone)
		json.NewEncoder(w).Encode(Fail(410, "invite expired"))
		return
	}

	// Добавляем пользователя в participants
	_, err = h.DB.Pool.Exec(r.Context(),
		`INSERT INTO session_participants (session_id, user_id, joined_at)
         VALUES ($1, $2, NOW())
         ON CONFLICT (session_id, user_id) DO NOTHING`,
		sessionID, uuid.MustParse(user.UserID))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Fail(500, "failed to join session"))
		return
	}

	// Получаем информацию о сессии
	var sessionName string
	h.DB.Pool.QueryRow(r.Context(),
		`SELECT name FROM sessions WHERE id = $1`, sessionID).Scan(&sessionName)

	json.NewEncoder(w).Encode(Ok(map[string]any{
		"session_id":   sessionID,
		"session_name": sessionName,
		"joined":       true,
	}))
}
