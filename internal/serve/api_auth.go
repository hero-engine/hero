package serve

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
)

// RegisterAuthAPI adds authentication endpoints.
func RegisterAuthAPI(mux *http.ServeMux, jq *JobQueue, jwtSecret string) {
	mux.HandleFunc("/auth/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		handleLogin(w, r, jq, jwtSecret)
	})

	mux.HandleFunc("/auth/me", func(w http.ResponseWriter, r *http.Request) {
		handleMe(w, r, jq, jwtSecret)
	})

	mux.HandleFunc("/api/users", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			handleListUsers(w, r, jq, jwtSecret)
		case "POST":
			handleCreateUser(w, r, jq, jwtSecret)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/users/", func(w http.ResponseWriter, r *http.Request) {
		username := strings.TrimPrefix(r.URL.Path, "/api/users/")
		if username == "" {
			http.Error(w, "username required", http.StatusBadRequest)
			return
		}
		if r.Method == "DELETE" {
			handleDeleteUser(w, r, jq, jwtSecret, username)
		} else {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
}

func handleLogin(w http.ResponseWriter, r *http.Request, jq *JobQueue, jwtSecret string) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	user, err := jq.AuthenticateUser(req.Username, req.Password)
	if err != nil {
		jsonError(w, err.Error(), http.StatusUnauthorized)
		return
	}

	token, err := IssueJWT(user, jwtSecret, 0)
	if err != nil {
		jsonError(w, "failed to issue token", http.StatusInternalServerError)
		return
	}

	jsonResponse(w, map[string]interface{}{
		"token": token,
		"user":  user,
	})
}

func handleMe(w http.ResponseWriter, r *http.Request, jq *JobQueue, jwtSecret string) {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		jsonError(w, "authorization required", http.StatusUnauthorized)
		return
	}

	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		jsonError(w, "invalid auth format", http.StatusUnauthorized)
		return
	}

	claims, err := ValidateJWT(parts[1], jwtSecret)
	if err != nil {
		jsonError(w, err.Error(), http.StatusUnauthorized)
		return
	}

	user, err := jq.GetUser(claims.Sub)
	if err != nil {
		jsonResponse(w, claims)
		return
	}

	jsonResponse(w, user)
}

func handleListUsers(w http.ResponseWriter, r *http.Request, jq *JobQueue, jwtSecret string) {
	if !requireAdmin(w, r, jwtSecret) {
		return
	}

	users, err := jq.ListUsers()
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if users == nil {
		users = []*User{}
	}
	jsonResponse(w, users)
}

func handleCreateUser(w http.ResponseWriter, r *http.Request, jq *JobQueue, jwtSecret string) {
	// Allow first user without auth (bootstrap), require admin after that
	if jq.UserCount() > 0 && !requireAdmin(w, r, jwtSecret) {
		return
	}

	var req struct {
		Username    string `json:"username"`
		Email       string `json:"email"`
		DisplayName string `json:"display_name"`
		Password    string `json:"password"`
		Role        string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	// First user is always admin
	if jq.UserCount() == 0 {
		req.Role = "admin"
	}

	user, err := jq.CreateUser(req.Username, req.Email, req.DisplayName, req.Password, req.Role)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
	jsonResponse(w, user)
}

func handleDeleteUser(w http.ResponseWriter, r *http.Request, jq *JobQueue, jwtSecret, username string) {
	if !requireAdmin(w, r, jwtSecret) {
		return
	}

	if err := jq.DeleteUser(username); err != nil {
		jsonError(w, err.Error(), http.StatusNotFound)
		return
	}
	jsonResponse(w, map[string]string{"status": "deleted", "username": username})
}

func requireAdmin(w http.ResponseWriter, r *http.Request, jwtSecret string) bool {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		// Also check static token from env (admin bootstrap)
		if token := os.Getenv("HERO_AUTH_TOKEN"); token != "" {
			parts := strings.SplitN(auth, " ", 2)
			if len(parts) == 2 && parts[1] == token {
				return true
			}
		}
		jsonError(w, "admin authorization required", http.StatusUnauthorized)
		return false
	}

	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 {
		jsonError(w, "invalid auth format", http.StatusUnauthorized)
		return false
	}

	// Static token = admin
	if staticToken := os.Getenv("HERO_AUTH_TOKEN"); staticToken != "" && parts[1] == staticToken {
		return true
	}

	// JWT — check role
	claims, err := ValidateJWT(parts[1], jwtSecret)
	if err != nil {
		jsonError(w, err.Error(), http.StatusUnauthorized)
		return false
	}
	if claims.Role != "admin" {
		jsonError(w, "admin role required", http.StatusForbidden)
		return false
	}
	return true
}
