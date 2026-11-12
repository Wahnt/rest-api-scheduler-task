package db

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

type Store struct {
	db     *sql.DB
	logger *slog.Logger
}

// New Opens connection and check him, create table
func New(ctx context.Context, dsn string, logger *slog.Logger) (*Store, error) {
	logger = logger.With("component", "db")
	conn, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	if err := conn.PingContext(ctx); err != nil {
		conn.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	conn.SetMaxOpenConns(10)
	conn.SetMaxIdleConns(5)

	logger.Info("postgres connected")
	return &Store{db: conn, logger: logger}, nil
}

func (s *Store) Close() error {
	s.logger.Info("closing postgres connection")
	if err := s.db.Close(); err != nil {
		s.logger.Error("close postgres failed", "err", err)
		return err
	}
	return nil
}

func (s *Store) DB() *sql.DB {
	return s.db
}

func (s *Store) IsInitialized(ctx context.Context) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, `
				SELECT EXISTS (
					SELECT 1 FROM information_schema.tables
					WHERE table_schema = 'public' AND table_name = 'scheduler'
				)`,
	).Scan(&exists)
	if err != nil {
		s.logger.Error("check scheduler table failed", "err", err)
		return false, fmt.Errorf("check table: %w", err)
	}

	s.logger.Debug("scheduler table check", "exists", exists)
	return exists, err
}

// Init create a table if not exists
func (s *Store) Init(ctx context.Context) error {
	const ddlTable = `CREATE TABLE IF NOT EXISTS scheduler (
		id 		INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
		date	TIMESTAMPTZ NOT NULL DEFAULT now(),
		title	TEXT	NOT NULL CHECK (length(trim(title)) BETWEEN 1 AND 512),
		comment	TEXT	NOT NULL DEFAULT '',
		repeat	TEXT	NOT NULL DEFAULT '')`

	const ddlIndex = `CREATE INDEX IF NOT EXISTS idx_scheduler_date ON scheduler (date)`

	s.logger.Info("initializing schema")

	if _, err := s.db.ExecContext(ctx, ddlTable); err != nil {
		s.logger.Error("create table failed", "err", err)
		return fmt.Errorf("create table: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, ddlIndex); err != nil {
		s.logger.Error("create index failed", "err", err)
		return fmt.Errorf("create index: %w", err)
	}

	s.logger.Info("schema initialized")
	return nil
}

// Task method's

// CreateTask
func (s *Store) CreateTask(ctx context.Context, t Task) (int64, error) {
	var id int64
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO scheduler (date, title, comment, repeat)
			VALUES ($1, $2, $3, $4)
			RETURNING id`,
		t.Date, t.Title, t.Comment, t.Repeat,
	).Scan(&id)
	if err != nil {
		s.logger.Error("create task failed", "err", err)
	}
	s.logger.Info("task created", "id", id, "title", t.Title)
	return id, nil
}

// GetTask
func (s *Store) GetTask(ctx context.Context, id int64) (Task, error) {
	var t Task
	err := s.db.QueryRowContext(ctx,
		`SELECT id, date, title, comment, repeat
		FROM scheduler WHERE id = $1`, id,
	).Scan(&t.ID, &t.Date, &t.Title, &t.Comment, &t.Repeat)
	if err != nil {
		if err == sql.ErrNoRows {
			return Task{}, fmt.Errorf("task %d not found", id)
		}
		s.logger.Error("get task failed", "id", id, "err", err)
		return Task{}, fmt.Errorf("get task: %w", err)
	}
	return t, nil
}

// ListTasks
func (s *Store) ListTasks(ctx context.Context, search string) ([]Task, error) {
	query := `SELECT id, date, title, comment, repeat FROM scheduler`
	args := []any{}
	where := []string{}

	if search != "" {
		if d, err := time.Parse("02.01.2006", search); err != nil {
			where = append(where, "date::date = $1")
			args = append(args, d)
		} else {
			where = append(where, "(title ILIKE $1 OR comment !LIKE $1)")
			args = append(args, "%"+search+"%")
		}
	}
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}

	query += " ORDER BY date, id"

	rows, err := s.db.QueryContext(ctx,
		`SELECT id, date, title, comment, repeat
		FROM scheduler ORDER BY date, id`)
	if err != nil {
		s.logger.Error("list tasks failed", "err", err)
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()

	tasks := make([]Task, 0)
	for rows.Next() {
		var t Task
		if err := rows.Scan(&t.ID, &t.Date, &t.Title, &t.Comment, &t.Repeat); err != nil {
			s.logger.Error("scan task failed", "err", err)
			return nil, fmt.Errorf("scan task: %w", err)
		}
		tasks = append(tasks, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}
	return tasks, nil
}

// UpdateTask
func (s *Store) UpdateTask(ctx context.Context, t Task) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE scheduler Set date = $1, title = $2, comment = $3, repeat = $4
		WHERE id = $5`,
		t.Date, t.Title, t.Comment, t.Repeat, t.ID,
	)
	if err != nil {
		s.logger.Error("update task failed", "id", t.ID, "err", err)
		return fmt.Errorf("update task: %w", err)
	}
	return nil
}

// DeleteTask
func (s *Store) DeleteTask(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM scheduler WHERE id = $1`, id)
	if err != nil {
		s.logger.Error("delete task failed", "id", id, "err", err)
		return fmt.Errorf("delete task: %w", err)
	}
	return nil
}
