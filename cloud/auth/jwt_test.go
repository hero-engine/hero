package auth

import (
	"testing"
	"time"
)

func TestIssueAndValidate(t *testing.T) {
	iss := NewIssuer("test-secret-key-32bytes-long!!", 1*time.Hour, 30*24*time.Hour)

	claims := Claims{
		UserID: "user-123",
		Email:  "test@example.com",
		Name:   "Test User",
		OrgID:  "org-456",
		Roles:  []string{"admin"},
	}

	token, err := iss.Issue(claims)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	got, err := iss.Validate(token)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}

	if got.UserID != "user-123" {
		t.Errorf("UserID = %q, want %q", got.UserID, "user-123")
	}
	if got.Email != "test@example.com" {
		t.Errorf("Email = %q, want %q", got.Email, "test@example.com")
	}
	if got.OrgID != "org-456" {
		t.Errorf("OrgID = %q, want %q", got.OrgID, "org-456")
	}
	if len(got.Roles) != 1 || got.Roles[0] != "admin" {
		t.Errorf("Roles = %v, want [admin]", got.Roles)
	}
	if got.Iat == 0 {
		t.Error("Iat should be set")
	}
	if got.Exp == 0 {
		t.Error("Exp should be set")
	}
}

func TestExpiredToken(t *testing.T) {
	iss := NewIssuer("test-secret", 1*time.Millisecond, 30*24*time.Hour)

	claims := Claims{
		UserID: "user-123",
		Exp:    time.Now().Add(-1 * time.Hour).Unix(),
	}

	token, err := iss.Issue(claims)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	_, err = iss.Validate(token)
	if err != ErrTokenExpired {
		t.Errorf("expected ErrTokenExpired, got %v", err)
	}
}

func TestInvalidSignature(t *testing.T) {
	iss1 := NewIssuer("secret-one", 1*time.Hour, 30*24*time.Hour)
	iss2 := NewIssuer("secret-two", 1*time.Hour, 30*24*time.Hour)

	token, err := iss1.Issue(Claims{UserID: "user-123"})
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	_, err = iss2.Validate(token)
	if err != ErrTokenInvalid {
		t.Errorf("expected ErrTokenInvalid, got %v", err)
	}
}

func TestMalformedToken(t *testing.T) {
	iss := NewIssuer("secret", 1*time.Hour, 30*24*time.Hour)

	cases := []string{
		"",
		"not-a-jwt",
		"two.parts",
		"three.parts.!!!invalid-base64!!!",
	}

	for _, tc := range cases {
		_, err := iss.Validate(tc)
		if err == nil {
			t.Errorf("expected error for token %q", tc)
		}
	}
}

func TestGenerateRefreshToken(t *testing.T) {
	t1, err := GenerateRefreshToken()
	if err != nil {
		t.Fatalf("GenerateRefreshToken: %v", err)
	}
	t2, err := GenerateRefreshToken()
	if err != nil {
		t.Fatalf("GenerateRefreshToken: %v", err)
	}

	if t1 == t2 {
		t.Error("expected unique tokens")
	}
	if len(t1) < 20 {
		t.Errorf("token too short: %d", len(t1))
	}
}

func TestHashToken(t *testing.T) {
	h1 := HashToken("token-a")
	h2 := HashToken("token-b")
	h3 := HashToken("token-a")

	if h1 == h2 {
		t.Error("different tokens should produce different hashes")
	}
	if h1 != h3 {
		t.Error("same token should produce same hash")
	}
	if len(h1) != 64 {
		t.Errorf("hash length = %d, want 64 (hex SHA-256)", len(h1))
	}
}
