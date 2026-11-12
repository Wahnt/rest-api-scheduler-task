package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"rest-api-scheduler-task/internal/api/middleware"
	"rest-api-scheduler-task/internal/api/service"
	"rest-api-scheduler-task/internal/db"
)

type TaskRequest struct {
	Date    string `json:"date"`
	Title   string `json:"title"`
	Comment string `json:"comment"`
	Repeat  string `json:"repeat"`
}

type TaskUpdateRequest struct {
	ID      int64  `json:"id"`
	Date    string `json:"date"`
	Title   string `json:"title"`
	Comment string `json:"comment"`
	Repeat  string `json:"repeat"`
}

type TaskResponse struct {
	ID int64 `json:"id"`
}

type TaskItem struct {
	ID      int64  `json:"id"`
	Date    string `json:"date"`
	Title   string `json:"title"`
	Comment string `json:"comment"`
	Repeat  string `json:"repeat"`
}

// CreateTasl validates the request, computes the date and inserts the task
func (h *Handler) CreateTask(w http.ResponseWriter, r *http.Request) {
	log := middleware.L(r.Context()).With("handler", "CreateTask")

	var req TaskRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Warn("decode task request failed", "err", err)
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		log.Warn("empty title")
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	date := today

	if req.Date != "" {
		d, err := time.Parse("20060102", req.Date)
		if err != nil {
			log.Warn("invalid date", "date", req.Date, "err", err)
			writeError(w, http.StatusBadRequest, "invalid date, expected 20060102")
			return
		}
		date = d
	}

	if req.Repeat != "" {
		next, err := service.NextDate(now, date.Format("20060102"), req.Repeat)
		if err != nil {
			log.Warn("invalid repeat", "repeat", req.Repeat, "err", err)
			writeError(w, http.StatusBadRequest, "invalid repeat: "+err.Error())
			return
		}

		if date.Before(today) {
			d, err := time.Parse("20060102", next)
			if err != nil {
				log.Error("parse next date failed", "next", next, "err", err)
				writeError(w, http.StatusInternalServerError, "internal error")
				return
			}
			date = d
		}
	} else if date.Before(today) {
		date = today
	}
	id, err := h.store.CreateTask(r.Context(), db.Task{
		Date:    date,
		Title:   req.Title,
		Comment: req.Comment,
		Repeat:  req.Repeat,
	})
	if err != nil {
		log.Error("create task failed", "err", err)
		writeError(w, http.StatusBadRequest, "db error")
		return
	}

	log.Info("task created", "id", id, "title", req.Title, "date", date, "repeat", req.Repeat)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(TaskResponse{ID: id})
}

// ListTasks returns a list of tasks with search
func (h *Handler) ListTasks(w http.ResponseWriter, r *http.Request) {
	log := middleware.L(r.Context()).With("handler", "ListTasks")

	search := strings.TrimSpace(r.URL.Query().Get("search"))

	tasks, err := h.store.ListTasks(r.Context(), search)
	if err != nil {
		log.Error("list tasks failed", "err", err, "search", search)
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	items := make([]TaskItem, 0, len(tasks))
	for _, t := range tasks {
		items = append(items, TaskItem{
			ID:      t.ID,
			Date:    t.Date.Format("20060102"),
			Title:   t.Title,
			Comment: t.Comment,
			Repeat:  t.Repeat,
		})
	}

	log.Info("task listed", "count", len(items))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"tasks": items})
}

// GetTask returns a task by ID or an ampty task
func (h *Handler) GetTask(w http.ResponseWriter, r *http.Request) {
	log := middleware.L(r.Context()).With("handler", "GetTask")

	idStr := r.URL.Query().Get("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		log.Warn("invalid id", "id", idStr, "err", err)
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	task, err := h.store.GetTask(r.Context(), id)
	if err != nil {
		log.Warn("get task failed", "id", id, "err", err)
		writeError(w, http.StatusBadRequest, "task not found")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(TaskItem{
		ID:      task.ID,
		Date:    task.Date.Format("20060102"),
		Title:   task.Title,
		Comment: task.Comment,
		Repeat:  task.Repeat,
	})
}

// Updates a task
func (h *Handler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	log := middleware.L(r.Context()).With("handler", "UpdateTask")

	var req TaskUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Warn("decode task request failed", "err", err)
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if req.ID == 0 {
		log.Warn("empty id")
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}
	id := req.ID

	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		log.Warn("empty title")
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	date := today
	if req.Date != "" {
		d, err := time.Parse("20060102", req.Date)
		if err != nil {
			log.Warn("invalid date", "date", req.Date, "err", err)
			writeError(w, http.StatusBadRequest, "invalid date, expected 20060102")
			return
		}
		date = d
	}

	if req.Repeat != "" {
		next, err := service.NextDate(now, date.Format("20060102"), req.Repeat)
		if err != nil {
			log.Warn("invalid repeat", "repeat", req.Repeat, "err", err)
			writeError(w, http.StatusBadRequest, "invalid repeat: "+err.Error())
			return
		}
		if date.Before(today) {
			d, err := time.Parse("20060102", next)
			if err != nil {
				log.Error("parse next date failed", "next", next, "err", err)
				writeError(w, http.StatusInternalServerError, "internal error")
				return
			}
			date = d
		}
	} else if date.Before(today) {
		date = today
	}

	err := h.store.UpdateTask(r.Context(), db.Task{
		ID:      id,
		Date:    date,
		Title:   req.Title,
		Comment: req.Comment,
		Repeat:  req.Repeat,
	})
	if err != nil {
		log.Error("update task failed", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	log.Info("task updated", "id", id, "title", req.Title, "date", date.Format("20060102"))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(TaskResponse{ID: id})
}

// Deletes a task by ID
func (h *Handler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	log := middleware.L(r.Context()).With("handler", "DeleteTask")

	idStr := r.URL.Query().Get("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		log.Warn("invalid id", "id", idStr, "err", err)
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	if err := h.store.DeleteTask(r.Context(), id); err != nil {
		log.Error("delete task failed", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	log.Info("task deleted", "id", id)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{})
}

// Marks a task as done: deletes it or reschedules to the next date
func (h *Handler) DoneTask(w http.ResponseWriter, r *http.Request) {
	log := middleware.L(r.Context()).With("handler", "DoneTask")

	idStr := r.URL.Query().Get("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		log.Warn("invalid id", "id", idStr, "err", err)
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	task, err := h.store.GetTask(r.Context(), id)
	if err != nil {
		log.Warn("get task failed", "id", id, "err", err)
		writeError(w, http.StatusNotFound, "task not found")
		return
	}

	if task.Repeat == "" {
		if err := h.store.DeleteTask(r.Context(), id); err != nil {
			log.Error("delete task failed", "id", id, "err", err)
			writeError(w, http.StatusInternalServerError, "db error")
			return
		}

		log.Info("task done, deleted", "id", id)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{})
		return
	}

	now := time.Now()
	next, err := service.NextDate(now, task.Date.Format("20060102"), task.Repeat)
	if err != nil {
		log.Error("next date failed", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "next date error")
		return
	}

	d, err := time.Parse("20060102", next)
	if err != nil {
		log.Error("parse netx date failed", "next", next, "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	task.Date = d
	if err := h.store.UpdateTask(r.Context(), task); err != nil {
		log.Error("update task failed", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	log.Info("task rescheduled", "id", id, "next", next)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{})
}

// writeError error helper
func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
