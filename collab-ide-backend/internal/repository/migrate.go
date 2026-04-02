package repository

import "context"

func Migrate(db *PostgresRepo) error {
	queries := []string{
		`CREATE EXTENSION IF NOT EXISTS "pgcrypto"`,
		`CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY,
			username TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			telegram_id BIGINT UNIQUE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS sessions (
			id UUID PRIMARY KEY,
			name TEXT NOT NULL,
			file_name TEXT NOT NULL,
			language TEXT NOT NULL,
			owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			active BOOLEAN NOT NULL DEFAULT TRUE,
			content TEXT NOT NULL DEFAULT '',
			version BIGINT NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS session_invites (
			session_id UUID PRIMARY KEY REFERENCES sessions(id) ON DELETE CASCADE,
			token TEXT UNIQUE NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			expires_at TIMESTAMPTZ NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS session_snapshots (
			session_id UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
			version BIGINT NOT NULL,
			content TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY (session_id, version)
		)`,
		`CREATE TABLE IF NOT EXISTS session_events (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			session_id UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
			user_id UUID NULL REFERENCES users(id) ON DELETE SET NULL,
			event_type TEXT NOT NULL,
			details JSONB NOT NULL DEFAULT '{}'::jsonb,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS ai_reviews (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			session_id UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
			type TEXT NOT NULL,
			start_line INT NOT NULL DEFAULT 1,
			end_line INT NOT NULL DEFAULT 1,
			original_snippet TEXT NOT NULL DEFAULT '',
			suggested_snippet TEXT NOT NULL DEFAULT '',
			message TEXT NOT NULL DEFAULT '',
			resolved BOOLEAN NOT NULL DEFAULT FALSE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS session_files (
			id UUID PRIMARY KEY,
			session_id UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
			path TEXT NOT NULL,
			is_dir BOOLEAN NOT NULL DEFAULT FALSE,
			content TEXT,
			version BIGINT NOT NULL DEFAULT 0,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE(session_id, path)
		)`,
		`CREATE TABLE IF NOT EXISTS session_scores (
			session_id UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			username TEXT NOT NULL,
			points BIGINT NOT NULL DEFAULT 0,
			incognito BOOLEAN NOT NULL DEFAULT FALSE,
			nickname TEXT NOT NULL DEFAULT '',
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY (session_id, user_id)
		)`,
		`CREATE TABLE IF NOT EXISTS session_participants (
			session_id UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY (session_id, user_id)
		)`,
	}
	for _, q := range queries {
		if _, err := db.Pool.Exec(context.Background(), q); err != nil {
			return err
		}
	}
	return nil
}
