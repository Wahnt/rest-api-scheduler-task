package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

// ── DB ──

type Task struct {
	ID      int64  `db:"id"`
	Date    string `db:"date"`
	Title   string `db:"title"`
	Comment string `db:"comment"`
	Repeat  string `db:"repeat"`
}

func openDB(t *testing.T) *sqlx.DB {
	dsn := DBDSN
	if envDSN := os.Getenv("TODO_DB"); envDSN != "" {
		dsn = envDSN
	}
	db, err := sqlx.Connect("postgres", dsn)
	assert.NoError(t, err)
	return db
}

// ── HTTP ──

func getURL(path string) string {
	port := Port
	envPort := os.Getenv("TODO_PORT")
	if len(envPort) > 0 {
		if eport, err := strconv.ParseInt(envPort, 10, 32); err == nil {
			port = int(eport)
		}
	}
	path = strings.TrimPrefix(strings.ReplaceAll(path, `\`, `/`), `../web/`)
	return fmt.Sprintf("http://localhost:%d/%s", port, path)
}

func getBody(path string) ([]byte, error) {
	resp, err := http.Get(getURL(path))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func requestJSON(apipath string, values map[string]any, method string) ([]byte, error) {
	var (
		data []byte
		err  error
	)
	if len(values) > 0 {
		data, err = json.Marshal(values)
		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(method, getURL(apipath), bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	if len(Token) > 0 {
		jar, err := cookiejar.New(nil)
		if err != nil {
			return nil, err
		}
		jar.SetCookies(req.URL, []*http.Cookie{{Name: "token", Value: Token}})
		client.Jar = jar
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func postJSON(apipath string, values map[string]any, method string) (map[string]any, error) {
	var m map[string]any
	body, err := requestJSON(apipath, values, method)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(body, &m)
	return m, err
}

// ── Tasks ──

type task struct {
	date    string
	title   string
	comment string
	repeat  string
}

func addTask(t *testing.T, task task) string {
	ret, err := postJSON("api/task", map[string]any{
		"date":    task.date,
		"title":   task.title,
		"comment": task.comment,
		"repeat":  task.repeat,
	}, http.MethodPost)
	assert.NoError(t, err)
	assert.NotNil(t, ret["id"])
	id := fmt.Sprint(ret["id"])
	assert.NotEmpty(t, id)
	return id
}

// ── Static ──

func walkDir(path string, f func(fname string) error) error {
	dirs, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	for _, v := range dirs {
		fname := filepath.Join(path, v.Name())
		if v.IsDir() {
			if err = walkDir(fname, f); err != nil {
				return err
			}
			continue
		}
		if err = f(fname); err != nil {
			return err
		}
	}
	return nil
}

// Signin
func TestMain(m *testing.M) {
	_ = godotenv.Load("../.env")
	token, err := login()
	if err != nil {
		fmt.Fprintf(os.Stderr, "login failed: %v\n", err)
		os.Exit(1)
	}
	Token = token
	os.Exit(m.Run())
}

func login() (string, error) {
	password := os.Getenv("TODO_PASSWORD")
	if password == "" {
		return "", fmt.Errorf("TODO_PASSWORD is not set")
	}
	body, err := requestJSON("api/signin", map[string]any{
		"password": password,
	}, http.MethodPost)
	if err != nil {
		return "", nil
	}

	var m map[string]string
	if err := json.Unmarshal(body, &m); err != nil {
		return "", nil
	}
	if m["error"] != "" {
		return "", fmt.Errorf("%s", m["error"])
	}
	if m["token"] == "" {
		return "", fmt.Errorf("empty token")
	}
	return m["token"], nil
}
