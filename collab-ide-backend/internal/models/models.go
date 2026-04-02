package models

import (
	"time"

	"github.com/google/uuid"
)

type Session struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	FileName  string    `json:"file_name"`
	Language  string    `json:"language"`
	OwnerID   uuid.UUID `json:"owner_id"`
	CreatedAt time.Time `json:"created_at"`
	Active    bool      `json:"active"`
	Content   string    `json:"content"`
	Version   int64     `json:"version"`
}

type Participant struct {
	SessionID  uuid.UUID `json:"session_id"`
	UserID     uuid.UUID `json:"user_id"`
	Username   string    `json:"username"`
	Avatar     string    `json:"avatar"`
	CursorLine int       `json:"cursor_line"`
	CursorCol  int       `json:"cursor_col"`
	JoinedAt   time.Time `json:"joined_at"`
}

type AIReview struct {
	ID               uuid.UUID `json:"id"`
	SessionID        uuid.UUID `json:"session_id"`
	Type             string    `json:"type"`
	StartLine        int       `json:"start_line"`
	EndLine          int       `json:"end_line"`
	OriginalSnippet  string    `json:"original_snippet"`
	SuggestedSnippet string    `json:"suggested_snippet"`
	Message          string    `json:"message"`
	Resolved         bool      `json:"resolved"`
	CreatedAt        time.Time `json:"created_at"`
}

type User struct {
	ID           uuid.UUID `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}
