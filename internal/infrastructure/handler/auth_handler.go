package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/crm-system-new/crm-identity/internal/domain"
	"github.com/crm-system-new/crm-identity/internal/service"
	"github.com/crm-system-new/crm-shared/pkg/httputil"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req service.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondJSON(w, http.StatusBadRequest, httputil.ErrorResponse{Error: "invalid request body"})
		return
	}

	resp, err := h.authService.Register(r.Context(), req)
	if err != nil {
		if errors.Is(err, domain.ErrEmailAlreadyExists) {
			httputil.RespondJSON(w, http.StatusConflict, httputil.ErrorResponse{Error: err.Error()})
			return
		}
		httputil.RespondError(w, err)
		return
	}

	httputil.RespondCreated(w, resp)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req service.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondJSON(w, http.StatusBadRequest, httputil.ErrorResponse{Error: "invalid request body"})
		return
	}

	tokenPair, err := h.authService.Login(r.Context(), req)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidCredentials) {
			httputil.RespondJSON(w, http.StatusUnauthorized, httputil.ErrorResponse{Error: err.Error()})
			return
		}
		httputil.RespondError(w, err)
		return
	}

	httputil.RespondJSON(w, http.StatusOK, tokenPair)
}

func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondJSON(w, http.StatusBadRequest, httputil.ErrorResponse{Error: "invalid request body"})
		return
	}

	tokenPair, err := h.authService.RefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		httputil.RespondJSON(w, http.StatusUnauthorized, httputil.ErrorResponse{Error: "invalid refresh token"})
		return
	}

	httputil.RespondJSON(w, http.StatusOK, tokenPair)
}
