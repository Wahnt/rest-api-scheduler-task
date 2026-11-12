package tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
)

func count(db *sqlx.DB) (int, error) {
	var c int
	return c, db.Get(&c, `SELECT count(id) FROM scheduler`)
}

// TestDB

func TestDB(t *testing.T) {
	db := openDB(t)
	defer db.Close()

	before, err := count(db)
	assert.NoError(t, err)

	today := time.Now().Format(`20060102`)

	var id int64

	err = db.QueryRow(`INSERT INTO scheduler (date, title, comment, repeat)
		VALUES ($1, 'Todo', 'Comment', '') RETURNING id`, today).Scan(&id)
	assert.NoError(t, err)

	var task Task

	err = db.Get(&task, `SELECT & FROM scheduler WHERE id=$1`, id)
	assert.NoError(t, err)
	assert.Equal(t, id, task.ID)
	assert.Equal(t, `Todo`, task.Title)
	assert.Equal(t, `Comment`, task.Comment)

	_, err = db.Exec(`DELETE FROM scheduler WHERE id=$1`, id)
	assert.NoError(t, err)

	after, err := count(db)
	assert.NoError(t, err)
	assert.Equal(t, before, after)
}

// TestAddTask
func TestAddTask(t *testing.T) {
	db := openDB(t)
	defer db.Close()

	tbl := []task{
		{"20240129", "", "", ""},
		{"20240192", "Qwerty", "", ""},
		{"28.01.2024", "Title", "", ""},
		{"20240112", "Title", "", "Whooo"},
	}

	for _, v := range tbl {
		m, err := postJSON("api/task", map[string]any{
			"date": v.date, "title": v.title, "comment": v.comment, "repeat": v.repeat,
		}, http.MethodPost)
		assert.NoError(t, err)

		e, ok := m["error"]
		assert.False(t, !ok || len(fmt.Sprint(e)) == 0,
			"Expected error for task %v", v)
	}

	now := time.Now()

	check := func() {
		for _, v := range tbl {
			today := v.date == "today"
			if today {
				v.date = now.Format(`20060102`)
			}
			m, err := postJSON("api/task", map[string]any{
				"date": v.date, "title": v.title, "comment": v.comment, "repeat": v.repeat,
			}, http.MethodPost)
			assert.NoError(t, err)

			e, ok := m["error"]
			if ok && len(fmt.Sprint(e)) > 0 {
				t.Errorf("Unexpected error %v for task %v", e, v)
				continue
			}

			var task Task
			mid, ok := m["id"]
			if !ok {
				t.Errorf("id not returned for task %v", v)
				continue
			}

			id := fmt.Sprint(mid)

			err = db.Get(&task, `SELECT * FROM scheduler WHERE id=$1`, id)
			assert.NoError(t, err)
			assert.Equal(t, id, strconv.FormatInt(task.ID, 10))
			assert.Equal(t, v.title, task.Title)
			assert.Equal(t, v.comment, task.Comment)
			assert.Equal(t, v.repeat, task.Repeat)
			if task.Date < now.Format(`20060102`) {
				t.Errorf("The date cannot be earlier than today %v", v)
				continue
			}
			if today && task.Date != now.Format(`20060102`) {
				t.Errorf("Must be today date %v", v)
			}
		}
	}

	tbl = []task{
		{"", "Title", "", ""},
		{"20231220", "Сделать что-нибудь", "Хорошо отдохнуть", ""},
		{"20240108", "Уроки", "", "d 10"},
		{"20240102", "Отдых в Сочи", "на лыжах", "у"},
		{"today", "Шнитнес", "", "d 1"},
		{"today", "Шмитнес как будто бы", "", ""},
	}
	check()
	if FullNextDate {
		tbl = []task{{"20240129", "Сходить в магазин", "", "w 1,2,3"}}
		check()
	}
}

// TestTasks

func getTasks(t *testing.T, search string) []map[string]string {
	url := "api/tasks"
	if Search {
		url += "?search" + search
	}
	body, err := requestJSON(url, nil, http.MethodGet)
	assert.NoError(t, err)

	var m map[string][]map[string]string

	err = json.Unmarshal(body, &m)
	assert.NoError(t, err)
	return m["tasks"]
}

func TestTasks(t *testing.T) {
	db := openDB(t)
	defer db.Close()

	now := time.Now()

	_, err := db.Exec("DELETE FROM scheduler")
	assert.NoError(t, err)

	tasks := getTasks(t, "")
	assert.NotNil(t, tasks)
	assert.Empty(t, tasks)

	addTask(t, task{date: now.Format(`20060102`), title: "takes tests", comment: "for postgres"})
	now = now.AddDate(0, 0, 1)
	date := now.Format(`20060102`)
	addTask(t, task{date: date, title: "Сходить в бассейн"})
	addTask(t, task{date: date, title: "Оплатить коммуналку", repeat: "d 30"})
	tasks = getTasks(t, "")
	assert.Equal(t, 3, len(tasks))

	now = now.AddDate(0, 0, 2)
	date = now.Format(`20060102`)
	addTask(t, task{date: date, title: "Поплавать", comment: "Бассейн с тренером", repeat: "d 7"})
	addTask(t, task{date: date, title: "Позвонить в УК", comment: "Разобраться горчей водой"})
	addTask(t, task{date: date, title: "Встретиться с Васей", comment: "в 18:00"})
	tasks = getTasks(t, "")
	assert.Equal(t, 6, len(tasks))

	if !Search {
		return
	}
	tasks = getTasks(t, "УК")
	assert.Equal(t, 1, len(tasks))
	tasks = getTasks(t, now.Format(`02.01.2006`))
	assert.Equal(t, 3, len(tasks))
}

// TestTask GET/PUT
func TestTask(t *testing.T) {
	db := openDB(t)
	defer db.Close()

	now := time.Now()
	task := task{date: now.Format(`20060102`), title: "Созвон в 16:00", comment: "Обсуждение планов", repeat: "d 5"}
	todo := addTask(t, task)

	body, err := requestJSON("api/task", nil, http.MethodGet)
	assert.NoError(t, err)
	var m map[string]string
	err = json.Unmarshal(body, &m)
	assert.NoError(t, err)

	e, ok := m["error"]
	assert.False(t, !ok || len(fmt.Sprint(e)) == 0, "Ожидается ошибка для вызова /api/task")

	body, err = requestJSON("api/task?id="+todo, nil, http.MethodGet)
	assert.NoError(t, err)
	err = json.Unmarshal(body, &m)
	assert.NoError(t, err)

	assert.Equal(t, todo, m["id"])
	assert.Equal(t, task.date, m["date"])
	assert.Equal(t, task.title, m["title"])
	assert.Equal(t, task.comment, m["comment"])
	assert.Equal(t, task.repeat, m["repeat"])
}

type fulltask struct {
	id string
	task
}

func TestEditTask(t *testing.T) {
	db := openDB(t)
	defer db.Close()

	now := time.Now()
	tsk := task{date: now.Format(`20060102`), title: "Заказать пиццу", comment: "в 17:00"}
	id := addTask(t, tsk)

	tbl := []fulltask{
		{"", task{"20240129", "Тест", "", ""}},
		{"abc", task{"20240129", "Тест", "", ""}},
		{"7645346343", task{"20240129", "Тест", "", ""}},
		{id, task{"20240129", "", "", ""}},
		{id, task{"20240192", "Qwerty", "", ""}},
		{id, task{"28.01.2024", "Заголовок", "", ""}},
		{id, task{"20240212", "Заголовок", "", "ooops"}},
	}
	for _, v := range tbl {
		m, err := postJSON("api/task", map[string]any{
			"id": v.id, "date": v.date, "title": v.title, "comment": v.comment, "repeat": v.repeat,
		}, http.MethodPut)
		assert.NoError(t, err)

		var errVal string
		if e, ok := m["error"]; ok {
			errVal = fmt.Sprint(e)
		}
		assert.NotEqual(t, len(errVal), 0, "Expected error for value %v", v)
	}

	updateTask := func(newVals map[string]any) {
		mupd, err := postJSON("api/task", newVals, http.MethodPut)
		assert.NoError(t, err)
		e, ok := mupd["error"]
		assert.False(t, ok && fmt.Sprint(e) != "")

		var task Task
		err = db.Get(&task, `SELECT * FROM scheduler WHERE id=$1`, id)
		assert.NoError(t, err)

		assert.Equal(t, id, strconv.FormatInt(task.ID, 10))
		assert.Equal(t, newVals["title"], task.Title)
		if _, is := newVals["comment"]; !is {
			newVals["comment"] = ""
		}
		if _, is := newVals["repeat"]; !is {
			newVals["repeat"] = ""
		}
		assert.Equal(t, newVals["comment"], task.Comment)
		assert.Equal(t, newVals["repeat"], task.Repeat)
		if task.Date < time.Now().Format(`20060102`) {
			t.Errorf("The date cannot be earlier than today")
		}
	}

	updateTask(map[string]any{
		"id": id, "date": now.Format(`20060102`), "title": "Заказать хинкали",
		"comment": "в 18:00", "repeat": "d 7",
	})
}

// TestDone + TestDelTask
func notFoundTask(t *testing.T, id string) {
	body, err := requestJSON("api/task?id="+id, nil, http.MethodGet)
	assert.NoError(t, err)
	var m map[string]any
	err = json.Unmarshal(body, &m)
	assert.NoError(t, err)
	_, ok := m["error"]
	assert.True(t, ok)
}

func TestDone(t *testing.T) {
	db := openDB(t)
	defer db.Close()

	now := time.Now()
	id := addTask(t, task{date: now.Format(`20060102`), title: "Свести баланс"})

	ret, err := postJSON("api/task/done?id="+id, nil, http.MethodPost)
	assert.NoError(t, err)
	assert.Empty(t, ret)
	notFoundTask(t, id)

	id = addTask(t, task{title: "Проверить работу /api/task/done", repeat: "d 3"})
	for i := 0; i < 3; i++ {
		ret, err := postJSON("api/task/done?id="+id, nil, http.MethodPost)
		assert.NoError(t, err)
		assert.Empty(t, ret)

		var task Task
		err = db.Get(&task, `SELECT * FROM scheduler WHERE id=$1`, id)
		assert.NoError(t, err)
		now = now.AddDate(0, 0, 3)
		assert.Equal(t, now.Format(`20060102`), task.Date)
	}
}

func TestDelTask(t *testing.T) {
	db := openDB(t)
	defer db.Close()

	id := addTask(t, task{title: "Временная задача", repeat: "d 3"})
	ret, err := postJSON("api/task?id="+id, nil, http.MethodDelete)
	assert.NoError(t, err)
	_ = ret
	notFoundTask(t, id)

	ret, err = postJSON("api/task", nil, http.MethodDelete)
	assert.NoError(t, err)
	assert.NotEmpty(t, ret)
	ret, err = postJSON("api/task?id=wjhgese", nil, http.MethodDelete)
	assert.NoError(t, err)
	assert.NotEmpty(t, ret)
}
