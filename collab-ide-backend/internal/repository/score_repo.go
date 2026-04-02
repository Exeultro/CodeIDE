package repository

import (
	"context"
	"github.com/google/uuid"
)

type ScoreRepo struct{ db *PostgresRepo }

func NewScoreRepo(db *PostgresRepo) *ScoreRepo { return &ScoreRepo{db: db} }

func (r *ScoreRepo) AddPoints(ctx context.Context, sessionID, userID uuid.UUID, username string, points int64) error {
	_, err := r.db.Pool.Exec(ctx, `INSERT INTO session_scores (session_id, user_id, username, points)
		VALUES ($1,$2,$3,$4)
		ON CONFLICT (session_id,user_id) DO UPDATE SET
		username = EXCLUDED.username,
		points = session_scores.points + EXCLUDED.points,
		updated_at = NOW()`, sessionID, userID, username, points)
	return err
}

func (r *ScoreRepo) SetProfile(ctx context.Context, sessionID, userID uuid.UUID, username string, incognito bool, nickname string) error {
	_, err := r.db.Pool.Exec(ctx, `INSERT INTO session_scores (session_id, user_id, username, incognito, nickname)
		VALUES ($1,$2,$3,$4,$5)
		ON CONFLICT (session_id,user_id) DO UPDATE SET username=EXCLUDED.username, incognito=EXCLUDED.incognito, nickname=EXCLUDED.nickname, updated_at=NOW()`,
		sessionID, userID, username, incognito, nickname)
	return err
}

func (r *ScoreRepo) Leaderboard(ctx context.Context, sessionID uuid.UUID) ([]map[string]any, error) {
	rows, err := r.db.Pool.Query(ctx, `SELECT user_id, username, points, incognito, nickname, updated_at
		FROM session_scores WHERE session_id=$1 ORDER BY points DESC, updated_at ASC`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var userID uuid.UUID
		var username, nickname string
		var points int64
		var incognito bool
		var updatedAt any
		if err := rows.Scan(&userID, &username, &points, &incognito, &nickname, &updatedAt); err != nil {
			continue
		}
		display := username
		if incognito {
			if nickname != "" {
				display = nickname
			} else {
				display = "Инкогнито"
			}
		}
		out = append(out, map[string]any{"user_id": userID, "username": username, "display_name": display, "points": points, "incognito": incognito, "nickname": nickname, "updated_at": updatedAt})
	}
	return out, nil
}
