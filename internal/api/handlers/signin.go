package handlers

import (
	"encoding/json"
	"net/http"

	"rest-api-scheduler-task/internal/api/middleware"
)

type SigninRequest struct {
	Password string `json:"password"`
}

type SigninResponse struct {
	Token string `json:"token"`
}

// Signin check password and returns JWT-token
func (h *Handler) Signin(w http.ResponseWriter, r *http.Request) {
	log := middleware.L(r.Context()).With("handler", "Signin")

	var req SigninRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Warn("decode sigin request failed", "err", err)
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if req.Password == "" {
		log.Warn("empty password")
		writeError(w, http.StatusBadRequest, "password is required")
		return
	}

	token, err := h.auth.Login(req.Password)
	if err != nil {
		log.Warn("login failed", "err", err)
		writeError(w, http.StatusUnauthorized, "invalid password")
		return
	}

	log.Info("sigin success")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(SigninResponse{Token: token})
}
