package rest

import (
	"collab-ide-backend/internal/auth"
	"context"
)

func GetUserFromContext(ctx context.Context) *auth.Claims {
	if val := ctx.Value("user"); val != nil {
		if claims, ok := val.(*auth.Claims); ok {
			return claims
		}
	}
	return nil
}
