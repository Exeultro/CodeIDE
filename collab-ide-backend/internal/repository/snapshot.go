package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
)

type SnapshotRepo struct {
	db *PostgresRepo
}

func NewSnapshotRepo(db *PostgresRepo) *SnapshotRepo {
	return &SnapshotRepo{db: db}
}

// LoadLatest возвращает последнее содержимое и версию документа
func (r *SnapshotRepo) LoadLatest(sessionID uuid.UUID) (string, int64, error) {
	var content string
	var version int64
	err := r.db.Pool.QueryRow(context.Background(),
		`SELECT content, version FROM session_snapshots 
         WHERE session_id = $1 ORDER BY version DESC LIMIT 1`, sessionID).
		Scan(&content, &version)
	if err == sql.ErrNoRows {
		return "", 0, nil
	}
	return content, version, err
}

// Save сохраняет снепшот документа с указанной версией
func (r *SnapshotRepo) Save(sessionID uuid.UUID, content string, version int64) error {
	_, err := r.db.Pool.Exec(context.Background(),
		`INSERT INTO session_snapshots (session_id, version, content) 
         VALUES ($1, $2, $3)
         ON CONFLICT (session_id, version) DO UPDATE SET content = EXCLUDED.content`,
		sessionID, version, content)
	return err
}

func (r *SnapshotRepo) AddEvent(sessionID uuid.UUID, userID uuid.UUID, eventType string, details map[string]interface{}) error {
	detailsJSON, _ := json.Marshal(details)

	// Если userID нулевой, используем NULL в БД
	var userIDPtr interface{}
	if userID == uuid.Nil {
		userIDPtr = nil
	} else {
		userIDPtr = userID
	}

	_, err := r.db.Pool.Exec(context.Background(),
		`INSERT INTO session_events (session_id, user_id, event_type, details) 
         VALUES ($1, $2, $3, $4)`,
		sessionID, userIDPtr, eventType, detailsJSON)
	return err
}

func (r *SnapshotRepo) SaveAIReview(sessionID uuid.UUID, review string, source string) error {
	message := review
	suggestedSnippet := ""

	// Парсим ответ AI
	if strings.Contains(review, "Исправление:") {
		parts := strings.SplitN(review, "Исправление:", 2)

		// Извлекаем проблему
		problemPart := strings.TrimSpace(parts[0])
		problemPart = strings.TrimPrefix(problemPart, "Проблема:")
		message = strings.TrimSpace(problemPart)

		// Извлекаем исправление
		if len(parts) > 1 {
			suggestedSnippet = strings.TrimSpace(parts[1])
			// Убираем лишние символы
			suggestedSnippet = strings.Trim(suggestedSnippet, "`\n")
		}
	}

	// Если нет формата, пробуем извлечь код из текста
	if suggestedSnippet == "" && strings.Contains(review, "```") {
		// Ищем код между тройными бэктиками
		start := strings.Index(review, "```")
		if start != -1 {
			end := strings.Index(review[start+3:], "```")
			if end != -1 {
				suggestedSnippet = review[start+3 : start+3+end]
				suggestedSnippet = strings.TrimSpace(suggestedSnippet)
			}
		}
	}

	// Обрезаем длинные строки
	if len(suggestedSnippet) > 500 {
		suggestedSnippet = suggestedSnippet[:500]
	}
	if len(message) > 500 {
		message = message[:500]
	}

	log.Printf("[AI] Message: %s", message)
	log.Printf("[AI] Suggested snippet length: %d", len(suggestedSnippet))

	_, err := r.db.Pool.Exec(context.Background(),
		`INSERT INTO ai_reviews (session_id, type, start_line, end_line, original_snippet, suggested_snippet, message, resolved)
         VALUES ($1, $2, 1, 1, $3, $4, $5, false)`,
		sessionID, "review", truncate(source, 400), suggestedSnippet, message)
	return err
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// GetHistoryWithPagination возвращает историю с пагинацией
func (r *SnapshotRepo) GetHistoryWithPagination(sessionID uuid.UUID, limit, offset int) ([]map[string]any, error) {
	rows, err := r.db.Pool.Query(context.Background(),
		`SELECT version, created_at, LEFT(content, 180) 
         FROM session_snapshots 
         WHERE session_id=$1 
         ORDER BY version DESC 
         LIMIT $2 OFFSET $3`,
		sessionID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []map[string]any
	for rows.Next() {
		var version int64
		var createdAt time.Time
		var preview string
		if err := rows.Scan(&version, &createdAt, &preview); err != nil {
			continue
		}
		result = append(result, map[string]any{
			"version":    version,
			"created_at": createdAt,
			"preview":    preview,
		})
	}
	return result, nil
}

// GetSnapshotByVersion возвращает снепшот по версии
func (r *SnapshotRepo) GetSnapshotByVersion(sessionID uuid.UUID, version int64) (string, error) {
	var content string
	err := r.db.Pool.QueryRow(context.Background(),
		`SELECT content FROM session_snapshots 
         WHERE session_id=$1 AND version=$2`,
		sessionID, version).Scan(&content)
	return content, err
}
