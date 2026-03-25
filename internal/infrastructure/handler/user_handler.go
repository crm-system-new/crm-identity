package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/crm-system-new/crm-identity/internal/service"
	"github.com/crm-system-new/crm-shared/pkg/httputil"
)

type UserHandler struct {
	userService *service.UserService
}

func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	user, err := h.userService.GetUser(r.Context(), id)
	if err != nil {
		httputil.RespondError(w, err)
		return
	}

	httputil.RespondJSON(w, http.StatusOK, user)
}

func (h *UserHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req service.UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondJSON(w, http.StatusBadRequest, httputil.ErrorResponse{Error: "invalid request body"})
		return
	}

	user, err := h.userService.UpdateProfile(r.Context(), id, req)
	if err != nil {
		httputil.RespondError(w, err)
		return
	}

	httputil.RespondJSON(w, http.StatusOK, user)
}

func (h *UserHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req service.ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondJSON(w, http.StatusBadRequest, httputil.ErrorResponse{Error: "invalid request body"})
		return
	}

	if err := h.userService.ChangePassword(r.Context(), id, req); err != nil {
		httputil.RespondError(w, err)
		return
	}

	httputil.RespondNoContent(w)
}

func (h *UserHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	p := httputil.ParsePagination(r)

	users, total, err := h.userService.ListUsers(r.Context(), p.Limit, p.Offset)
	if err != nil {
		httputil.RespondError(w, err)
		return
	}

	httputil.RespondJSON(w, http.StatusOK, httputil.ListResponse[*service.UserResponse]{
		Items:  users,
		Total:  total,
		Limit:  p.Limit,
		Offset: p.Offset,
	})
}
