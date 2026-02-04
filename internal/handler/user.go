package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Adopten123/go-messenger/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/redis/go-redis/v9"
)

type UserHandler struct {
	service     *service.UserService
	tokenSecret string
	rdb         *redis.Client
	fileService *service.FileService
}

func NewUserHandler(
	service *service.UserService,
	tokenSecret string,
	rdb *redis.Client,
	fs *service.FileService) *UserHandler {

	return &UserHandler{
		service:     service,
		tokenSecret: tokenSecret,
		rdb:         rdb,
		fileService: fs,
	}
}

// RegisterRequest - Struct of JSON-request
type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// RegisterResponse - Struct of JSON-response
type RegisterResponse struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	// 1. Decoding JSON from the request-body
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// 2. Validation
	if req.Email == "" || req.Password == "" || req.Username == "" {
		http.Error(w, "all fields are required", http.StatusBadRequest)
		return
	}

	// 3. Calling a service
	user, err := h.service.CreateUser(r.Context(), req.Email, req.Username, req.Password)
	if err != nil {
		fmt.Printf("FAILED TO CREATE USER: %v\n", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// 4. Making response
	resp := RegisterResponse{
		ID:       user.ID.String(),
		Username: user.Username,
		Email:    user.Email,
	}

	// 5. Sending response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated) // 201 Created
	json.NewEncoder(w).Encode(resp)
}

func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	token, err := h.service.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		http.Error(w, "invalid email or password", http.StatusUnauthorized)
		return
	}

	resp := LoginResponse{Token: token}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// GetMe - test method, returns the current user ID
func (h *UserHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	// Getting the ID from the context (where the Middleware put it)
	userID := r.Context().Value(UserIDKey).(string)

	w.Write([]byte("You are logged in as user ID: " + userID))
}

func (h *UserHandler) GetOnlineStatus(w http.ResponseWriter, r *http.Request) {
	targetID := chi.URLParam(r, "user_id")
	if targetID == "" {
		http.Error(w, "user_id is required", http.StatusBadRequest)
		return
	}

	exists, err := h.rdb.Exists(r.Context(), "user:"+targetID+":online").Result()
	if err != nil {
		http.Error(w, "redis error", http.StatusInternalServerError)
		return
	}

	status := map[string]bool{
		"online": exists > 0,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (h *UserHandler) UploadAvatar(w http.ResponseWriter, r *http.Request) {
	// 1. Parse multipart/form-data (max 10mb)
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		http.Error(w, "file very big", http.StatusBadRequest)
		return
	}

	// 2. Get file from form
	file, header, err := r.FormFile("avatar")
	if err != nil {
		http.Error(w, "invalid file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// 3. Upload in MinIO
	url, err := h.fileService.UploadFile(
		r.Context(), file,
		header.Size, header.Filename, header.Header.Get("Content-Type"),
	)
	if err != nil {
		http.Error(w, "failed to upload", http.StatusInternalServerError)
		return
	}

	// 4. Update user in DB
	userID := r.Context().Value(UserIDKey).(string)

	var userUUID pgtype.UUID
	userUUID.Scan(userID)

	err = h.service.UpdateAvatar(r.Context(), userUUID, url)
	if err != nil {
		http.Error(w, "failed to update user profile", http.StatusInternalServerError)
		return
	}

	// 5. Url response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"avatar_url": url})
}
