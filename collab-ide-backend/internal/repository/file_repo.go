package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrFileNotFound    = errors.New("file not found")
	ErrVersionConflict = errors.New("version conflict")
	ErrAlreadyExists   = errors.New("file or folder already exists")
)

type FileRepo struct {
	db *PostgresRepo
}

func NewFileRepo(db *PostgresRepo) *FileRepo {
	return &FileRepo{db: db}
}

// Exists проверяет существование файла/папки
func (r *FileRepo) Exists(ctx context.Context, sessionID uuid.UUID, path string) (bool, error) {
	var exists bool
	err := r.db.Pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM session_files WHERE session_id = $1 AND path = $2)`,
		sessionID, path).Scan(&exists)
	return exists, err
}

// ListFiles возвращает плоский список всех объектов в сессии
func (r *FileRepo) ListFiles(ctx context.Context, sessionID uuid.UUID) ([]map[string]interface{}, error) {
	rows, err := r.db.Pool.Query(ctx,
		`SELECT id, path, is_dir, content, version, created_at, updated_at
         FROM session_files WHERE session_id = $1 ORDER BY path`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []map[string]interface{}
	for rows.Next() {
		var id uuid.UUID
		var path string
		var isDir bool
		var content sql.NullString
		var version int64
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&id, &path, &isDir, &content, &version, &createdAt, &updatedAt); err != nil {
			continue
		}
		item := map[string]interface{}{
			"id":         id,
			"path":       path,
			"is_dir":     isDir,
			"created_at": createdAt,
			"updated_at": updatedAt,
		}
		if !isDir {
			item["version"] = version
			if content.Valid {
				item["content"] = content.String // для удобства (но в списке лучше не передавать)
			}
		}
		files = append(files, item)
	}
	return files, nil
}

// CreateFile создаёт новый файл или папку
func (r *FileRepo) CreateFile(ctx context.Context, sessionID uuid.UUID, path string, isDir bool, content *string) (*uuid.UUID, error) {
	id := uuid.New()
	_, err := r.db.Pool.Exec(ctx,
		`INSERT INTO session_files (id, session_id, path, is_dir, content, version)
         VALUES ($1, $2, $3, $4, $5, 0)`,
		id, sessionID, path, isDir, content)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return nil, ErrAlreadyExists
		}
		return nil, err
	}
	return &id, nil
}

// GetFile возвращает содержимое и версию файла
func (r *FileRepo) GetFile(ctx context.Context, sessionID uuid.UUID, path string) (string, int64, error) {
	var content string
	var version int64
	err := r.db.Pool.QueryRow(ctx,
		`SELECT content, version FROM session_files
         WHERE session_id = $1 AND path = $2 AND is_dir = false`,
		sessionID, path).Scan(&content, &version)
	if err == sql.ErrNoRows {
		return "", 0, ErrFileNotFound
	}
	return content, version, err
}

// UpdateFile обновляет содержимое файла с проверкой версии
func (r *FileRepo) UpdateFile(ctx context.Context, sessionID uuid.UUID, path, content string, baseVersion int64) (int64, error) {
	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	var currentVersion int64
	err = tx.QueryRow(ctx,
		`SELECT version FROM session_files
         WHERE session_id = $1 AND path = $2 AND is_dir = false FOR UPDATE`,
		sessionID, path).Scan(&currentVersion)
	if err == sql.ErrNoRows {
		return 0, ErrFileNotFound
	}
	if err != nil {
		return 0, err
	}
	if currentVersion != baseVersion {
		return 0, ErrVersionConflict
	}

	newVersion := currentVersion + 1
	_, err = tx.Exec(ctx,
		`UPDATE session_files SET content = $1, version = $2, updated_at = NOW()
         WHERE session_id = $3 AND path = $4`,
		content, newVersion, sessionID, path)
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return newVersion, nil
}

// DeleteFile рекурсивно удаляет файл или папку
func (r *FileRepo) DeleteFile(ctx context.Context, sessionID uuid.UUID, path string) error {
	// удаляем саму запись и все записи, у которых путь начинается с path/ (для папок)
	_, err := r.db.Pool.Exec(ctx,
		`DELETE FROM session_files
         WHERE session_id = $1 AND (path = $2 OR path LIKE $2 || '/%')`,
		sessionID, path)
	return err
}
