package handlers

import (
	"net/http"
	"time"

	"rest-api-scheduler-task/internal/api/middleware"
	"rest-api-scheduler-task/internal/api/service"
)

// NextDate public handler returns date in this format -> 20060102
func (h *Handler) NextDate(w http.ResponseWriter, r *http.Request) {
	log := middleware.L(r.Context()).With("handler", "NextDate")

	nowStr := r.URL.Query().Get("now")
	dateStr := r.URL.Query().Get("date")
	repeat := r.URL.Query().Get("repeat")

	log.Debug("nextdate request",
		"now", nowStr,
		"date", dateStr,
		"repeat", repeat)

	now, err := time.Parse("20060102", nowStr)
	if err != nil {
		log.Warn("invalid now", "now", nowStr, "err", err)
		http.Error(w, "invalid now", http.StatusBadRequest)
		return
	}

	next, err := service.NextDate(now, dateStr, repeat)
	if err != nil {
		log.Warn("next date failed", "date", dateStr, "repeat", repeat, "err", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Info("next date computed",
		"date", dateStr,
		"repeat", repeat,
		"result", next)

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(next))
}
