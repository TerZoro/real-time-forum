package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"real-time-forum/database"
	"real-time-forum/models"

	"github.com/gofrs/uuid"
	"golang.org/x/crypto/bcrypt"
)

type registerRequest struct {
	Nickname  string `json:"nickname"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Age       int    `json:"age"`
	Gender    string `json:"gender"`
	Password  string `json:"password"`
}

func Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	req.Nickname = strings.TrimSpace(req.Nickname)
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.FirstName = strings.TrimSpace(req.FirstName)
	req.LastName = strings.TrimSpace(req.LastName)

	if req.Nickname == "" || req.Email == "" || req.FirstName == "" ||
		req.LastName == "" || req.Password == "" || req.Age <= 0 || req.Gender == "" {
		jsonError(w, "all fields are required", http.StatusBadRequest)
		return
	}

	if len(req.Password) > 72 {
		jsonError(w, "password must be 72 characters or less", http.StatusBadRequest)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	id, _ := uuid.NewV4()
	_, err = database.DB.Exec(
		`INSERT INTO users (id, nickname, first_name, last_name, email, age, gender, password)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id.String(), req.Nickname, req.FirstName, req.LastName,
		req.Email, req.Age, req.Gender, string(hash),
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			jsonError(w, "nickname or email already in use", http.StatusConflict)
		} else {
			jsonError(w, "internal error", http.StatusInternalServerError)
		}
		return
	}

	// auto login after registration
	sessionID := createSession(id.String())
	setSessionCookie(w, sessionID)

	user := models.User{
		ID:        id.String(),
		Nickname:  req.Nickname,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Email:     req.Email,
		Age:       req.Age,
		Gender:    req.Gender,
		CreatedAt: time.Now(),
	}
	jsonOK(w, user)
}

type loginRequest struct {
	Identifier string `json:"identifier"` // nickname or email
	Password   string `json:"password"`
}

func Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	identifier := strings.TrimSpace(req.Identifier)
	if identifier == "" || req.Password == "" {
		jsonError(w, "identifier and password required", http.StatusBadRequest)
		return
	}

	var user models.User
	err := database.DB.QueryRow(
		`SELECT id, nickname, first_name, last_name, email, age, gender, password, created_at
		 FROM users WHERE nickname = ? OR email = ?`,
		identifier, strings.ToLower(identifier),
	).Scan(&user.ID, &user.Nickname, &user.FirstName, &user.LastName,
		&user.Email, &user.Age, &user.Gender, &user.Password, &user.CreatedAt)

	if err != nil {
		jsonError(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	if err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		jsonError(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	sessionID := createSession(user.ID)
	setSessionCookie(w, sessionID)

	user.Password = ""
	jsonOK(w, user)
}

func Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cookie, err := r.Cookie("session_id")
	if err == nil {
		database.DB.Exec(`DELETE FROM sessions WHERE id = ?`, cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
	jsonOK(w, map[string]string{"message": "logged out"})
}

// Me - returns current user from session

func Me(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	user, ok := GetSessionUser(r)
	if !ok {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	jsonOK(w, user)
}

func createSession(userID string) string {
	id, _ := uuid.NewV4()
	database.DB.Exec(`INSERT INTO sessions (id, user_id) VALUES (?, ?)`, id.String(), userID)
	return id.String()
}

func setSessionCookie(w http.ResponseWriter, sessionID string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Path:     "/",
		MaxAge:   86400,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// reads from session cookie and returns the authorized user
func GetSessionUser(r *http.Request) (models.User, bool) {
	cookie, err := r.Cookie("session_id")
	if err != nil {
		return models.User{}, false
	}

	var user models.User
	err = database.DB.QueryRow(
		`SELECT u.id, u.nickname, u.first_name, u.last_name, u.email, u.age, u.gender, u.created_at
		 FROM sessions s JOIN users u ON s.user_id = u.id
		 WHERE s.id = ?`,
		cookie.Value,
	).Scan(&user.ID, &user.Nickname, &user.FirstName, &user.LastName,
		&user.Email, &user.Age, &user.Gender, &user.CreatedAt)

	if err != nil {
		return models.User{}, false
	}
	return user, true
}

func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := GetSessionUser(r); !ok {
			jsonError(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func jsonOK(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
