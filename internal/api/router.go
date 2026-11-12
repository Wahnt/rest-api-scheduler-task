package api

import (
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"rest-api-scheduler-task/internal/api/handlers"
	"rest-api-scheduler-task/internal/api/middleware"
	"rest-api-scheduler-task/internal/auth"
	"rest-api-scheduler-task/internal/db"

	"github.com/go-chi/chi/v5"
)

// NewRouter assembles chi-router and wraps it in Logging
func NewRouter(logger *slog.Logger, staticDir string, store *db.Store, authMgr *auth.Manager) http.Handler {
	r := chi.NewRouter()
	// API
	h := handlers.New(store, logger, authMgr)
	r.Route("/api", func(r chi.Router) {
		// public
		r.Get("/health", h.Health)
		r.Get("/nextdate", h.NextDate)
		r.Post("/signin", h.Signin)
		// CRUD protected
		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(authMgr))

			r.Post("/task", h.CreateTask)
			r.Get("/tasks", h.ListTasks)
			r.Get("/task", h.GetTask)
			r.Put("/task", h.UpdateTask)
			r.Delete("/task", h.DeleteTask)
			r.Post("/task/done", h.DoneTask)
		})

		r.NotFound(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error": "not found"}`))
		})
	})
	// Static
	abs, err := filepath.Abs(staticDir)
	if err != nil {
		logger.Error("static dir abs failed", "err", err, "static", staticDir)
		abs = staticDir
	} else {
		logger.Info("static dir resolved", "abs", abs)
	}
	if _, err := os.Stat(filepath.Join(abs, "css", "style.css")); err != nil {
		logger.Info("css not found", "abs", abs, "err", err)
	}

	r.Handle("/*", http.StripPrefix("/", http.FileServer(http.Dir(abs))))
	return middleware.Logging(logger, r)
}
