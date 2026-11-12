package handlers

import (
	"net/http"

	"rest-api-scheduler-task/internal/api/middleware"
)

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	log := middleware.L(r.Context()).With("handler", "Health")
	log.Info("health check")
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status": "ok"}`))
}
