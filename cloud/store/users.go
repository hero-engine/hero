package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// User represents a Hero Cloud user.
type User struct {
	ID         string    `json:"id"`
	Email      string    `json:"email"`
	Name       string    `json:"name"`
	AvatarURL  string    `json:"avatar_url"`
	Provider   string    `json:"provider"`
	ProviderID string    `json:"provider_id"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// UpsertUser creates or updates a user by provider+provider_id.
// Returns the user (with ID populated).
func (db *DB) UpsertUser(ctx context.Context, u *User) (*User, error) {
	var result User
	err := db.pool.QueryRow(ctx, `
		INSERT INTO users (email, name, avatar_url, provider, provider_id)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (provider, provider_id) WHERE provider_id != ''
		DO UPDATE SET
			email = EXCLUDED.email,
			name = EXCLUDED.name,
			avatar_url = EXCLUDED.avatar_url,
			updated_at = now()
		RETURNING id, email, name, avatar_url, provider, provider_id, created_at, updated_at
	`, u.Email, u.Name, u.AvatarURL, u.Provider, u.ProviderID,
	).Scan(
		&result.ID, &result.Email, &result.Name, &result.AvatarURL,
		&result.Provider, &result.ProviderID, &result.CreatedAt, &result.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("upserting user: %w", err)
	}
	return &result, nil
}

// GetUserByID retrieves a user by ID.
func (db *DB) GetUserByID(ctx context.Context, id string) (*User, error) {
	var u User
	err := db.pool.QueryRow(ctx, `
		SELECT id, email, name, avatar_url, provider, provider_id, created_at, updated_at
		FROM users WHERE id = $1
	`, id).Scan(
		&u.ID, &u.Email, &u.Name, &u.AvatarURL,
		&u.Provider, &u.ProviderID, &u.CreatedAt, &u.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting user: %w", err)
	}
	return &u, nil
}

// StoreRefreshToken saves a hashed refresh token for a user.
func (db *DB) StoreRefreshToken(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error {
	_, err := db.pool.Exec(ctx, `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
	`, userID, tokenHash, expiresAt)
	if err != nil {
		return fmt.Errorf("storing refresh token: %w", err)
	}
	return nil
}

// ValidateRefreshToken checks if a hashed refresh token exists and is not expired.
// Returns the user_id if valid.
func (db *DB) ValidateRefreshToken(ctx context.Context, tokenHash string) (string, error) {
	var userID string
	err := db.pool.QueryRow(ctx, `
		SELECT user_id FROM refresh_tokens
		WHERE token_hash = $1 AND expires_at > now()
	`, tokenHash).Scan(&userID)
	if err == pgx.ErrNoRows {
		return "", fmt.Errorf("refresh token not found or expired")
	}
	if err != nil {
		return "", fmt.Errorf("validating refresh token: %w", err)
	}
	return userID, nil
}

// RevokeRefreshToken deletes a refresh token by hash.
func (db *DB) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	_, err := db.pool.Exec(ctx, `
		DELETE FROM refresh_tokens WHERE token_hash = $1
	`, tokenHash)
	if err != nil {
		return fmt.Errorf("revoking refresh token: %w", err)
	}
	return nil
}

// RevokeAllRefreshTokens deletes all refresh tokens for a user.
func (db *DB) RevokeAllRefreshTokens(ctx context.Context, userID string) error {
	_, err := db.pool.Exec(ctx, `
		DELETE FROM refresh_tokens WHERE user_id = $1
	`, userID)
	if err != nil {
		return fmt.Errorf("revoking all refresh tokens: %w", err)
	}
	return nil
}
