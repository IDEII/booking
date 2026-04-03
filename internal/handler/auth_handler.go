package handler

import (
	"encoding/json"
	"net/http"

	"booking-service/internal/domain"
	"booking-service/internal/service"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

type DummyLoginRequest struct {
	Role domain.UserRole `json:"role"`
}

func (h *AuthHandler) DummyLogin(w http.ResponseWriter, r *http.Request) {
	var req DummyLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		domain.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	if req.Role != domain.RoleAdmin && req.Role != domain.RoleUser {
		domain.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid role")
		return
	}

	token, err := h.authService.DummyLogin(r.Context(), req.Role)
	if err != nil {
		domain.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	domain.WriteJSON(w, http.StatusOK, map[string]string{"token": token})
}

type RegisterRequest struct {
	Email    string          `json:"email"`
	Password string          `json:"password"`
	Role     domain.UserRole `json:"role"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		domain.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	user, err := h.authService.Register(r.Context(), req.Email, req.Password, req.Role)
	if err != nil {
		domain.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	domain.WriteJSON(w, http.StatusCreated, map[string]interface{}{"user": user})
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		domain.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	token, err := h.authService.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		domain.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", err.Error())
		return
	}

	domain.WriteJSON(w, http.StatusOK, map[string]string{"token": token})
}
