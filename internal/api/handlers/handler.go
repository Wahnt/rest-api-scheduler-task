package handlers

import (
	"context"
	"log/slog"

	"rest-api-scheduler-task/internal/auth"
	"rest-api-scheduler-task/internal/db"
)

// Interface for job with task
type TaskStore interface {
	CreateTask(ctx context.Context, t db.Task) (int64, error)
	ListTasks(ctx context.Context, search string) ([]db.Task, error)
	GetTask(ctx context.Context, id int64) (db.Task, error)
	UpdateTask(ctx context.Context, t db.Task) error
	DeleteTask(ctx context.Context, id int64) error
}

type Handler struct {
	store  TaskStore
	logger *slog.Logger
	auth   *auth.Manager
}

func New(store TaskStore, logger *slog.Logger, authMgr *auth.Manager) *Handler {
	return &Handler{store: store, logger: logger.With("component", "handlers"), auth: authMgr}
}
