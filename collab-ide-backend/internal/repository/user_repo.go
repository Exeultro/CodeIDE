package repository

import (
	"context"
	"database/sql"
	"log"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type UserRepo struct {
	db *PostgresRepo
}

func NewUserRepo(db *PostgresRepo) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) CreateUser(ctx context.Context, username, password string) (*uuid.UUID, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	id := uuid.New()
	_, err = r.db.Pool.Exec(ctx,
		`INSERT INTO users (id, username, password_hash) VALUES ($1, $2, $3)`,
		id, username, string(hashed))
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func (r *UserRepo) FindByUsername(ctx context.Context, username string) (*uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.Pool.QueryRow(ctx,
		`SELECT id FROM users WHERE username = $1`, username).Scan(&id)

	if err != nil && err.Error() == "sql: no rows in result set" {
		return nil, nil
	}

	if err != nil {
		log.Printf("FindByUsername error for '%s': %v", username, err)
		return nil, err
	}

	return &id, nil
}

func (r *UserRepo) Authenticate(ctx context.Context, username, password string) (*uuid.UUID, error) {
	var id uuid.UUID
	var hash string
	err := r.db.Pool.QueryRow(ctx,
		`SELECT id, password_hash FROM users WHERE username = $1`, username).
		Scan(&id, &hash)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		log.Printf("Authenticate error for '%s': %v", username, err)
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return nil, nil
	}

	return &id, nil
}
