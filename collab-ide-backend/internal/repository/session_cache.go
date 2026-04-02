package repository

import (
	"collab-ide-backend/internal/models"
	"context"
	"encoding/json"
	"time"
)

type SessionCache struct {
	redis *RedisRepo
}

func NewSessionCache(redis *RedisRepo) *SessionCache {
	return &SessionCache{redis: redis}
}

func (c *SessionCache) Get(sessionID string) (*models.Session, error) {
	data, err := c.redis.Client.Get(context.Background(), "session:"+sessionID).Bytes()
	if err != nil {
		return nil, err
	}

	var session models.Session
	json.Unmarshal(data, &session)
	return &session, nil
}

func (c *SessionCache) Set(session *models.Session) error {
	data, _ := json.Marshal(session)
	return c.redis.Client.Set(context.Background(), "session:"+session.ID.String(), data, 5*time.Minute).Err()
}
