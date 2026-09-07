package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"ecommerce-backend/internal/domain"
	"ecommerce-backend/internal/service"
)

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req domain.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.authService.Register(r.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrUserAlreadyExists) {
			respondError(w, http.StatusConflict, "user with this email already exists")
			return
		}
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, resp)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req domain.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.authService.Login(r.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			respondError(w, http.StatusUnauthorized, "invalid email or password")
			return
		}
		respondError(w, http.StatusInternalServerError, "login failed")
		return
	}

	respondJSON(w, http.StatusOK, resp)
}

func (h *Handler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req domain.RefreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.authService.RefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "invalid or expired refresh token")
		return
	}

	respondJSON(w, http.StatusOK, resp)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	var req domain.RefreshTokenRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	if req.RefreshToken != "" {
		_ = h.authService.Logout(r.Context(), req.RefreshToken)
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "logged out successfully"})
}

func (h *Handler) GetMe(w http.ResponseWriter, r *http.Request) {
	claims := GetUserClaims(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	user, err := h.authService.GetUserByID(r.Context(), claims.UserID)
	if err != nil || user == nil {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"id":    claims.UserID,
			"email": claims.Email,
			"role":  claims.Role,
		})
		return
	}

	respondJSON(w, http.StatusOK, user.ToResponse())
}

func (h *Handler) GetAdminStats(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "admin access granted",
		"cluster": "healthy",
	})
}
