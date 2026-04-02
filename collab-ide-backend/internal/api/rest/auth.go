package rest

import (
	"encoding/json"
	"log"
	"net/http"

	"collab-ide-backend/internal/auth"
	"collab-ide-backend/internal/repository"
)

type AuthHandler struct {
	userRepo *repository.UserRepo
}

func NewAuthHandler(userRepo *repository.UserRepo) *AuthHandler {
	return &AuthHandler{userRepo: userRepo}
}

type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Register - упрощенная версия без проверки существования
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Fail(400, "Неверный формат запроса"))
		return
	}

	if req.Username == "" || req.Password == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Fail(400, "Имя пользователя и пароль обязательны"))
		return
	}

	log.Printf("Register attempt for user: %s", req.Username)

	// Прямое создание пользователя
	userID, err := h.userRepo.CreateUser(r.Context(), req.Username, req.Password)
	if err != nil {
		log.Printf("Register error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Fail(500, "Не удалось создать пользователя: "+err.Error()))
		return
	}

	log.Printf("User created successfully: %s", userID)

	// Генерируем JWT токен
	token, err := auth.GenerateToken(userID.String(), req.Username)
	if err != nil {
		log.Printf("Token generation error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Fail(500, "Ошибка генерации токена"))
		return
	}

	json.NewEncoder(w).Encode(Ok(map[string]interface{}{
		"token":    token,
		"user_id":  userID,
		"username": req.Username,
	}))
}

// Login - оставляем без изменений
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Fail(400, "Неверный формат запроса"))
		return
	}

	if req.Username == "" || req.Password == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Fail(400, "Имя пользователя и пароль обязательны"))
		return
	}

	userID, err := h.userRepo.Authenticate(r.Context(), req.Username, req.Password)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Fail(500, "Ошибка аутентификации"))
		return
	}

	if userID == nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(Fail(401, "Неверное имя пользователя или пароль"))
		return
	}

	token, err := auth.GenerateToken(userID.String(), req.Username)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Fail(500, "Ошибка генерации токена"))
		return
	}

	json.NewEncoder(w).Encode(Ok(map[string]interface{}{
		"token":    token,
		"user_id":  userID,
		"username": req.Username,
	}))
}
