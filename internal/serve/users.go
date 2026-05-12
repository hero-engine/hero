package serve

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// User represents a team server user.
type User struct {
	ID           string `json:"id"`
	Username     string `json:"username"`
	Email        string `json:"email"`
	DisplayName  string `json:"display_name"`
	Role         string `json:"role"` // admin, member
	OAuthProvider string `json:"oauth_provider,omitempty"`
	OAuthID      string `json:"oauth_id,omitempty"`
	CreatedAt    string `json:"created_at"`
	LastLogin    string `json:"last_login,omitempty"`
}

// CreateUser adds a new user with a hashed password.
func (jq *JobQueue) CreateUser(username, email, displayName, password, role string) (*User, error) {
	if username == "" {
		return nil, fmt.Errorf("username is required")
	}
	if password == "" {
		return nil, fmt.Errorf("password is required")
	}
	if role == "" {
		role = "member"
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hashing password: %w", err)
	}

	id := generateID()
	now := time.Now().Format(time.RFC3339)

	_, err = jq.db.Exec(`
		INSERT INTO users (id, username, email, display_name, password_hash, role, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id, username, email, displayName, string(hash), role, now)
	if err != nil {
		return nil, fmt.Errorf("creating user: %w", err)
	}

	return &User{
		ID: id, Username: username, Email: email,
		DisplayName: displayName, Role: role, CreatedAt: now,
	}, nil
}

// AuthenticateUser validates username/password and returns the user.
func (jq *JobQueue) AuthenticateUser(username, password string) (*User, error) {
	var user User
	var hash string

	err := jq.db.QueryRow(`
		SELECT id, username, email, display_name, password_hash, role,
			oauth_provider, oauth_id, created_at, last_login
		FROM users WHERE username = ?`, username).Scan(
		&user.ID, &user.Username, &user.Email, &user.DisplayName,
		&hash, &user.Role, &user.OAuthProvider, &user.OAuthID,
		&user.CreatedAt, &user.LastLogin)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("invalid username or password")
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return nil, fmt.Errorf("invalid username or password")
	}

	// Update last login
	now := time.Now().Format(time.RFC3339)
	jq.db.Exec("UPDATE users SET last_login = ? WHERE id = ?", now, user.ID)
	user.LastLogin = now

	return &user, nil
}

// FindOrCreateOAuthUser finds a user by OAuth provider+ID, or creates one.
func (jq *JobQueue) FindOrCreateOAuthUser(provider, oauthID, email, displayName string) (*User, error) {
	var user User
	err := jq.db.QueryRow(`
		SELECT id, username, email, display_name, role, oauth_provider, oauth_id, created_at
		FROM users WHERE oauth_provider = ? AND oauth_id = ?`,
		provider, oauthID).Scan(
		&user.ID, &user.Username, &user.Email, &user.DisplayName,
		&user.Role, &user.OAuthProvider, &user.OAuthID, &user.CreatedAt)

	if err == nil {
		// Existing user — update last login
		now := time.Now().Format(time.RFC3339)
		jq.db.Exec("UPDATE users SET last_login = ?, email = ?, display_name = ? WHERE id = ?",
			now, email, displayName, user.ID)
		user.LastLogin = now
		return &user, nil
	}

	if err != sql.ErrNoRows {
		return nil, err
	}

	// Create new user from OAuth
	username := email
	if username == "" {
		username = provider + "-" + oauthID
	}

	id := generateID()
	now := time.Now().Format(time.RFC3339)

	_, err = jq.db.Exec(`
		INSERT INTO users (id, username, email, display_name, role, oauth_provider, oauth_id, created_at, last_login)
		VALUES (?, ?, ?, ?, 'member', ?, ?, ?, ?)`,
		id, username, email, displayName, provider, oauthID, now, now)
	if err != nil {
		return nil, fmt.Errorf("creating OAuth user: %w", err)
	}

	return &User{
		ID: id, Username: username, Email: email, DisplayName: displayName,
		Role: "member", OAuthProvider: provider, OAuthID: oauthID, CreatedAt: now,
	}, nil
}

// GetUser retrieves a user by ID.
func (jq *JobQueue) GetUser(id string) (*User, error) {
	var user User
	err := jq.db.QueryRow(`
		SELECT id, username, email, display_name, role, oauth_provider, oauth_id, created_at, last_login
		FROM users WHERE id = ?`, id).Scan(
		&user.ID, &user.Username, &user.Email, &user.DisplayName,
		&user.Role, &user.OAuthProvider, &user.OAuthID,
		&user.CreatedAt, &user.LastLogin)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}
	return &user, nil
}

// ListUsers returns all users.
func (jq *JobQueue) ListUsers() ([]*User, error) {
	rows, err := jq.db.Query(`
		SELECT id, username, email, display_name, role, oauth_provider, oauth_id, created_at, last_login
		FROM users ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		var u User
		rows.Scan(&u.ID, &u.Username, &u.Email, &u.DisplayName,
			&u.Role, &u.OAuthProvider, &u.OAuthID, &u.CreatedAt, &u.LastLogin)
		users = append(users, &u)
	}
	return users, nil
}

// DeleteUser removes a user by username.
func (jq *JobQueue) DeleteUser(username string) error {
	result, err := jq.db.Exec("DELETE FROM users WHERE username = ?", username)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("user %q not found", username)
	}
	return nil
}

// UpdatePassword changes a user's password.
func (jq *JobQueue) UpdatePassword(username, newPassword string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	result, err := jq.db.Exec("UPDATE users SET password_hash = ? WHERE username = ?", string(hash), username)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("user %q not found", username)
	}
	return nil
}

// UserCount returns the number of users.
func (jq *JobQueue) UserCount() int {
	var count int
	jq.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	return count
}

func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
